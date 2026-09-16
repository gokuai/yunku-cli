package app

import "testing"

func TestNormalizeDirectRuntimeProductIDPreservesLegacyHiddenVendorRouting(t *testing.T) {
	dynamicMu.Lock()
	previousAliases := dynamicAliases
	dynamicAliases = nil
	dynamicMu.Unlock()
	t.Cleanup(func() {
		dynamicMu.Lock()
		dynamicAliases = previousAliases
		dynamicMu.Unlock()
	})

	cases := map[string]string{
		"tb":                       "teambition",
		"ykc-discovery":       "discovery",
		"ykc-ai-sincere-hire": "ai-sincere-hire",
	}

	for input, want := range cases {
		if got := normalizeDirectRuntimeProductID(input); got != want {
			t.Fatalf("normalizeDirectRuntimeProductID(%q) = %q, want %q", input, got, want)
		}
	}
}
