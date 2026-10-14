package app

import (
	"fmt"
	"net/url"

	"github.com/gokuai/yunku-cli/internal/browser"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

// openWebBrowser is a variable so the command can be tested without launching
// a real browser.
var openWebBrowser = browser.Open

func newAuthWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "通过 SSO 打开网页端",
		Example: `  ykc auth web
  ykc auth web --url https://yk3.gokuai.com/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _ := cmd.Flags().GetString("url")
			if err := validateWebTarget(target); err != nil {
				return err
			}

			accessToken, err := ensureGokuaiAccessToken()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			ssoURL, err := gokuai.BuildSSOURL(accessToken, target, secret)
			if err != nil {
				return apperrors.NewAuth(fmt.Sprintf("生成网页端 SSO 地址失败: %v", err), apperrors.WithCause(err))
			}
			if err := openWebBrowser(ssoURL); err != nil {
				return apperrors.NewAPI("无法自动打开网页端，请检查默认浏览器或 BROWSER 配置", apperrors.WithCause(err))
			}

			return outputJSON(cmd, map[string]interface{}{
				"opened":     true,
				"target_url": target,
				"message":    "已通过 SSO 打开网页端",
			})
		},
	}
	cmd.Flags().String("url", gokuai.GetAPIHost()+"/", "网页端目标地址（默认使用 GOKUAI_API_HOST）")
	return cmd
}

func validateWebTarget(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return apperrors.NewValidation("网页端地址必须是完整的 http 或 https URL")
	}
	if u.User != nil {
		return apperrors.NewValidation("网页端地址不允许包含用户信息")
	}
	return nil
}
