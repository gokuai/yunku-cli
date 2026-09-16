package gokuai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestIsTokenInvalidResponse(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`{"error_code":40102,"error_msg":"token invalid"}`, true},
		{`{"error_code":0}`, false},
		{`{"error_code":40103}`, false},
		{`{"foo":"bar"}`, false},
	}
	for _, c := range cases {
		if got := isTokenInvalidResponse([]byte(c.body)); got != c.want {
			t.Errorf("isTokenInvalidResponse(%s) = %v, want %v", c.body, got, c.want)
		}
	}
}

func TestUserClientRefreshesTokenOn40102(t *testing.T) {
	var calls int32
	var tokensSeen []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		tokensSeen = append(tokensSeen, token)
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			// First request: stale token.
			_ = json.NewEncoder(w).Encode(map[string]int{"error_code": TokenInvalidErrorCode})
		default:
			// Retry after refresh: succeed with the new token.
			_ = json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
		}
	}))
	defer srv.Close()

	var refreshCalls int32
	client := NewUserClient("old-token")
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()
	client.TokenRefresher = func() (string, error) {
		atomic.AddInt32(&refreshCalls, 1)
		return "new-token", nil
	}

	body, err := client.doUserRequest("/m-api/2/file/info", map[string]string{
		"mount_id": "1",
	}, "secret")
	if err != nil {
		t.Fatalf("doUserRequest returned error: %v", err)
	}
	if string(body) != `{"ok":"1"}` && string(body) != "{\"ok\":\"1\"}\n" {
		t.Fatalf("unexpected body: %s", body)
	}
	if atomic.LoadInt32(&refreshCalls) != 1 {
		t.Errorf("refresher called %d times, want 1", refreshCalls)
	}
	if len(tokensSeen) != 2 || tokensSeen[0] != "old-token" || tokensSeen[1] != "new-token" {
		t.Errorf("token sequence = %v, want [old-token new-token]", tokensSeen)
	}
	if client.AccessToken != "new-token" {
		t.Errorf("client.AccessToken = %q, want new-token", client.AccessToken)
	}
}

func TestUserClientNoRefreshWithoutCallback(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_ = json.NewEncoder(w).Encode(map[string]int{"error_code": TokenInvalidErrorCode})
	}))
	defer srv.Close()

	client := NewUserClient("token")
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	body, err := client.doUserRequest("/m-api/2/file/info", map[string]string{}, "secret")
	if err != nil {
		t.Fatalf("doUserRequest returned error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected exactly 1 call without refresher, got %d", calls)
	}
	if !isTokenInvalidResponse(body) {
		t.Errorf("expected 40102 body to be returned unchanged, got %s", body)
	}
}
