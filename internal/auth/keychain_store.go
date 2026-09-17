package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/gokuai/yunku-cli/internal/keychain"
)

var (
	migrationOnce sync.Once
	migrationDone bool
)

// tokenKeychainService 返回 token 存储使用的 keychain 服务名。
// 默认为 keychain.Service；可通过环境变量 YKC_KEYCHAIN_SERVICE 覆盖，
// 用于测试将 token 存储重定向到隔离服务，避免读写用户真实凭证。
func tokenKeychainService() string {
	if s := os.Getenv("YKC_KEYCHAIN_SERVICE"); s != "" {
		return s
	}
	return keychain.Service
}

// SaveTokenDataKeychain saves TokenData to the platform keychain.
// This is the new secure storage method using random master key.
func SaveTokenDataKeychain(data *TokenData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token data: %w", err)
	}
	// Zero sensitive data after use
	defer func() {
		for i := range jsonData {
			jsonData[i] = 0
		}
	}()

	if err := keychain.Set(tokenKeychainService(), keychain.AccountToken, string(jsonData)); err != nil {
		return fmt.Errorf("save to keychain: %w", err)
	}
	return nil
}

// LoadTokenDataKeychain loads TokenData from the platform keychain.
func LoadTokenDataKeychain() (*TokenData, error) {
	jsonStr, err := keychain.Get(tokenKeychainService(), keychain.AccountToken)
	if err != nil {
		return nil, fmt.Errorf("load from keychain: %w", err)
	}
	if jsonStr == "" {
		return nil, fmt.Errorf("no token data in keychain")
	}

	var data TokenData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("parse token data: %w", err)
	}
	return &data, nil
}

// DeleteTokenDataKeychain removes TokenData from the platform keychain.
func DeleteTokenDataKeychain() error {
	return keychain.Remove(tokenKeychainService(), keychain.AccountToken)
}

// TokenDataExistsKeychain checks if token data exists in keychain.
func TokenDataExistsKeychain() bool {
	return keychain.Exists(tokenKeychainService(), keychain.AccountToken)
}

// EnsureMigration performs one-time migration from legacy .data to keychain.
// This should be called early in the auth flow (e.g., during GetAccessToken).
// The migration is idempotent and thread-safe.
func EnsureMigration(configDir string, logger *slog.Logger) {
	migrationOnce.Do(func() {
		result := keychain.MigrateFromLegacy(configDir)
		migrationDone = true

		if result.Migrated {
			if logger != nil {
				logger.Info("migrated token data to secure keychain storage",
					"from", result.FromPath,
					"backup", result.BackupPath)
			}
		} else if result.NeedRelogin {
			if logger != nil {
				logger.Warn("cannot migrate legacy token data, please re-login",
					"error", result.Error)
			}
		} else if result.Error != nil {
			if logger != nil {
				logger.Error("migration failed", "error", result.Error)
			}
		}
	})
}

// IsMigrationDone returns true if migration has been attempted.
func IsMigrationDone() bool {
	return migrationDone
}

// Client credential storage functions.
// These store the clientSecret associated with a specific clientId,
// allowing token refresh to work even if environment variables change.

const clientSecretPrefix = "client-secret:"

// SaveClientSecret stores the client secret for a specific client ID.
// This is called during login to snapshot the credentials used.
func SaveClientSecret(clientID, clientSecret string) error {
	if clientID == "" || clientSecret == "" {
		return nil // Nothing to save
	}
	account := clientSecretPrefix + clientID
	if err := keychain.Set(keychain.Service, account, clientSecret); err != nil {
		return fmt.Errorf("save client secret: %w", err)
	}
	return nil
}

// LoadClientSecret retrieves the stored client secret for a specific client ID.
// Returns empty string if not found.
func LoadClientSecret(clientID string) string {
	if clientID == "" {
		return ""
	}
	account := clientSecretPrefix + clientID
	secret, err := keychain.Get(keychain.Service, account)
	if err != nil {
		return ""
	}
	return secret
}

// DeleteClientSecret removes the stored client secret for a specific client ID.
func DeleteClientSecret(clientID string) error {
	if clientID == "" {
		return nil
	}
	account := clientSecretPrefix + clientID
	return keychain.Remove(keychain.Service, account)
}
