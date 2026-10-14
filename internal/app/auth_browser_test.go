package app

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authpkg "github.com/gokuai/yunku-cli/internal/auth"
	"github.com/gokuai/yunku-cli/internal/keychain"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

func startTestCallbackServer(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln, "http://" + ln.Addr().String() + browserLoginCallbackPath
}

func TestReceiveBrowserToken_Success(t *testing.T) {
	ln, redirect := startTestCallbackServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan *browserTokenResult, 1)
	errDone := make(chan error, 1)
	go func() {
		res, err := receiveBrowserToken(ctx, ln)
		if err != nil {
			errDone <- err
			return
		}
		done <- res
	}()

	// 模拟浏览器登录后 passport 回跳：query 直接携带 token
	callbackURL := redirect + "?access_token=at-123&refresh_token=rt-456&expires_in=21600&token_type=passport"
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status = %d, want 200", resp.StatusCode)
	}

	select {
	case res := <-done:
		if res.AccessToken != "at-123" || res.RefreshToken != "rt-456" ||
			res.ExpiresIn != 21600 || res.TokenType != "passport" {
			t.Fatalf("unexpected token result: %+v", res)
		}
	case err := <-errDone:
		t.Fatalf("receiveBrowserToken error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for token callback")
	}
}

func TestReceiveBrowserToken_OAuthError(t *testing.T) {
	ln, redirect := startTestCallbackServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errDone := make(chan error, 1)
	go func() {
		_, err := receiveBrowserToken(ctx, ln)
		errDone <- err
	}()

	resp, err := http.Get(redirect + "?error=access_denied&error_description=user+cancelled")
	if err != nil {
		t.Fatalf("callback GET failed: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case err := <-errDone:
		if err == nil || !bytes.Contains([]byte(err.Error()), []byte("access_denied")) {
			t.Fatalf("expected access_denied error, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for error callback")
	}
}

func TestReceiveBrowserToken_MissingAccessToken(t *testing.T) {
	ln, redirect := startTestCallbackServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errDone := make(chan error, 1)
	go func() {
		_, err := receiveBrowserToken(ctx, ln)
		errDone <- err
	}()

	resp, err := http.Get(redirect)
	if err != nil {
		t.Fatalf("callback GET failed: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case err := <-errDone:
		if err == nil {
			t.Fatal("expected error for callback without access_token")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for callback")
	}
}

func TestReceiveBrowserToken_Timeout(t *testing.T) {
	ln, _ := startTestCallbackServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := receiveBrowserToken(ctx, ln)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("timeout handling took too long")
	}
}

func TestCompleteBrowserLogin_PersistsTokenAndCredentials(t *testing.T) {
	// 隔离 keychain 服务与配置目录，避免读写真实登录凭证。
	t.Setenv("YKC_KEYCHAIN_SERVICE", "ykc-cli-test-app-browser")
	t.Cleanup(func() {
		_ = keychain.Remove("ykc-cli-test-app-browser", keychain.AccountToken)
	})

	configDir := filepath.Join(t.TempDir(), "config")
	t.Setenv("YKC_CONFIG_DIR", configDir)

	if err := authpkg.SaveTokenData(configDir, &authpkg.TokenData{AccessToken: "preflight"}); err != nil {
		t.Skipf("SaveTokenData() unavailable in this environment: %v", err)
	}
	_ = authpkg.DeleteTokenData(configDir)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	// completeBrowserLogin 接收的是 exchange_token 兑换后的正式 token
	resp := &gokuai.TokenResponse{
		AccessToken:  "final-at",
		RefreshToken: "final-rt",
		ExpiresIn:    21600,
	}
	if err := completeBrowserLogin(cmd, "oauth-cid", "the-secret", resp); err != nil {
		t.Fatalf("completeBrowserLogin error: %v", err)
	}

	data, err := loadGokuaiTokenData()
	if err != nil {
		t.Fatalf("load token data: %v", err)
	}
	if data.AccessToken != "final-at" || data.RefreshToken != "final-rt" {
		t.Fatalf("persisted token mismatch: %+v", data)
	}
	if !data.IsAccessTokenValid() {
		t.Fatal("persisted access token should be valid")
	}

	// 兑换用的签名凭证一并落盘，登录后的命令无需再依赖环境变量
	cid, secret, err := loadGokuaiUserCredentials()
	if err != nil {
		t.Fatalf("load user credentials: %v", err)
	}
	if cid != "oauth-cid" || secret != "the-secret" {
		t.Fatalf("persisted credentials mismatch: %q/%q", cid, secret)
	}

	if !bytes.Contains(out.Bytes(), []byte("final-at")) {
		t.Fatalf("command output should contain the access token: %s", out.String())
	}
}

func TestResolveBrowserExchangeClient(t *testing.T) {
	t.Setenv("GOKUAI_USER_CLIENT_ID", "")
	t.Setenv("GOKUAI_USER_CLIENT_SECRET", "")
	t.Setenv("YKC_CONFIG_DIR", t.TempDir())

	t.Run("env user credentials win", func(t *testing.T) {
		t.Setenv("GOKUAI_USER_CLIENT_ID", "user-cid")
		t.Setenv("GOKUAI_USER_CLIENT_SECRET", "user-secret")
		cid, secret, err := resolveBrowserExchangeClient("oauth-cid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cid != "user-cid" || secret != "user-secret" {
			t.Fatalf("got %q/%q, want user-cid/user-secret", cid, secret)
		}
	})

	t.Run("oauth client id with env secret", func(t *testing.T) {
		t.Setenv("GOKUAI_USER_CLIENT_SECRET", "oauth-secret")
		cid, secret, err := resolveBrowserExchangeClient("oauth-cid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cid != "oauth-cid" || secret != "oauth-secret" {
			t.Fatalf("got %q/%q, want oauth-cid/oauth-secret", cid, secret)
		}
	})

	t.Run("missing secret returns validation error", func(t *testing.T) {
		_, _, err := resolveBrowserExchangeClient("oauth-cid")
		if err == nil {
			t.Fatal("expected error when no secret is configured")
		}
		if !bytes.Contains([]byte(err.Error()), []byte("GOKUAI_USER_CLIENT_SECRET")) {
			t.Fatalf("error should name the expected config, got: %v", err)
		}
	})

	t.Run("saved credentials of the same client are reused", func(t *testing.T) {
		if err := saveGokuaiUserCredentials("oauth-cid", "saved-secret"); err != nil {
			t.Fatalf("save credentials: %v", err)
		}
		cid, secret, err := resolveBrowserExchangeClient("oauth-cid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cid != "oauth-cid" || secret != "saved-secret" {
			t.Fatalf("got %q/%q, want oauth-cid/saved-secret", cid, secret)
		}
	})

	t.Run("saved credentials of another client are ignored", func(t *testing.T) {
		if err := saveGokuaiUserCredentials("other-cid", "other-secret"); err != nil {
			t.Fatalf("save credentials: %v", err)
		}
		if _, _, err := resolveBrowserExchangeClient("oauth-cid"); err == nil {
			t.Fatal("expected error when saved secret belongs to another client")
		}
	})
}

// TestRunBrowserLogin_ExchangesPassportToken 覆盖完整浏览器登录流程：
// 本地回调收到 passport 短时效网页 token 后，必须通过 /m-api/oauth2/token2
// 的 exchange_token 授权兑换成正式 token 再落盘（回归：曾直接保存网页 token，
// 导致 ykc auth status 报未登录）。
func TestRunBrowserLogin_ExchangesPassportToken(t *testing.T) {
	t.Setenv("YKC_KEYCHAIN_SERVICE", "ykc-cli-test-app-browser-flow")
	t.Cleanup(func() {
		_ = keychain.Remove("ykc-cli-test-app-browser-flow", keychain.AccountToken)
	})

	configDir := filepath.Join(t.TempDir(), "config")
	t.Setenv("YKC_CONFIG_DIR", configDir)

	if err := authpkg.SaveTokenData(configDir, &authpkg.TokenData{AccessToken: "preflight"}); err != nil {
		t.Skipf("SaveTokenData() unavailable in this environment: %v", err)
	}
	_ = authpkg.DeleteTokenData(configDir)

	// mock /m-api/oauth2/token2：校验 exchange_token 请求参数，返回正式 token
	var token2Form url.Values
	token2Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/m-api/oauth2/token2" {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		token2Form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"final-at","expires_in":21600,"refresh_token":"final-rt"}`))
	}))
	defer token2Srv.Close()

	t.Setenv("GOKUAI_API_HOST", token2Srv.URL)
	t.Setenv("GOKUAI_OAUTH_CLIENT_ID", "oauth-cid")
	t.Setenv("GOKUAI_USER_CLIENT_ID", "")
	t.Setenv("GOKUAI_USER_CLIENT_SECRET", "the-secret")
	t.Setenv("BROWSER", "/usr/bin/true")

	var out, errBuf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	done := make(chan error, 1)
	go func() {
		done <- runBrowserLogin(cmd, "oauth-cid")
	}()

	// 轮询 stderr 拿到授权地址，模拟浏览器登录后 passport 回跳网页 token
	loginURL := waitForLine(t, &errBuf, "授权地址: ")
	redirect := redirectURIFromLoginURL(t, loginURL)
	callbackResp, err := http.Get(redirect + "?access_token=passport-web-token&expires_in=300&token_type=passport")
	if err != nil {
		t.Fatalf("callback GET failed: %v", err)
	}
	_ = callbackResp.Body.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runBrowserLogin error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for runBrowserLogin")
	}

	// 兑换请求参数正确（ExchangeToken 以 auth 参数标识 passport 认证器）
	if got := token2Form.Get("grant_type"); got != "exchange_token" {
		t.Errorf("grant_type = %q, want exchange_token", got)
	}
	if got := token2Form.Get("auth"); got != "passport" {
		t.Errorf("auth = %q, want passport", got)
	}
	if got := token2Form.Get("exchange_token"); got != "passport-web-token" {
		t.Errorf("exchange_token = %q, want passport-web-token", got)
	}
	if got := token2Form.Get("client_id"); got != "oauth-cid" {
		t.Errorf("client_id = %q, want oauth-cid", got)
	}

	// 输出与落盘的都是兑换后的正式 token，而非网页 token
	if !bytes.Contains(out.Bytes(), []byte("final-at")) {
		t.Fatalf("output should contain the exchanged token, got: %s", out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("passport-web-token")) {
		t.Fatalf("output must not leak the passport web token: %s", out.String())
	}

	data, err := loadGokuaiTokenData()
	if err != nil {
		t.Fatalf("load token data: %v", err)
	}
	if data.AccessToken != "final-at" || data.RefreshToken != "final-rt" {
		t.Fatalf("persisted token mismatch: %+v", data)
	}

	cid, secret, err := loadGokuaiUserCredentials()
	if err != nil {
		t.Fatalf("load user credentials: %v", err)
	}
	if cid != "oauth-cid" || secret != "the-secret" {
		t.Fatalf("persisted credentials mismatch: %q/%q", cid, secret)
	}
}

