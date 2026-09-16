package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
)

const tokenRefreshLockTimeout = 15 * time.Second

// ensureGokuaiAccessToken returns a valid access token, proactively refreshing
// it with the stored refresh token when the local access token has expired
// (or is within its 5-minute validity buffer). It is used when building the
// user client so every ykc command starts with a usable token.
func ensureGokuaiAccessToken() (string, error) {
	data, err := loadGokuaiTokenData()
	if err != nil {
		return "", err
	}
	if data != nil && data.IsAccessTokenValid() {
		return data.AccessToken, nil
	}
	return refreshGokuaiAccessToken()
}

// refreshGokuaiAccessToken exchanges the stored refresh token for a new
// access token. A cross-process file lock serializes refreshes so concurrent
// ykc invocations do not race; after acquiring the lock the token store is
// re-read in case another process already refreshed, avoiding duplicate
// refresh calls.
func refreshGokuaiAccessToken() (string, error) {
	configDir := defaultConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}
	lockPath := filepath.Join(configDir, "token.refresh.lock")

	ctx, cancel := context.WithTimeout(context.Background(), tokenRefreshLockTimeout)
	defer cancel()

	unlock, err := acquireTokenRefreshLock(ctx, lockPath)
	if err != nil {
		return "", apperrors.NewAuth(fmt.Sprintf("等待 token 刷新锁超时: %v", err))
	}
	defer unlock()

	// Another process may have refreshed while we waited for the lock.
	data, err := loadGokuaiTokenData()
	if err != nil {
		return "", err
	}
	if data != nil && data.IsAccessTokenValid() {
		return data.AccessToken, nil
	}
	if data == nil || data.RefreshToken == "" {
		return "", apperrors.NewAuth("登录已失效，请重新登录 (ykc auth login)")
	}

	userClientID, userClientSecret, err := loadGokuaiUserCredentials()
	if err != nil || userClientID == "" || userClientSecret == "" {
		return "", apperrors.NewAuth("用户凭证未找到，请重新登录 (ykc auth login)")
	}

	slog.Info("gokuai access token expired, refreshing via refresh token")
	client := gokuai.NewAuthClient(userClientID, userClientSecret)
	resp, err := client.RefreshToken(data.RefreshToken)
	if err != nil {
		return "", apperrors.NewAuth(
			fmt.Sprintf("刷新 token 失败，请重新登录 (ykc auth login): %v", err),
			apperrors.WithCause(err))
	}

	if err := saveRefreshedGokuaiToken(resp); err != nil {
		return "", fmt.Errorf("保存刷新后的 token 失败: %w", err)
	}
	return resp.AccessToken, nil
}

// newTokenRefresher returns the reactive refresh callback for UserClient,
// invoked when the API responds with error_code 40102 despite a seemingly
// valid local token. It reuses the same locked refresh path; a nil value
// disables the behavior (e.g. bearer-token mode).
func newTokenRefresher() func() (string, error) {
	return refreshGokuaiAccessToken
}
