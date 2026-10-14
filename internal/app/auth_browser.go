package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gokuai/yunku-cli/internal/browser"
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

const (
	// browserLoginCallbackPath 是本地回调 HTTP 服务接收授权结果的路径。
	browserLoginCallbackPath = "/callback"
	// browserLoginTimeout 是等待用户在浏览器中完成登录的最长时间。
	browserLoginTimeout = 5 * time.Minute
	// passportExchangeAuth 是 exchange_token 兑换时传给 token2 接口的
	// auth 值：服务端据此加载 PassportAuth（gokuai 的 passport 网页授权
	// 认证器），由其回调 passport /oauth/check_login 验证回跳的网页 token。
	passportExchangeAuth = "passport"
)

// browserTokenResult 是 passport 登录成功后回跳到本地服务的 token 集合
// （passport 自定义 Passport TokenType 的回跳参数）。
type browserTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
}

// runBrowserLogin 执行浏览器 OAuth2 登录：
//  1. 在 127.0.0.1 随机端口启动一次性的本地回调 HTTP 服务；
//  2. 打开浏览器访问 passport /oauth/login 授权页（redirect_uri 指向本地服务）；
//  3. 用户登录后浏览器被带回本地回调，query 中携带 passport 短时效网页 token
//     （约 300 秒，非最终凭证）；
//  4. 立即以 exchange_token 授权（domain=passport）调用 /m-api/oauth2/token2，
//     把网页 token 兑换成与密码登录等价的正式 token 并保存。
//
// oauthClientID 是浏览器授权页使用的独立 oauth client
// (GOKUAI_OAUTH_CLIENT_ID)。前提：该 client 已开通 authorization_code
// 授权，并在 passport 的 oauth_client_endpoints 中为用户所属站点登记了
// host 为 127.0.0.1 的 redirect_uri（passport 仅校验 host，不校验端口和
// 路径）。
func runBrowserLogin(cmd *cobra.Command, oauthClientID string) (*browserTokenResult , error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("启动本地登录回调服务失败: %v", err), apperrors.WithCause(err))
	}
	redirectURI := "http://" + listener.Addr().String() + browserLoginCallbackPath

	state, err := randomOAuthState()
	if err != nil {
		return nil, apperrors.NewAPI(fmt.Sprintf("生成登录状态参数失败: %v", err), apperrors.WithCause(err))
	}
	loginURL := gokuai.BuildPassportLoginURL(oauthClientID, redirectURI, state, "")

	stderr := cmd.ErrOrStderr()
	fmt.Fprintf(stderr, "正在打开浏览器进行 OAuth2 授权登录...\n")
	fmt.Fprintf(stderr, "授权地址: %s\n", loginURL)
	if err := browser.Open(loginURL); err != nil {
		fmt.Fprintln(stderr, "未能自动打开浏览器，请手动复制上面的地址到浏览器中打开。")
		slog.Debug("open browser failed", "error", err)
	}
	fmt.Fprintf(stderr, "请在浏览器中完成登录（等待上限 %v）...\n", browserLoginTimeout)

	// 裸 Command（未经 Execute）的 Context() 可能为 nil，兜底为 Background
	parentCtx := cmd.Context()
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, browserLoginTimeout)
	defer cancel()

	result, err := receiveBrowserToken(ctx, listener)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, apperrors.NewAuth("浏览器登录超时，请重新执行登录命令")
		}
		return nil, apperrors.NewAuth(fmt.Sprintf("浏览器登录失败: %v", err), apperrors.WithCause(err))
	}
	return result, nil
}

