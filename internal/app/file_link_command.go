package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

func newFileLinkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "文件外链操作",
	}

	cmd.AddCommand(
		newFileLinkCreateCommand(),
		newFileLinkCloseCommand(),
		newFileLinkListCommand(),
		newFileLinkUpdateCommand(),
		newFileLinkDetailCommand(),
	)

	return cmd
}

func setPublicFlags(cmd *cobra.Command) {
	cmd.Flags().String("deadline", "2d", "过期时间戳(秒)或字符串(如2d/1w/1m/1y), -1永不失效")
	cmd.Flags().String("auth", "100", "权限: 100预览, 110预览+下载, 111预览+下载+上传, 101预览+上传, 001仅上传")
	cmd.Flags().String("password", "", "外链密码")
	cmd.Flags().String("scope", "0", "访问范围: 0所有人, 1仅企业成员, 3需身份验证")
	cmd.Flags().String("startline", "", "生效时间戳(正整数秒)，不传则立即生效")
	cmd.Flags().String("emails", "", "发送到邮箱地址列表，分号分隔")
	cmd.Flags().String("content", "", "发送邮箱时告知的内容")
	cmd.Flags().Bool("keep", false, "外链不随文件修改而更新 (不添加此参数则随文件更新)")
	cmd.Flags().String("access-limit", "", "访问次数限制，默认不限制")
	cmd.Flags().String("authems", "", "scope为3时身份验证的邮箱或手机号，分号分隔")
	cmd.Flags().Bool("annotation-view", false, "开启批注显示")
	cmd.Flags().Bool("edit", false, "允许协同编辑")
}

func getPublicParams(cmd *cobra.Command) (map[string]string, error) {
	params := map[string]string{}

	deadline, _ := cmd.Flags().GetString("deadline")
	if err := ValidateDeadline(deadline); err != nil {
		return nil, err
	}
	auth, err := getStringFlagInSet(cmd, "auth", "100", "110", "111", "101", "001")
	if err != nil {
		return nil, err
	}
	password, _ := cmd.Flags().GetString("password")
	scope, err := getStringFlagInSet(cmd, "scope", "0", "1", "3")
	if err != nil {
		return nil, err
	}
	startline, _ := cmd.Flags().GetString("startline")
	if err := ValidateStartline(startline); err != nil {
		return nil, err
	}
	emails, _ := cmd.Flags().GetString("emails")
	content, _ := cmd.Flags().GetString("content")
	keep, _ := cmd.Flags().GetBool("keep")
	accessLimit, _ := cmd.Flags().GetString("access-limit")
	authems, _ := cmd.Flags().GetString("authems")
	if scope == "3" && authems == "" {
		return nil, apperrors.NewValidation("authems 不允许为空")
	}
	if authems != "" {
		phoneRe := regexp.MustCompile(`^1[3-9]\d{9}$`)
		emailRe := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		for _, item := range strings.Split(authems, ";") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !phoneRe.MatchString(item) && !emailRe.MatchString(item) {
				return nil, apperrors.NewValidation(fmt.Sprintf("authems 格式错误，需为手机号或邮箱: %s", item))
			}
		}
	}
	annotationView, _ := cmd.Flags().GetBool("annotation-view")
	edit, _ := cmd.Flags().GetBool("edit")
	if edit && scope == "0" {
		return nil, apperrors.NewValidation("匿名访问的外链无法开启编辑权限")
	}

	if deadline != "" {
		params["deadline"] = deadline
	}
	if auth != "" {
		params["auth"] = auth
	}
	if password != "" {
		params["password"] = password
	}
	if scope != "" {
		params["scope"] = scope
	}
	if startline != "" {
		params["startline"] = startline
	}
	if emails != "" {
		params["emails"] = emails
	}
	if content != "" {
		params["content"] = content
	}
	if keep {
		params["keep"] = "1"
	}
	if accessLimit != "" {
		params["access_limit"] = accessLimit
	}
	if authems != "" {
		params["authems"] = authems
	}
	if annotationView {
		params["annotation_view"] = "1"
	}
	if edit {
		params["edit"] = "1"
	}
	return params, nil
}

