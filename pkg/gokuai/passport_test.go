package gokuai

import (
	"net/url"
	"strings"
	"testing"
)

func TestGetPassportDomain_Default(t *testing.T) {
	t.Setenv("GOKUAI_PASSPORT_DOMAIN", "")
	if got := GetPassportDomain(); got != "https://passport.gokuai.com" {
		t.Fatalf("GetPassportDomain() = %q, want default https://passport.gokuai.com", got)
	}
}

func TestGetPassportDomain_EnvOverride(t *testing.T) {
	cases := map[string]string{
		"passport.example.com":  "https://passport.example.com",
		"http://passport.local": "http://passport.local",
		"https://p.x.com/":      "https://p.x.com",
		"https://p.x.com/sso":   "https://p.x.com/sso",
	}
	for env, want := range cases {
		t.Setenv("GOKUAI_PASSPORT_DOMAIN", env)
		if got := GetPassportDomain(); got != want {
			t.Errorf("GetPassportDomain(%q) = %q, want %q", env, got, want)
		}
	}
}

func TestBuildPassportLoginURL(t *testing.T) {
	t.Setenv("GOKUAI_PASSPORT_DOMAIN", "")

	redirect := "http://127.0.0.1:54321/callback"
	raw := BuildPassportLoginURL("the-client-id", redirect, "random-state", "")
	if !strings.HasPrefix(raw, "https://passport.gokuai.com/oauth/login?") {
		t.Fatalf("unexpected URL prefix: %s", raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("built URL is not parseable: %v", err)
	}
	q := u.Query()
	if got := q.Get("client_id"); got != "the-client-id" {
		t.Errorf("client_id = %q, want the-client-id", got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q, want code", got)
	}
	if got := q.Get("redirect_uri"); got != redirect {
		t.Errorf("redirect_uri = %q, want %q", got, redirect)
	}
	if got := q.Get("state"); got != "random-state" {
		t.Errorf("state = %q, want random-state", got)
	}
	if q.Has("scope") {
		t.Errorf("scope should be omitted when empty, got %q", q.Get("scope"))
	}
}

func TestBuildPassportLoginURL_WithScopeAndCustomDomain(t *testing.T) {
	t.Setenv("GOKUAI_PASSPORT_DOMAIN", "passport.intranet.local")
	raw := BuildPassportLoginURL("cid", "http://127.0.0.1:1/callback", "st", "all")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("built URL is not parseable: %v", err)
	}
	if u.Scheme != "https" || u.Host != "passport.intranet.local" {
		t.Fatalf("unexpected host: %s", raw)
	}
	if got := u.Query().Get("scope"); got != "all" {
		t.Errorf("scope = %q, want all", got)
	}
}