// waitForLine 等待 buf 中出现以 prefix 开头的行并返回该行内容。
func waitForLine(t *testing.T, buf *bytes.Buffer, prefix string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(buf.String(), "\n") {
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for stderr line with prefix %q, buf: %s", prefix, buf.String())
	return ""
}

// redirectURIFromLoginURL 从授权地址中解析 redirect_uri 参数。
func redirectURIFromLoginURL(t *testing.T, loginURL string) string {
	t.Helper()
	u, err := url.Parse(loginURL)
	if err != nil {
		t.Fatalf("parse login URL: %v", err)
	}
	redirect := u.Query().Get("redirect_uri")
	if redirect == "" {
		t.Fatalf("login URL has no redirect_uri: %s", loginURL)
	}
	return redirect
}

func TestResolveBrowserOAuthClientID(t *testing.T) {
	t.Setenv("GOKUAI_OAUTH_CLIENT_ID", "")

	t.Run("missing config returns validation error", func(t *testing.T) {
		cmd := newAuthLoginCommand()
		if _, err := resolveBrowserOAuthClientID(cmd); err == nil {
			t.Fatal("expected error when oauth client id is absent")
		}
	})

	t.Run("env value is used", func(t *testing.T) {
		t.Setenv("GOKUAI_OAUTH_CLIENT_ID", "env-cid")
		cmd := newAuthLoginCommand()
		cid, err := resolveBrowserOAuthClientID(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cid != "env-cid" {
			t.Fatalf("got %q, want env-cid", cid)
		}
	})

	t.Run("flag takes precedence over env", func(t *testing.T) {
		t.Setenv("GOKUAI_OAUTH_CLIENT_ID", "env-cid")
		cmd := newAuthLoginCommand()
		if err := cmd.Flags().Set("oauth-client-id", "flag-cid"); err != nil {
			t.Fatal(err)
		}
		cid, err := resolveBrowserOAuthClientID(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cid != "flag-cid" {
			t.Fatalf("got %q, want flag-cid", cid)
		}
	})
}

func TestAuthLoginBrowser_MissingOAuthClientID(t *testing.T) {
	// 浏览器模式只认独立 oauth client_id：未设置 GOKUAI_OAUTH_CLIENT_ID 时
	// 即使存在其他凭证也应直接报 validation 错误，而不是走 get_client_oauth。
	t.Setenv("GOKUAI_OAUTH_CLIENT_ID", "")
	t.Setenv("GOKUAI_USER_CLIENT_ID", "")
	t.Setenv("GOKUAI_USER_CLIENT_SECRET", "")
	t.Setenv("GOKUAI_CLIENT_ID", "")
	t.Setenv("GOKUAI_CLIENT_SECRET", "")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth", "login", "--browser"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("GOKUAI_OAUTH_CLIENT_ID")) {
		t.Fatalf("error should name the expected config, got: %v", err)
	}
}
