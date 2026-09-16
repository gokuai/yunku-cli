package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/internal/output"
	"github.com/gokuai/yunku-cli/pkg/configmeta"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

func init() {
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_CLIENT_ID",
		Category:    configmeta.CategoryAuth,
		Description: "够快企业管理 API client_id (ent 命令需要，与 GOKUAI_CLIENT_SECRET 配合做 HMAC-SHA1 签名认证)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_CLIENT_SECRET",
		Category:    configmeta.CategoryAuth,
		Description: "够快企业管理 API client_secret (ent 命令需要，与 GOKUAI_CLIENT_ID 配合做 HMAC-SHA1 签名认证)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_ORG_CLIENT_ID",
		Category:    configmeta.CategoryAuth,
		Description: "库授权 client_id (ent file 库文件操作命令需要，也可用 --org-client-id 传入)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_ORG_CLIENT_SECRET",
		Category:    configmeta.CategoryAuth,
		Description: "库授权 client_secret (ent file 库文件操作命令需要，也可用 --org-client-secret 传入)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_BEARER_TOKEN",
		Category:    configmeta.CategoryAuth,
		Description: "Bearer Token 认证，等同全局参数 --bearer-token (flag 优先)，设置后用户 API 跳过签名直接使用该 token",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_USER_CLIENT_ID",
		Category:    configmeta.CategoryAuth,
		Description: "用户授权 client_id (用户 API 签名用，设置后 ykc auth login 可跳过 ent oauth get 换取步骤)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:        "GOKUAI_USER_CLIENT_SECRET",
		Category:    configmeta.CategoryAuth,
		Description: "用户授权 client_secret (用户 API 签名用，与 GOKUAI_USER_CLIENT_ID 配合)",
		Sensitive:   true,
	})
	configmeta.Register(configmeta.ConfigItem{
		Name:         "GOKUAI_API_HOST",
		Category:     configmeta.CategoryNetwork,
		Description:  "够快 API 主机地址 (覆盖默认值，用于私有化部署或测试环境)",
		DefaultValue: "yk3.gokuai.com",
	})
}

// === 库授权缓存 ===

const gokuaiOrgBindCacheFile = "gokuai_org_bind_cache.json"

// lastBindCacheKey 记录最近一次成功 bind 的库授权，后续命令
// 不带 --org-id/--mount-id 时自动复用（有效期内）。
const lastBindCacheKey = "last_bind"

