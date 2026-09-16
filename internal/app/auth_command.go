package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	authpkg "github.com/gokuai/yunku-cli/internal/auth"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

func buildAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "auth",
		Short:             "够快云库认证",
		Long:              "够快云库 API 认证管理，支持用户名密码登录、Token 刷新等。",
		Args:              cobra.NoArgs,
		TraverseChildren:  true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newAuthLoginCommand(),
		newAuthRefreshCommand(),
		newAuthStatusCommand(),
		newAuthLogoutCommand(),
	)
	return cmd
}

func newAuthLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "用户名密码登录",
		Example: `  ykc auth login --username user@example.com --password yourpassword
  ykc auth login -u user@example.com -p yourpassword`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")

			if username == "" || password == "" {
				return apperrors.NewValidation("用户名和密码不能为空")
			}

			var authClient *gokuai.AuthClient
			var userClientID, userClientSecret string

			// 优先使用 GOKUAI_USER_CLIENT_ID/SECRET 直接登录
			if id := os.Getenv("GOKUAI_USER_CLIENT_ID"); id != "" {
				secret := os.Getenv("GOKUAI_USER_CLIENT_SECRET")
				if secret != "" {
					slog.Info("using GOKUAI_USER_CLIENT_ID for login")
					authClient = gokuai.NewAuthClient(id, secret)
					userClientID = id
					userClientSecret = secret
				}
			}

			// 如果没有 user client id，走设备应用授权流程
			if authClient == nil {
				// 获取设备凭证
				clientID := getEnvOrFlag(cmd, "client-id", "GOKUAI_CLIENT_ID")
				clientSecret := getEnvOrFlag(cmd, "client-secret", "GOKUAI_CLIENT_SECRET")

				// 使用设备凭证获取用户授权凭证
				client := gokuai.NewClient(clientID, clientSecret)

				oauthResp, err := client.GetClientOAuth()
				if err != nil {
					slog.Error("gokuai get client oauth failed", "error", err)
					return apperrors.NewAuth(fmt.Sprintf("获取用户授权失败: %v", err), apperrors.WithCause(err))
				}

				slog.Info("gokuai get client oauth success",
					"user_client_id", oauthResp.ClientID,
				)

				userClientID = oauthResp.ClientID
				userClientSecret = oauthResp.ClientSecret
				authClient = gokuai.NewAuthClient(oauthResp.ClientID, oauthResp.ClientSecret)
			}

			// 记录登录请求参数（不包含密码）
			slog.Info("gokuai auth login request",
				"username", username,
				"client_id", userClientID,
				"grant_type", "password",
			)

			resp, err := authClient.LoginByPassword(username, password)
			if err != nil {
				slog.Error("gokuai auth login failed", "error", err)
				return apperrors.NewAuth(fmt.Sprintf("登录失败: %v", err), apperrors.WithCause(err))
			}

			// 记录登录成功（不包含敏感 token）
			slog.Info("gokuai auth login success",
				"expires_in", resp.ExpiresIn,
			)

			// 保存 token 和用户授权凭证
			if err := saveGokuaiToken(resp.AccessToken, resp.RefreshToken, resp.ExpiresIn); err != nil {
				return fmt.Errorf("保存 token 失败: %v", err)
			}
			if err := saveGokuaiUserCredentials(userClientID, userClientSecret); err != nil {
				return fmt.Errorf("保存用户凭证失败: %v", err)
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().StringP("username", "u", "", "用户名")
	cmd.Flags().StringP("password", "p", "", "密码")
	cmd.Flags().String("client-id", "", "设备应用 Client ID (可使用环境变量 GOKUAI_CLIENT_ID)")
	cmd.Flags().String("client-secret", "", "设备应用 Client Secret (可使用环境变量 GOKUAI_CLIENT_SECRET)")
	if os.Getenv("GOKUAI_USER_CLIENT_ID") != "" {
		_ = cmd.Flags().MarkHidden("client-id")
		_ = cmd.Flags().MarkHidden("client-secret")
	}
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	return cmd
}

func newAuthRefreshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "refresh",
		Short:   "刷新 Access Token",
		Example: `  ykc auth refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			refreshToken, _ := cmd.Flags().GetString("refresh-token")
			if refreshToken == "" {
				// 尝试从本地读取
				if saved, err := loadGokuaiRefreshToken(); err == nil && saved != "" {
					refreshToken = saved
				}
			}

			if refreshToken == "" {
				return apperrors.NewAuth("refresh token 为空，请先登录或通过 --refresh-token 传入")
			}

			// 从缓存获取用户凭证
			userClientID, userClientSecret, err := loadGokuaiUserCredentials()
			if err != nil || userClientID == "" || userClientSecret == "" {
				return apperrors.NewAuth("用户凭证未找到，请先登录")
			}

			client := gokuai.NewAuthClient(userClientID, userClientSecret)
			resp, err := client.RefreshToken(refreshToken)
			if err != nil {
				return apperrors.NewAuth(fmt.Sprintf("刷新 token 失败: %v", err), apperrors.WithCause(err))
			}

			// 保存新 token（合并更新：服务端未返回新 refresh_token 时保留原有值）
			if err := saveRefreshedGokuaiToken(resp); err != nil {
				return fmt.Errorf("保存 token 失败: %v", err)
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("refresh-token", "", "Refresh Token (可选，默认从本地登录存储读取)")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "查看认证状态",
		Example: `  ykc auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			accessToken, _ := cmd.Flags().GetString("token")
			if accessToken == "" {
				// 尝试从本地读取
				if saved, err := loadGokuaiAccessToken(); err == nil && saved != "" {
					accessToken = saved
				}
			}

			if accessToken == "" {
				return outputJSON(cmd, map[string]interface{}{
					"authenticated": false,
					"message":       "未登录，请运行 ykc auth login 登录",
				})
			}

			// 通过 account info 接口验证 token 有效性
			secret, err := getSecret()
			if err != nil {
				return outputJSON(cmd, map[string]interface{}{
					"authenticated": false,
					"message":       "用户凭证未找到，请先登录",
				})
			}

			client := gokuai.NewUserClient(accessToken)
			accountInfo, err := client.GetAccountInfo(secret)
			if err != nil {
				return outputJSON(cmd, map[string]interface{}{
					"authenticated": false,
					"message":       "Token 无效或已过期",
				})
			}

			userClientID, _ := getUserClientID()
			resp := map[string]interface{}{
				"authenticated": true,
				"member_id":     accountInfo.MemberID,
				"member_name":   accountInfo.MemberName,
				"member_email":  accountInfo.MemberEmail,
				"member_account":        accountInfo.MemberAccount,
				"client_id":     userClientID,
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("token", "", "Access Token (默认读取本地登录凭证)")
	return cmd
}

func newAuthLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "清除认证信息",
		Example: `  ykc auth logout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := defaultConfigDir()
			legacyTokenFile := filepath.Join(configDir, gokuaiTokenFile)
			credFile := filepath.Join(configDir, gokuaiUserCredFile)

			// 清除本地存储的 token 和用户凭证
			_ = authpkg.DeleteTokenData(configDir)
			_ = os.Remove(legacyTokenFile)
			_ = os.Remove(credFile)

			resp := map[string]string{
				"success": "true",
				"message": "已清除认证信息",
			}
			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

// === Helper Functions ===

func getEnvOrFlag(cmd *cobra.Command, flagName, envName string) string {
	// 先检查命令行参数
	if val, _ := cmd.Flags().GetString(flagName); val != "" {
		return val
	}
	// 再检查环境变量（.env 已在启动时加载到 os.Environ）
	return os.Getenv(envName)
}

// gokuaiTokenFile is the legacy plaintext token file used before token
// storage was switched to encrypted keychain storage (authpkg.TokenData).
// It is only read once, to transparently migrate existing sessions.
const gokuaiTokenFile = "gokuai_token.json"

type gokuaiTokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func saveGokuaiToken(accessToken, refreshToken string, expiresIn int) error {
	return saveGokuaiTokenWithRefreshExpiry(accessToken, refreshToken, expiresIn, 0)
}

// saveGokuaiTokenWithRefreshExpiry persists the token set, recording both
// access-token expiry and refresh-token expiry. refreshExpiresIn <= 0 means
// the refresh expiry is left unset (treated as non-expiring by checks).
func saveGokuaiTokenWithRefreshExpiry(accessToken, refreshToken string, expiresIn, refreshExpiresIn int) error {
	configDir := defaultConfigDir()
	data := &authpkg.TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		Source:       "gokuai",
	}
	if refreshExpiresIn > 0 {
		data.RefreshExpAt = time.Now().Add(time.Duration(refreshExpiresIn) * time.Second)
	}
	return authpkg.SaveTokenData(configDir, data)
}

// saveRefreshedGokuaiToken merges a refresh response into the existing stored
// token data: the access token is always updated; the refresh token is only
// replaced when the server returns a non-empty new one (otherwise the previous
// refresh token is retained, as OAuth servers commonly do not rotate it).
func saveRefreshedGokuaiToken(resp *gokuai.TokenResponse) error {
	configDir := defaultConfigDir()

	data, _ := authpkg.LoadTokenData(configDir)
	if data == nil {
		data = &authpkg.TokenData{}
	}
	data.Source = "gokuai"
	data.AccessToken = resp.AccessToken
	data.ExpiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	if resp.RefreshToken != "" {
		data.RefreshToken = resp.RefreshToken
		if resp.RefreshExpiresIn > 0 {
			data.RefreshExpAt = time.Now().Add(time.Duration(resp.RefreshExpiresIn) * time.Second)
		}
	}

	return authpkg.SaveTokenData(configDir, data)
}

// loadGokuaiTokenData loads the persisted GoKuai token from encrypted
// storage, migrating from the legacy plaintext gokuai_token.json the first
// time it is called after upgrading (previously tokens were written with
// os.WriteFile in plain JSON).
func loadGokuaiTokenData() (*authpkg.TokenData, error) {
	configDir := defaultConfigDir()
	data, err := authpkg.LoadTokenData(configDir)
	if err == nil && data != nil && (data.AccessToken != "" || data.RefreshToken != "") {
		return data, nil
	}

	legacyFile := filepath.Join(configDir, gokuaiTokenFile)
	legacyBytes, legacyErr := os.ReadFile(legacyFile)
	if legacyErr != nil {
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	var legacy gokuaiTokenData
	if jsonErr := json.Unmarshal(legacyBytes, &legacy); jsonErr != nil {
		return nil, jsonErr
	}
	migrated := &authpkg.TokenData{
		AccessToken:  legacy.AccessToken,
		RefreshToken: legacy.RefreshToken,
		ExpiresAt:    legacy.ExpiresAt,
		Source:       "gokuai",
	}
	if saveErr := authpkg.SaveTokenData(configDir, migrated); saveErr == nil {
		_ = os.Remove(legacyFile)
	}
	return migrated, nil
}

func loadGokuaiAccessToken() (string, error) {
	data, err := loadGokuaiTokenData()
	if err != nil {
		return "", err
	}
	if !data.IsAccessTokenValid() {
		return "", apperrors.NewAuth("token 已过期")
	}
	return data.AccessToken, nil
}

func loadGokuaiRefreshToken() (string, error) {
	data, err := loadGokuaiTokenData()
	if err != nil {
		return "", err
	}
	return data.RefreshToken, nil
}

// === 用户凭证存储 ===

const gokuaiUserCredFile = "gokuai_user_cred.json"

type gokuaiUserCredData struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func saveGokuaiUserCredentials(clientID, clientSecret string) error {
	configDir := defaultConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	data := gokuaiUserCredData{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	credFile := filepath.Join(configDir, gokuaiUserCredFile)
	return os.WriteFile(credFile, jsonData, 0o600)
}

func loadGokuaiUserCredentials() (string, string, error) {
	configDir := defaultConfigDir()
	credFile := filepath.Join(configDir, gokuaiUserCredFile)

	data, err := os.ReadFile(credFile)
	if err != nil {
		return "", "", err
	}

	var credData gokuaiUserCredData
	if err := json.Unmarshal(data, &credData); err != nil {
		return "", "", err
	}

	return credData.ClientID, credData.ClientSecret, nil
}

// GetGokuaiLogPath 返回够快云库日志文件路径
func GetGokuaiLogPath() string {
	return filepath.Join(defaultConfigDir(), "logs", "ykc.log")
}
