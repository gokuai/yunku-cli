package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/spf13/cobra"
)

// newAccountCommand creates the account management command group
func newAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "账户管理",
		Long: `账户管理，包括获取账户信息、设备管理、密码修改等。
需要先登录获取 access_token: ykc auth login`,
	}

	cmd.AddCommand(
		newAccountMountCommand(),
		newAccountInfoCommand(),
		newAccountFindPasswordCommand(),

		newAccountEntInfoCommand(),
		newAccountDeviceListCommand(),
		newAccountDeviceToggleCommand(),
		newAccountDeviceDeleteCommand(),
		newAccountDeviceDisableNewCommand(),
		newAccountPasswordChangeCommand(),
		newAccountServersCommand(),
		newAccountSettingsCommand(),
		newAccountMailResendCommand(),
		newAccountMailBindCommand(),
		newAccountUpdateEntCommand(),
		newAccountAvatarUploadCommand(),
		newAccountSoftwareCommand(),
	)

	return cmd
}

func newAccountInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "获取账户信息",
		Example: `  ykc account info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetAccountInfo(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newAccountDeviceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "devices",
		Short:   "获取设备列表",
		Example: `  ykc account devices`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetDeviceList(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newAccountDeviceToggleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle-device",
		Short: "启用/禁用设备",
		Example: `  ykc account toggle-device --device-id xxx --state 1
  ykc account toggle-device --device-id xxx --state 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID, _ := cmd.Flags().GetString("device-id")
			state, _ := cmd.Flags().GetString("state")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.ToggleDevice(deviceID, state, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("device-id", "", "设备ID (必需)")
	cmd.Flags().String("state", "", "状态: 1启用/0禁用 (必需)")
	cmd.MarkFlagRequired("device-id")
	cmd.MarkFlagRequired("state")
	return cmd
}

func newAccountDeviceDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-device",
		Short:   "删除设备",
		Example: `  ykc account delete-device --device-id xxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID, _ := cmd.Flags().GetString("device-id")

			if deviceID == "" {
				return apperrors.NewValidation("device-id 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DeleteDevice(deviceID, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("device-id", "", "设备ID (必需)")
	cmd.MarkFlagRequired("device-id")
	return cmd
}

func newAccountDeviceDisableNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-new-device",
		Short: `启用/禁用"禁用新设备登录"功能`,
		Example: `  # 启用"禁用新设备登录"功能
  ykc account disable-new-device --state 1

  # 禁用"禁用新设备登录"功能
  ykc account disable-new-device --state 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			state, _ := cmd.Flags().GetString("state")

			if state != "0" && state != "1" {
				return apperrors.NewValidation("state 必须为 1(启用) 或 0(禁用)")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DisableNewDevice(state, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("state", "", `状态: 1(启用"禁用新设备"功能) / 0(禁用该功能) (必需)`)
	cmd.MarkFlagRequired("state")
	return cmd
}

func newAccountPasswordChangeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "change-password",
		Short:   "修改密码",
		Example: `  ykc account change-password --old-password xxx --new-password yyy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			oldPassword, _ := cmd.Flags().GetString("old-password")
			newPassword, _ := cmd.Flags().GetString("new-password")

			if oldPassword == "" || newPassword == "" {
				return apperrors.NewValidation("old-password 和 new-password 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			refreshToken, _ := cmd.Flags().GetString("refresh-token")
			if refreshToken == "" {
				if saved, err := loadGokuaiRefreshToken(); err == nil && saved != "" {
					refreshToken = saved
				}
			}
			if refreshToken == "" {
				return apperrors.NewAuth("refresh_token 为空，请先登录或通过 --refresh-token 传入")
			}

			if err := userClient.ChangePassword(oldPassword, newPassword, refreshToken, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("old-password", "", "旧密码 (必需)")
	cmd.Flags().String("new-password", "", "新密码 (必需)")
	cmd.Flags().String("refresh-token", "", "Refresh Token")
	cmd.MarkFlagRequired("old-password")
	cmd.MarkFlagRequired("new-password")
	return cmd
}

func newAccountServersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "servers",
		Short:   "获取服务器列表",
		Example: `  ykc account servers --server-type upload`,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverType, _ := cmd.Flags().GetString("server-type")
			storagePoint, _ := cmd.Flags().GetString("storage-point")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetServers(serverType, storagePoint, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("server-type", "", "服务器类型: upload/download")
	cmd.Flags().String("storage-point", "", "存储点")
	return cmd
}

func newAccountSettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   "获取设置信息",
		Example: `  ykc account settings`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetSettings(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newAccountMailResendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resend-mail",
		Short:   "重新发送验证邮件",
		Example: `  ykc account resend-mail --email user@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")

			if email == "" {
				return apperrors.NewValidation("email 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.ResendMail(email, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("email", "", "邮箱地址 (必需)")
	cmd.MarkFlagRequired("email")
	return cmd
}

func newAccountMailBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bind-mail",
		Short:   "绑定邮箱",
		Example: `  ykc account bind-mail --email user@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")

			if email == "" {
				return apperrors.NewValidation("email 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.BindMail(email, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("email", "", "邮箱地址 (必需)")
	cmd.MarkFlagRequired("email")
	return cmd
}

func newAccountAvatarUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload-avatar",
		Short: "上传头像",
		Example: `  ykc account upload-avatar --file /path/to/avatar.jpg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")

			if file == "" {
				return apperrors.NewValidation("file 不能为空")
			}

			fileBytes, err := os.ReadFile(file)
			if err != nil {
				return apperrors.NewValidation(fmt.Sprintf("读取文件失败: %v", err))
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.UploadAvatar(fileBytes, filepath.Base(file), "avatar", secret)
			if err != nil {
				return err
			}

			// 用上传返回的 uri 调用 set_info 设置头像
			setResp, err := userClient.SetAccountInfo(map[string]string{
				"avatar": resp.URI,
			}, nil, "", secret)
			if err != nil {
				return fmt.Errorf("上传成功但设置头像失败: %w", err)
			}

			return outputJSON(cmd, setResp)
		},
	}
	cmd.Flags().String("file", "", "头像文件路径")
	cmd.MarkFlagRequired("file")
	return cmd
}

func newAccountSoftwareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "software",
		Short:   "获取软件信息",
		Example: `  ykc account software`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetSoftware(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newAccountMountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mount",
		Short:   "获取文件库列表",
		Example: `  ykc account mount`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetMountList(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newAccountFindPasswordCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "find-password",
		Short:   "找回密码",
		Example: `  ykc account find-password --email user@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")

			if email == "" {
				return apperrors.NewValidation("email 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.FindPassword(email, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("email", "", "邮箱地址 (必需)")
	cmd.MarkFlagRequired("email")
	return cmd
}

func newAccountEntInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ent-info",
		Short:   "获取企业信息",
		Example: `  ykc account ent-info --ent-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")

			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetAccountEntInfo(entID, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}

// validateLogoFile 校验企业 Logo 文件格式与大小，并返回文件内容。
// 仅支持 JPG/GIF/PNG 格式，文件大小必须小于 500KB。
func validateLogoFile(path string) ([]byte, string, error) {
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".png":  true,
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !allowedExts[ext] {
		return nil, "", apperrors.NewValidation(fmt.Sprintf("logo 格式不支持 %s，仅支持 JPG、GIF、PNG", ext))
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, "", apperrors.NewValidation(fmt.Sprintf("读取 logo 文件失败: %v", err))
	}
	if info.Size() >= 500*1024 {
		return nil, "", apperrors.NewValidation("logo 文件大小必须小于 500KB")
	}

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, "", apperrors.NewValidation(fmt.Sprintf("读取 logo 文件失败: %v", err))
	}
	return fileBytes, filepath.Base(path), nil
}

func newAccountUpdateEntCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-ent",
		Short:   "更新企业信息",
		Example: `  ykc account update-ent --ent-id 1 --name "新企业名"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")

			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}

			name, _ := cmd.Flags().GetString("name")
			logo, _ := cmd.Flags().GetString("logo")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			var logoURI string
			if logo != "" {
				fileBytes, fileName, err := validateLogoFile(logo)
				if err != nil {
					return err
				}
				uploadResp, err := userClient.UploadAvatar(fileBytes, fileName, "ent", secret)
				if err != nil {
					return err
				}
				logoURI = uploadResp.URI
			}

			if err := userClient.UpdateEnt(entID, name, logoURI, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("name", "", "企业名称")
	cmd.Flags().String("logo", "", "企业Logo文件路径 (仅支持 JPG/GIF/PNG，小于500KB)")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}
