package app

import (
	"bytes"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	authpkg "github.com/gokuai/yunku-cli/internal/auth"
	"github.com/gokuai/yunku-cli/internal/keychain"
)

func TestAuthStatusRefreshFailureLeavesStoredTokenIntact(t *testing.T) {
	// 隔离 keychain 服务并清理测试数据，避免读写/删除真实登录凭证。
	// go test 并发运行不同包的测试进程，各包需使用不同服务名以免互相干扰。
	t.Setenv("YKC_KEYCHAIN_SERVICE", "ykc-cli-test-app")
	t.Cleanup(func() {
		_ = keychain.Remove("ykc-cli-test-app", keychain.AccountToken)
	})

	root := t.TempDir()
	configDir := filepath.Join(root, "config")

	t.Setenv("YKC_CONFIG_DIR", configDir)

	err := authpkg.SaveTokenData(configDir, &authpkg.TokenData{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-123",
		ExpiresAt:    time.Now().Add(-time.Hour),
		RefreshExpAt: time.Now().Add(24 * time.Hour),
		CorpID:       "dingcorp",
	})
	if err != nil {
		t.Skipf("SaveTokenData() unavailable in this environment: %v", err)
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("refresh failed")
	})

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"auth", "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput:\n%s", err, out.String())
	}

	// Verify token data still exists in keychain after refresh failure
	if !authpkg.TokenDataExistsKeychain() {
		t.Fatal("secure token data should remain in keychain after refresh failure")
	}

	if !bytes.Contains(out.Bytes(), []byte("\"authenticated\"")) {
		t.Fatalf("output should still report authenticated status:\n%s", out.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
