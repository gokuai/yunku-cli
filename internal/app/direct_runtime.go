package app

import (
	"os"
	"strings"
	"sync"

	"github.com/gokuai/yunku-cli/internal/cli"
	"github.com/gokuai/yunku-cli/internal/executor"
)

var (
	dynamicMu            sync.RWMutex
	dynamicEndpoints     map[string]string
	dynamicProducts      map[string]bool
	dynamicAliases       map[string]string
	dynamicToolEndpoints map[string]string // tool name → endpoint
)

var legacyDirectRuntimeAliases = map[string]string{
	"tb":                        "teambition",
	"ykc-discovery":         "discovery",
	"ykc-ai-sincere-hire":  "ai-sincere-hire",
}

// SetDynamicServers is a no-op since MCP infrastructure has been removed.
// Dynamic servers are no longer supported.
func SetDynamicServers() {
	dynamicMu.Lock()
	defer dynamicMu.Unlock()
	dynamicEndpoints = nil
	dynamicProducts = nil
	dynamicAliases = nil
	dynamicToolEndpoints = nil
}

func shouldUseDirectRuntime(invocation executor.Invocation) bool {
	if strings.TrimSpace(os.Getenv(cli.CatalogFixtureEnv)) != "" {
		return false
	}
	switch invocation.Kind {
	case "compat_invocation", "helper_invocation":
		return true
	default:
		return false
	}
}

func directRuntimeEndpoint(productID, toolName string) (string, bool) {
	// Priority 0: env-var override always wins (YKC_<PRODUCT>_MCP_URL).
	normalized := normalizeDirectRuntimeProductID(productID)
	for _, candidate := range []string{strings.TrimSpace(productID), normalized} {
		if candidate == "" {
			continue
		}
		if override, ok := productEndpointOverride(candidate); ok {
			return override, true
		}
	}

	dynamicMu.RLock()
	de := dynamicEndpoints
	te := dynamicToolEndpoints
	dynamicMu.RUnlock()

	// Priority 1: tool-level endpoint (resolves multi-endpoint products).
	if tool := strings.TrimSpace(toolName); tool != "" && te != nil {
		if endpoint, ok := te[tool]; ok {
			return endpoint, true
		}
	}

	// Priority 2: product-level endpoint.
	for _, candidate := range []string{strings.TrimSpace(productID), normalized} {
		if candidate == "" {
			continue
		}
		if de != nil {
			if endpoint, ok := de[candidate]; ok {
				return endpoint, true
			}
		}
	}
	return "", false
}

// DirectRuntimeProductIDs returns the set of product IDs that have direct
// runtime endpoints configured, sourced from dynamic server discovery.
func DirectRuntimeProductIDs() map[string]bool {
	dynamicMu.RLock()
	dp := dynamicProducts
	dynamicMu.RUnlock()
	ids := make(map[string]bool, len(dp))
	for key := range dp {
		ids[key] = true
	}
	return ids
}

func normalizeDirectRuntimeProductID(productID string) string {
	dynamicMu.RLock()
	da := dynamicAliases
	dynamicMu.RUnlock()
	trimmed := strings.TrimSpace(productID)
	if da != nil {
		if normalizedID, ok := da[trimmed]; ok && normalizedID != "" {
			return normalizedID
		}
	}
	if normalizedID, ok := legacyDirectRuntimeAliases[trimmed]; ok {
		return normalizedID
	}
	return trimmed
}
