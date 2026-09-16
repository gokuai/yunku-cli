package gokuai

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestComputeFileSHA1 verifies that SHA1 is computed correctly.
func TestComputeFileSHA1(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Known content → known SHA1
	// echo -n "hello" | sha1sum → aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d
	content := []byte("hello")
	fpath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fpath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	hash, size, err := ComputeFileSHA1(fpath)
	if err != nil {
		t.Fatalf("ComputeFileSHA1 error: %v", err)
	}
	if hash != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Errorf("wrong SHA1: got %s", hash)
	}
	if size != int64(len(content)) {
		t.Errorf("wrong size: got %d, want %d", size, len(content))
	}
}

// TestComputeFileSHA1_Missing checks error on missing file.
func TestComputeFileSHA1_Missing(t *testing.T) {
	t.Parallel()
	_, _, err := ComputeFileSHA1("/nonexistent/path/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestNormalizeUploadServerURL checks URL normalisation.
func TestNormalizeUploadServerURL(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"", ""},
		{"upload.example.com", "http://upload.example.com"},
		{"http://upload.example.com/", "http://upload.example.com"},
		{"https://upload.example.com", "https://upload.example.com"},
		{"upload.example.com,upload2.example.com", "http://upload.example.com"},
		{"  http://upload.example.com  ", "http://upload.example.com"},
	}
	for _, c := range cases {
		got := normalizeUploadServerURL(c.input)
		if got != c.want {
			t.Errorf("normalizeUploadServerURL(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestChunksRequired verifies chunk-count arithmetic.
func TestChunksRequired(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size, chunk, want int64
	}{
		{0, 4 * 1024 * 1024, 1},
		{1, 4 * 1024 * 1024, 1},
		{4 * 1024 * 1024, 4 * 1024 * 1024, 1},
		{4*1024*1024 + 1, 4 * 1024 * 1024, 2},
		{8 * 1024 * 1024, 4 * 1024 * 1024, 2},
		{10, 3, 4},
	}
	for _, c := range cases {
		got := chunksRequired(c.size, c.chunk)
		if got != c.want {
			t.Errorf("chunksRequired(%d, %d) = %d, want %d", c.size, c.chunk, got, c.want)
		}
	}
}

// TestUploadInit_Instant202 checks that HTTP 202 returns empty session (instant done).
func TestUploadInit_Instant202(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload_init" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	session, err := c.doUploadInit(srv.URL, "file.txt", "pathhash", "filehash", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "" {
		t.Errorf("expected empty session for 202, got %q", session)
	}
}

// TestUploadInit_Session200 checks that HTTP 200 with JSON returns the session.
func TestUploadInit_Session200(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"session":"sess-abc123"}`))
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	session, err := c.doUploadInit(srv.URL, "file.txt", "ph", "fh", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "sess-abc123" {
		t.Errorf("got session %q, want sess-abc123", session)
	}
}

// TestUploadInit_Error5xx checks that a server error is propagated.
func TestUploadInit_Error5xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	_, err := c.doUploadInit(srv.URL, "file.txt", "ph", "fh", 100)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// TestUploadParts verifies that chunks are sent with correct headers and ranges.
func TestUploadParts(t *testing.T) {
	t.Parallel()

	const chunkSize = 5
	content := []byte("hello world!") // 12 bytes → 3 chunks: 0-4, 5-9, 10-11

	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.bin")
	if err := os.WriteFile(fpath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		ranges = append(ranges, r.Header.Get("x-gk-upload-range"))
		if r.Header.Get("x-gk-upload-session") != "mysession" {
			t.Errorf("wrong session header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	err := c.doUploadParts(srv.URL, fpath, int64(len(content)), chunkSize, "mysession")
	if err != nil {
		t.Fatalf("doUploadParts error: %v", err)
	}

	wantRanges := []string{"0-4", "5-9", "10-11"}
	if len(ranges) != len(wantRanges) {
		t.Fatalf("got %d chunks, want %d", len(ranges), len(wantRanges))
	}
	for i, r := range ranges {
		if r != wantRanges[i] {
			t.Errorf("chunk %d range = %q, want %q", i, r, wantRanges[i])
		}
	}
}

// TestUploadFinish_OK verifies successful finish call.
func TestUploadFinish_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload_finish" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-gk-upload-session") != "sess" {
			t.Errorf("wrong session header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	if err := c.doUploadFinish(srv.URL, "sess"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestUploadFinish_Error ensures non-2xx triggers an error.
func TestUploadFinish_Error(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	c := &OrgClient{OrgClientID: "test", HTTPClient: &http.Client{}}
	if err := c.doUploadFinish(srv.URL, "sess"); err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

// ── UserClient upload tests ───────────────────────────────────────────────────

// TestUserUploadInit_UsesTokenHeader verifies that the user upload_init sends
// x-gk-token (not ?org_client_id) and returns the session on HTTP 200.
func TestUserUploadInit_UsesTokenHeader(t *testing.T) {
	t.Parallel()

	const wantToken = "my-access-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Must have x-gk-token, not org_client_id in URL
		if r.Header.Get("x-gk-token") != wantToken {
			t.Errorf("x-gk-token = %q, want %q", r.Header.Get("x-gk-token"), wantToken)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query params for user upload_init: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"session":"user-sess-xyz"}`))
	}))
	defer srv.Close()

	c := &UserClient{AccessToken: wantToken, HTTPClient: &http.Client{}}
	session, err := c.doUserUploadInit(srv.URL, "report.pdf", "ph", "fh", 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "user-sess-xyz" {
		t.Errorf("got session %q, want user-sess-xyz", session)
	}
}

// TestUserUploadInit_Instant202 checks that HTTP 202 returns empty session.
func TestUserUploadInit_Instant202(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := &UserClient{AccessToken: "tok", HTTPClient: &http.Client{}}
	session, err := c.doUserUploadInit(srv.URL, "f.txt", "ph", "fh", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "" {
		t.Errorf("expected empty session for 202, got %q", session)
	}
}

// TestUserUploadParts_ChunkRanges verifies chunking ranges for UserClient.
func TestUserUploadParts_ChunkRanges(t *testing.T) {
	t.Parallel()

	const chunkSize = 4
	content := []byte("abcdefghij") // 10 bytes → 3 chunks: 0-3, 4-7, 8-9

	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.bin")
	if err := os.WriteFile(fpath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	var ranges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ranges = append(ranges, r.Header.Get("x-gk-upload-range"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &UserClient{AccessToken: "tok", HTTPClient: &http.Client{}}
	err := c.doUserUploadParts(srv.URL, fpath, int64(len(content)), chunkSize, "sess")
	if err != nil {
		t.Fatalf("doUserUploadParts error: %v", err)
	}

	wantRanges := []string{"0-3", "4-7", "8-9"}
	if len(ranges) != len(wantRanges) {
		t.Fatalf("got %d chunks, want %d", len(ranges), len(wantRanges))
	}
	for i, r := range ranges {
		if r != wantRanges[i] {
			t.Errorf("chunk %d range = %q, want %q", i, r, wantRanges[i])
		}
	}
}
