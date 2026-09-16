package gokuai

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestBuildSSOURL(t *testing.T) {
	token := "test-access-token"
	target := "https://yk3.gokuai.com/some/edit/page?file=1"
	secret := "test-secret"

	raw, err := BuildSSOURL(token, target, secret)
	if err != nil {
		t.Fatalf("BuildSSOURL returned error: %v", err)
	}

	if !strings.HasPrefix(raw, GetAPIHost()+"/account/sso?") {
		t.Fatalf("unexpected SSO url prefix: %s", raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := u.Query()

	for _, key := range []string{"token", "url", "n", "t", "s"} {
		if q.Get(key) == "" {
			t.Errorf("missing query param %q in %s", key, raw)
		}
	}
	if q.Get("token") != token {
		t.Errorf("token = %q, want %q", q.Get("token"), token)
	}
	if q.Get("url") != target {
		t.Errorf("url = %q, want %q", q.Get("url"), target)
	}
	if _, err := strconv.ParseUint(q.Get("n"), 10, 64); err != nil {
		t.Errorf("n = %q is not purely numeric: %v", q.Get("n"), err)
	}

	// The sign must verify against the same Sign convention the server uses.
	params := map[string]string{
		"token": q.Get("token"),
		"url":   q.Get("url"),
		"n":     q.Get("n"),
		"t":     q.Get("t"),
	}
	expected := Sign(params, secret)
	if q.Get("s") != expected {
		t.Errorf("sign = %q, want %q", q.Get("s"), expected)
	}
}
