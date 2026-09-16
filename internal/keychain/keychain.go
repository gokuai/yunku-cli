// Package keychain provides cross-platform secure storage for secrets.
// - macOS: System Keychain stores DEK (Data Encryption Key), data encrypted with AES-256-GCM
// - Linux: File-based DEK storage with AES-256-GCM encryption
// - Windows: DPAPI + Registry storage
package keychain

const (
	// Service is the unified keychain service name for all secrets.
	Service = "ykc-cli"

	// AccountToken is the account key for storing auth token data.
	AccountToken = "auth-token"
)

// KeychainAccess abstracts keychain Get/Set/Remove for dependency injection.
type KeychainAccess interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Remove(service, account string) error
}

// Get retrieves a value from the keychain.
// Returns empty string and nil error if the entry does not exist.
func Get(service, account string) (string, error) {
	return platformGet(service, account)
}

// Set stores a value in the keychain, overwriting any existing entry.
func Set(service, account, data string) error {
	return platformSet(service, account, data)
}

// Remove deletes an entry from the keychain.
// Returns nil if the entry does not exist.
func Remove(service, account string) error {
	return platformRemove(service, account)
}

// Exists checks if an entry exists in the keychain.
func Exists(service, account string) bool {
	val, err := Get(service, account)
	return err == nil && val != ""
}