// validateAnnotationView 校验 annotation-view 参数：
// 开启批注显示时，企业需已开启 annotation 模块，且目标文件扩展名需在允许范围内。
func validateAnnotationView(userClient *gokuai.UserClient, secret string, mountID int, fullpath, dir string) error {
	emc := gokuai.NewEntModuleConfigWithStore(userClient, secret, cacheStoreFromEnv())
	setting, err := emc.ModuleSetting(mountID, "annotation")
	if err != nil {
		return err
	}

	// 文件夹外链无需校验扩展名
	if dir == "1" {
		return nil
	}

	allowedExt, err := parseAnnotationExtension(setting)
	if err != nil {
		return err
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fullpath), "."))
	if ext == "" {
		return apperrors.NewValidation(fmt.Sprintf("文件 %q 没有扩展名，无法开启批注显示，允许的扩展名: %s", fullpath, allowedExt))
	}
	for _, item := range strings.Split(allowedExt, "|") {
		if strings.EqualFold(strings.TrimSpace(item), ext) {
			return nil
		}
	}
	return apperrors.NewValidation(fmt.Sprintf("文件扩展名 %q 不支持批注显示，允许的扩展名: %s", ext, allowedExt))
}

// parseAnnotationExtension 从 annotation 模块配置中解析允许的扩展名。
// 配置可能是对象 {"extension":"..."}，也可能是 JSON 编码的字符串 "{\"extension\":\"...\"}"。
func parseAnnotationExtension(setting json.RawMessage) (string, error) {
	data := strings.TrimSpace(string(setting))
	if data == "" {
		return "", nil
	}

	// 字符串包裹时先解码为原始 JSON
	if data[0] == '"' {
		var inner string
		if err := json.Unmarshal([]byte(data), &inner); err != nil {
			return "", apperrors.NewInternal(fmt.Sprintf("解析 annotation 配置失败: %v", err))
		}
		data = inner
	}

	var cfg struct {
		Extension string `json:"extension"`
	}
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return "", apperrors.NewInternal(fmt.Sprintf("解析 annotation 配置失败: %v", err))
	}
	return cfg.Extension, nil
}

func newFileLinkCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建文件外链",
		Example: `  ykc file link create --mount-id 1 --fullpath documents/a.txt --dir 0 --deadline 10d
  ykc file link create --mount-id 1 --fullpath documents/a.txt --dir 0 --deadline 2d --password 123456 --auth 110 --scope 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			dir, _ := cmd.Flags().GetString("dir")

			params, err := getPublicParams(cmd)
			if err != nil {
				return err
			}

			params["fullpath"] = fullpath
			params["dir"] = dir

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			// 开启批注显示时校验企业 annotation 模块及文件扩展名
			if params["annotation_view"] == "1" {
				if err := validateAnnotationView(userClient, secret, mountID, fullpath, dir); err != nil {
					return err
				}
			}

			resp, err := userClient.CreateFileLink(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("dir", "", "是否文件夹")
	setPublicFlags(cmd)
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("dir")
	return cmd
}

func newFileLinkUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "更新文件链接设置",
		Example: `  ykc file link update --code abc123 --deadline 1735660800`,
		RunE: func(cmd *cobra.Command, args []string) error {
			code, _ := cmd.Flags().GetString("code")
			params, err := getPublicParams(cmd)
			if err != nil {
				return err
			}
			params["code"] = code
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.UpdateFileLink(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("code", "", "外链code (必需)")
	setPublicFlags(cmd)
	cmd.MarkFlagRequired("code")
	cmd.Flags().String("mount-id", "", "mount-id")
	cmd.Flags().MarkHidden("mount-id")
	return cmd
}

func newFileLinkCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "close",
		Short:   "关闭文件外链",
		Example: `  ykc file link close --mount-id 1 --code u8ffpweilaft22xrhbt6lyjz9m5d815y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			code, _ := cmd.Flags().GetString("code")

			params := map[string]string{
				"code": code,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.CloseFileLink(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("code", "", "外链唯一code (必需)")
	cmd.MarkFlagRequired("code")
	return cmd
}

func newFileLinkListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "获取文件外链列表",
		Example: `  ykc file link list --mount-id 1 --fullpath documents/
  ykc file link list --mount-id 1 --fullpath documents/ --state -1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			state, _ := cmd.Flags().GetString("state")

			params := map[string]string{
				"fullpath": fullpath,
			}
			if state != "" {
				params["state"] = state
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileLinks(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("state", "", "外链状态: -1全部, 1开启, 0关闭(默认未关闭)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileLinkDetailCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "detail",
		Short:   "获取文件外链详情",
		Example: `  ykc file link detail --code abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			code, _ := cmd.Flags().GetString("code")

			params := map[string]string{
				"code": code,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileLinkDetail(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("code", "", "外链code (必需)")
	cmd.MarkFlagRequired("code")
	cmd.Flags().String("mount-id", "", "mount-id")
	cmd.Flags().MarkHidden("mount-id")
	return cmd
}
