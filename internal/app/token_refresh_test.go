package app

import (
	"path/filepath"
	"testing"
	"time"

	authpkg "github.com/gokuai/yunku-cli/internal/auth"
	"github.com/gokuai/yunku-cli/internal/keychain"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
)

func TestSaveRefreshedGokuaiTokenKeepsOldRefreshTokenWhenNotRotated(t *testing.T) {
	// 隔离 keychain 服务并清理测试数据，避免读写/删除真实登录凭证。
	// go test 并发运行不同包的测试进程，各包需使用不同服务名以免互相干扰。
	t.Setenv("YKC_KEYCHAIN_SERVICE", "ykc-cli-test-app")
	t.Cleanup(func() {
		_ = keychain.Remove("ykc-cli-test-app", keychain.AccountToken)
	})

	configDir := filepath.Join(t.TempDir(), "config")
	t.Setenv("YKC_CONFIG_DIR", configDir)

	original := &authpkg.TokenData{
		AccessToken:  "old-access",
		RefreshToken: "stable-refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour),
		RefreshExpAt: time.Now().Add(24 * time.Hour),
		CorpID:       "corp-1",
	}
	if err := authpkg.SaveTokenData(configDir, original); err != nil {
		t.Skipf("SaveTokenData() unavailable in this environment: %v", err)
	}

	// Server returns a new access token but does NOT rotate the refresh token.
	resp := &gokuai.TokenResponse{
		AccessToken: "new-access",
		ExpiresIn:   7200,
	}
	if err := saveRefreshedGokuaiToken(resp); err != nil {
		t.Fatalf("saveRefreshedGokuaiToken: %v", err)
	}

	loaded, err := authpkg.LoadTokenData(configDir)
	if err != nil {
		t.Fatalf("LoadTokenData: %v", err)
	}
	if loaded.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", loaded.AccessToken)
	}
	if loaded.RefreshToken != "stable-refresh-token" {
		t.Errorf("RefreshToken = %q, want previous token preserved (got %q)",
			loaded.RefreshToken, "stable-refresh-token")
	}
	if loaded.CorpID != "corp-1" {
		t.Errorf("CorpID = %q, want other fields preserved", loaded.CorpID)
	}
	if !loaded.ExpiresAt.After(time.Now()) {
		t.Errorf("ExpiresAt = %v, want a future time", loaded.ExpiresAt)
	}
}

func TestSaveRefreshedGokuaiTokenReplacesRotatedRefreshToken(t *testing.T) {
	// 隔离 keychain 服务并清理测试数据，避免读写/删除真实登录凭证。
	// go test 并发运行不同包的测试进程，各包需使用不同服务名以免互相干扰。
	t.Setenv("YKC_KEYCHAIN_SERVICE", "ykc-cli-test-app")
	t.Cleanup(func() {
		_ = keychain.Remove("ykc-cli-test-app", keychain.AccountToken)
	})

	configDir := filepath.Join(t.TempDir(), "config")
	t.Setenv("YKC_CONFIG_DIR", configDir)

	if err := authpkg.SaveTokenData(configDir, &authpkg.TokenData{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
	}); err != nil {
		t.Skipf("SaveTokenData() unavailable in this environment: %v", err)
	}

	resp := &gokuai.TokenResponse{
		AccessToken:      "new-access",
		RefreshToken:     "rotated-refresh",
		ExpiresIn:        3600,
		RefreshExpiresIn: 86400,
	}
	if err := saveRefreshedGokuaiToken(resp); err != nil {
		t.Fatalf("saveRefreshedGokuaiToken: %v", err)
	}

	loaded, err := authpkg.LoadTokenData(configDir)
	if err != nil {
		t.Fatalf("LoadTokenData: %v", err)
	}
	if loaded.RefreshToken != "rotated-refresh" {
		t.Errorf("RefreshToken = %q, want rotated-refresh", loaded.RefreshToken)
	}
	if !loaded.RefreshExpAt.After(time.Now()) {
		t.Errorf("RefreshExpAt = %v, want a future time after rotation", loaded.RefreshExpAt)
	}
}
