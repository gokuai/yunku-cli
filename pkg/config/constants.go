// Package config provides shared constants used across multiple internal
// packages. Only cross-cutting values belong here; package-private
// constants should remain in their own package.
package config

import (
	"os"
	"time"
)

// ── File permissions ────────────────────────────────────────────────────
// These are used consistently by auth, security, and cache packages to
// protect sensitive data on disk.

const (
	// DirPerm is the permission mode for directories that hold sensitive
	// data (token store, cache, lock files). Owner-only rwx.
	DirPerm os.FileMode = 0o700

	// FilePerm is the permission mode for sensitive files (encrypted
	// tokens, cache entries, lock files). Owner-only rw.
	FilePerm os.FileMode = 0o600
)

// ── HTTP ────────────────────────────────────────────────────────────────

const (
	// HTTPTimeout is the default timeout for outgoing HTTP requests.
	HTTPTimeout = 30 * time.Second

	// MaxResponseBodySize limits the amount of data read from a single
	// HTTP response to prevent memory exhaustion from malicious servers.
	MaxResponseBodySize = 10 * 1024 * 1024 // 10 MB
)

// ── Cache ───────────────────────────────────────────────────────────────

const (
	// DefaultPartition is the cache partition used when no tenant/org
	// context is available.
	DefaultPartition = "default/default"
)
