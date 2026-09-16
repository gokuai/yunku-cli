package app

import (
	"testing"
)

// TestEntFileBindFlagsHiddenWhenOnlyOrgClientConfigured verifies that the
// auto-bind flags (--org-id/--mount-id) are hidden when the user has directly
// configured library credentials (GOKUAI_ORG_CLIENT_ID) without enterprise
// credentials (GOKUAI_CLIENT_ID) — in that state auto-bind cannot work, so
// the flags would only mislead.
func TestEntFileBindFlagsHiddenWhenOnlyOrgClientConfigured(t *testing.T) {
	cases := []struct {
		name       string
		orgClient  string
		entClient  string
		wantHidden bool
	}{
		{name: "only-org-client", orgClient: "org-id-1", entClient: "", wantHidden: true},
		{name: "both-configured", orgClient: "org-id-1", entClient: "ent-id-1", wantHidden: false},
		{name: "only-ent-client", orgClient: "", entClient: "ent-id-1", wantHidden: false},
		{name: "neither", orgClient: "", entClient: "", wantHidden: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOKUAI_ORG_CLIENT_ID", tc.orgClient)
			t.Setenv("GOKUAI_CLIENT_ID", tc.entClient)

			cmd := newEntFileCommand()
			for _, flagName := range []string{"org-id", "mount-id"} {
				flag := cmd.PersistentFlags().Lookup(flagName)
				if flag == nil {
					t.Fatalf("flag --%s not registered", flagName)
				}
				if flag.Hidden != tc.wantHidden {
					t.Errorf("flag --%s Hidden = %v, want %v", flagName, flag.Hidden, tc.wantHidden)
				}
			}
		})
	}
}