// resolveBrowserExchangeClient 解析浏览器登录第二步（exchange_token 兑换）
// 使用的 client 凭证，解析规则与登录后用户 API 的签名凭证一致：
//   - client_id: GOKUAI_USER_CLIENT_ID > 浏览器授权 client (GOKUAI_OAUTH_CLIENT_ID)
//   - secret:    GOKUAI_USER_CLIENT_SECRET > 本地保存的同 client_id 的 secret
//
// 兑换所得 token 绑定该 client，登录后的 m-api 签名也用同一凭证，
// 因此两者必须一致，否则兑换成功后的命令会签名失败。
func resolveBrowserExchangeClient(oauthClientID string) (string, string, error) {
	exchangeClientID := oauthClientID
	if id := os.Getenv("GOKUAI_USER_CLIENT_ID"); id != "" {
		exchangeClientID = id
	}

	secret := os.Getenv("GOKUAI_USER_CLIENT_SECRET")
	if secret == "" {
		// 回退到本地保存的凭证，但仅当其 client_id 与本次兑换的 client
		// 一致时才可用，避免误用其他 client 的 secret 导致签名错误。
		if cid, saved, err := loadGokuaiUserCredentials(); err == nil && saved != "" && cid == exchangeClientID {
			secret = saved
		}
	}
	if secret == "" {
		return "", "", apperrors.NewValidation(
			"浏览器登录兑换 token 需要 client_secret 签名，请设置环境变量 GOKUAI_USER_CLIENT_SECRET（需与所用 client_id 配对）")
	}
	return exchangeClientID, secret, nil
}

// receiveBrowserToken 在 listener 上运行一次性回调服务，阻塞直到收到
// passport 回跳的 token 参数、授权错误或 ctx 结束。
func receiveBrowserToken(ctx context.Context, listener net.Listener) (*browserTokenResult, error) {
	resultCh := make(chan *browserTokenResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(browserLoginCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if oauthErr := q.Get("error"); oauthErr != "" {
			desc := q.Get("error_description")
			writeCallbackPage(w, false, "授权失败", fmt.Sprintf("%s: %s", oauthErr, desc))
			select {
			case errCh <- fmt.Errorf("授权被拒绝 (%s: %s)", oauthErr, desc):
			default:
			}
			return
		}

		accessToken := q.Get("access_token")
		if accessToken == "" {
			writeCallbackPage(w, false, "授权失败", "回调参数中缺少 access_token，请返回终端重新登录。")
			select {
			case errCh <- errors.New("回调参数中缺少 access_token"):
			default:
			}
			return
		}

		expiresIn := 0
		if raw := q.Get("expires_in"); raw != "" {
			if n, convErr := strconv.Atoi(raw); convErr == nil {
				expiresIn = n
			}
		}

		writeCallbackPage(w, true, "登录成功", "CLI 授权已完成，可以关闭本页面返回终端。")
		select {
		case resultCh <- &browserTokenResult{
			AccessToken:  accessToken,
			RefreshToken: q.Get("refresh_token"),
			ExpiresIn:    expiresIn,
			TokenType:    q.Get("token_type"),
		}:
		default:
		}
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case result := <-resultCh:
		shutdownCallbackServer(srv)
		return result, nil
	case err := <-errCh:
		shutdownCallbackServer(srv)
		return nil, err
	case err := <-serveErr:
		return nil, fmt.Errorf("本地回调服务异常: %w", err)
	case <-ctx.Done():
		shutdownCallbackServer(srv)
		return nil, ctx.Err()
	}
}

func shutdownCallbackServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// randomOAuthState 生成一次性的随机 state 参数（32 位十六进制字符）。
func randomOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// writeCallbackPage 向浏览器输出回调结果页。所有动态内容均做 HTML 转义。
func writeCallbackPage(w http.ResponseWriter, success bool, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	status := "error"
	icon := "✕"
	if success {
		status = "success"
		icon = "✓"
	}
	safeTitle := html.EscapeString(title)
	fmt.Fprintf(w, callbackPageHTML, safeTitle, status, icon, safeTitle, html.EscapeString(message))
}

const callbackPageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
body{margin:0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f5f6f8;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{background:#fff;border-radius:12px;padding:40px 48px;box-shadow:0 4px 24px rgba(0,0,0,.08);text-align:center;max-width:420px}
.status{font-size:40px;line-height:1;margin-bottom:16px}
.success .status{color:#2e9b55}.error .status{color:#d04141}
h1{font-size:20px;margin:0 0 12px}
p{color:#666;margin:0;font-size:14px;line-height:1.6}
</style>
</head>
<body>
<div class="card %s"><div class="status">%s</div><h1>%s</h1><p>%s</p></div>
</body>
</html>`
