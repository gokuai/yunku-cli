package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── identity.go ───────────────────────────────────────────────────────

func TestGenerateUUID_Format(t *testing.T) {
	t.Parallel()
	uuid := generateUUID()
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %s", len(parts), uuid)
	}
	if len(uuid) != 36 {
		t.Fatalf("expected 36 chars, got %d: %s", len(uuid), uuid)
	}
	// Version 4 marker
	if uuid[14] != '4' {
		t.Fatalf("expected version 4 at position 14, got %c", uuid[14])
	}
}

func TestIdentity_LoadMissing(t *testing.T) {
	t.Parallel()
	id := Load(t.TempDir())
	if id != nil {
		t.Fatal("expected nil for missing identity")
	}
}

func TestIdentity_EnsureExistsCreatesNew(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	id := EnsureExists(dir)
	if id == nil || id.AgentID == "" {
		t.Fatal("expected non-nil identity with agentID")
	}
	if id.Source != "ykc" {
		t.Fatalf("expected source ykc, got %s", id.Source)
	}
}

func TestIdentity_EnsureExistsLoadsExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	id1 := EnsureExists(dir)
	id2 := EnsureExists(dir)
	if id1.AgentID != id2.AgentID {
		t.Fatalf("expected same ID, got %s vs %s", id1.AgentID, id2.AgentID)
	}
}

func TestIdentity_LoadInvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, identityFile), []byte("not json"), 0o600)
	if Load(dir) != nil {
		t.Fatal("expected nil for invalid JSON")
	}
}

func TestIdentity_LoadEmptyAgentID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	data, _ := json.Marshal(Identity{AgentID: "", Source: "ykc"})
	_ = os.WriteFile(filepath.Join(dir, identityFile), data, 0o600)
	if Load(dir) != nil {
		t.Fatal("expected nil for empty agentID")
	}
}

func TestIdentity_Headers(t *testing.T) {
	t.Parallel()
	id := &Identity{AgentID: "test-uuid", Source: "ykc"}
	h := id.Headers()
	if h["x-ykc-agent-id"] != "test-uuid" {
		t.Fatalf("wrong agent-id header: %s", h["x-ykc-agent-id"])
	}
	if h["x-ykc-source"] != "ykc" {
		t.Fatalf("wrong source header: %s", h["x-ykc-source"])
	}
	if _, ok := h["x-ykc-scenario-code"]; !ok {
		t.Fatal("missing scenario-code header")
	}
}

func TestIdentity_Headers_Nil(t *testing.T) {
	t.Parallel()
	var id *Identity
	if h := id.Headers(); h != nil {
		t.Fatalf("expected nil headers for nil identity, got %v", h)
	}
}

// ─── token.go ──────────────────────────────────────────────────────────

func TestTokenData_IsAccessTokenValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data *TokenData
		want bool
	}{
		{"nil", nil, false},
		{"empty token", &TokenData{}, false},
		{"expired", &TokenData{AccessToken: "t", ExpiresAt: time.Now().Add(-time.Hour)}, false},
		{"valid", &TokenData{AccessToken: "t", ExpiresAt: time.Now().Add(time.Hour)}, true},
		{"within buffer", &TokenData{AccessToken: "t", ExpiresAt: time.Now().Add(3 * time.Minute)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.IsAccessTokenValid(); got != tt.want {
				t.Fatalf("IsAccessTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenData_IsRefreshTokenValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data *TokenData
		want bool
	}{
		{"nil", nil, false},
		{"empty", &TokenData{}, false},
		{"expired", &TokenData{RefreshToken: "r", RefreshExpAt: time.Now().Add(-time.Hour)}, false},
		{"valid", &TokenData{RefreshToken: "r", RefreshExpAt: time.Now().Add(time.Hour)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.IsRefreshTokenValid(); got != tt.want {
				t.Fatalf("IsRefreshTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenData_HasPersistentCode(t *testing.T) {
	t.Parallel()
	if (&TokenData{}).HasPersistentCode() {
		t.Fatal("expected false for empty")
	}
	if !(&TokenData{PersistentCode: "code"}).HasPersistentCode() {
		t.Fatal("expected true")
	}
	var nilData *TokenData
	if nilData.HasPersistentCode() {
		t.Fatal("expected false for nil")
	}
}