type orgBindCacheEntry struct {
	OrgClientID     string    `json:"org_client_id"`
	OrgClientSecret string    `json:"org_client_secret"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type orgBindCache map[string]orgBindCacheEntry

func orgBindCacheKey(clientID, orgID string, mountID int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", clientID, orgID, mountID)))
	return fmt.Sprintf("%x", h[:])
}

func loadOrgBindCache() (orgBindCache, error) {
	configDir := defaultConfigDir()
	cacheFile := filepath.Join(configDir, gokuaiOrgBindCacheFile)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return orgBindCache{}, nil
	}

	var cache orgBindCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return orgBindCache{}, nil
	}

	return cache, nil
}

func saveOrgBindCache(cache orgBindCache) error {
	configDir := defaultConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	cacheFile := filepath.Join(configDir, gokuaiOrgBindCacheFile)
	return os.WriteFile(cacheFile, jsonData, 0o600)
}

// === Helper functions ===

// runtimeBearerToken is set from the --bearer-token global flag via SetBearerToken.
// When non-empty, getUserClient returns a bearer-token-mode client that skips signing.
var runtimeBearerToken string

// SetBearerToken stores the bearer token from the CLI flag for use by gokuai API clients.
func SetBearerToken(token string) {
	runtimeBearerToken = token
}

// runtimeClientID / runtimeClientSecret are set from the ent-scoped
// --client-id / --client-secret persistent flags via SetEntClientCredentials.
// When non-empty they take precedence over the GOKUAI_CLIENT_ID / GOKUAI_CLIENT_SECRET
// environment variables for enterprise API clients.
var runtimeClientID, runtimeClientSecret string

// SetEntClientCredentials stores enterprise client credentials from CLI flags
// for use by gokuai enterprise API clients.
func SetEntClientCredentials(clientID, clientSecret string) {
	runtimeClientID = clientID
	runtimeClientSecret = clientSecret
}

// resolveEntCredentials returns the enterprise client_id/client_secret to use,
// with precedence: CLI flag > environment variable.
func resolveEntCredentials() (string, string) {
	clientID := runtimeClientID
	clientSecret := runtimeClientSecret
	if clientID == "" {
		clientID = os.Getenv("GOKUAI_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("GOKUAI_CLIENT_SECRET")
	}
	return clientID, clientSecret
}

func getClient() *gokuai.Client {
	clientID, clientSecret := resolveEntCredentials()
	return gokuai.NewClient(clientID, clientSecret)
}

func getUserClient() (*gokuai.UserClient, error) {
	if runtimeBearerToken != "" {
		return gokuai.NewBearerTokenUserClient(runtimeBearerToken), nil
	}

	accessToken, err := ensureGokuaiAccessToken()
	if err != nil || accessToken == "" {
		if err == nil {
			err = apperrors.NewAuth("请先登录，运行 'ykc auth login'")
		}
		return nil, err
	}

	client := gokuai.NewUserClient(accessToken)
	// 服务端返回 40102(token 失效)时自动用 refresh token 刷新一次并重放请求
	client.TokenRefresher = newTokenRefresher()
	return client, nil
}

// getSecret 获取签名用的 secret
// 优先级: GOKUAI_USER_CLIENT_SECRET > 本地缓存的用户凭证
// 注意: 用户 API (/m-api/) 必须使用用户级别的 client_secret (来自 GetClientOAuth)，
// 而非企业级的 GOKUAI_CLIENT_SECRET，否则会导致签名错误
func getSecret() (string, error) {
	// 优先使用用户级别的 secret
	if secret := os.Getenv("GOKUAI_USER_CLIENT_SECRET"); secret != "" {
		return secret, nil
	}
	_, userSecret, err := loadGokuaiUserCredentials()
	if err != nil || userSecret == "" {
		return "", apperrors.NewAuth("请设置环境变量 GOKUAI_USER_CLIENT_SECRET 或先登录")
	}
	return userSecret, nil
}

// getUserClientID 获取用户级别的 client_id
// 优先级: GOKUAI_USER_CLIENT_ID > 本地缓存的用户凭证
func getUserClientID() (string, error) {
	if id := os.Getenv("GOKUAI_USER_CLIENT_ID"); id != "" {
		return id, nil
	}
	userClientID, _, err := loadGokuaiUserCredentials()
	if err != nil || userClientID == "" {
		return "", apperrors.NewAuth("请设置环境变量 GOKUAI_USER_CLIENT_ID 或先登录")
	}
	return userClientID, nil
}

func getOrgClient(cmd *cobra.Command) (*gokuai.OrgClient, error) {
	// 优先通过 --org-id/--mount-id 自动 bind 获取库授权
	orgID, _ := cmd.Flags().GetString("org-id")
	mountID, _ := cmd.Flags().GetInt("mount-id")
	title, _ := cmd.Flags().GetString("title")

	if orgID != "" || mountID > 0 {
		clientID, clientSecret := resolveEntCredentials()
		if clientID == "" || clientSecret == "" {
			return nil, apperrors.NewAuth("自动获取库授权需要企业凭证，请通过 --client-id/--client-secret 或环境变量 GOKUAI_CLIENT_ID/GOKUAI_CLIENT_SECRET 设置")
		}

		// 尝试从缓存读取
		cacheKey := orgBindCacheKey(clientID, orgID, mountID)
		cache, _ := loadOrgBindCache()
		if entry, ok := cache[cacheKey]; ok && time.Now().Before(entry.ExpiresAt) {
			// 刷新 last_bind，使后续无参调用复用本次指定的库
			cache[lastBindCacheKey] = entry
			_ = saveOrgBindCache(cache)
			return gokuai.NewOrgClient(entry.OrgClientID, entry.OrgClientSecret), nil
		}

		// 缓存未命中，调用 bind 接口
		client := gokuai.NewClient(clientID, clientSecret)
		params := map[string]string{}
		if orgID != "" {
			params["org_id"] = orgID
		}
		if mountID > 0 {
			params["mount_id"] = fmt.Sprintf("%d", mountID)
		}
		if title != "" {
			params["title"] = title
		}
		resp, err := client.BindOrg(params)
		if err != nil {
			return nil, fmt.Errorf("自动获取库授权失败: %w", err)
		}

		// 写入缓存，有效期 1 小时；同时记录为最近一次绑定
		entry := orgBindCacheEntry{
			OrgClientID:     resp.OrgClientID,
			OrgClientSecret: resp.OrgClientSecret,
			ExpiresAt:       time.Now().Add(1 * time.Hour),
		}
		cache[cacheKey] = entry
		cache[lastBindCacheKey] = entry
		_ = saveOrgBindCache(cache)

		return gokuai.NewOrgClient(resp.OrgClientID, resp.OrgClientSecret), nil
	}

	// 直接配置: --org-client-id/--org-client-secret 或对应环境变量
	orgClientID, _ := cmd.Flags().GetString("org-client-id")
	orgClientSecret, _ := cmd.Flags().GetString("org-client-secret")

	if orgClientID == "" {
		orgClientID = os.Getenv("GOKUAI_ORG_CLIENT_ID")
	}
	if orgClientSecret == "" {
		orgClientSecret = os.Getenv("GOKUAI_ORG_CLIENT_SECRET")
	}

	if orgClientID != "" && orgClientSecret != "" {
		return gokuai.NewOrgClient(orgClientID, orgClientSecret), nil
	}

	// 未提供任何参数时，复用最近一次 bind 的库授权（有效期内）
	cache, _ := loadOrgBindCache()
	if entry, ok := cache[lastBindCacheKey]; ok && time.Now().Before(entry.ExpiresAt) {
		return gokuai.NewOrgClient(entry.OrgClientID, entry.OrgClientSecret), nil
	}

	return nil, apperrors.NewAuth("请提供库授权参数: 通过 --org-id/--mount-id 自动获取库授权(绑定后 1 小时内可省略), 或设置 --org-client-id/--org-client-secret (环境变量 GOKUAI_ORG_CLIENT_ID/GOKUAI_ORG_CLIENT_SECRET)")
}

func getOpenClient() *gokuai.Client {
	clientID, clientSecret := resolveEntCredentials()
	return gokuai.NewClient(clientID, clientSecret)
}

func outputJSON(cmd *cobra.Command, v interface{}) error {
	// Legacy --json compact flag (used by doctor/config commands)
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}

	return output.WriteCommandPayload(cmd, v, output.FormatJSON)
}
