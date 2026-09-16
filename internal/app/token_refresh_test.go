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
	t.Cleanup(func() {
		_ = keychain.Remove(keychain.Service, keychain.AccountToken)
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
	t.Cleanup(func() {
		_ = keychain.Remove(keychain.Service, keychain.AccountToken)
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
