package config

import (
	"testing"
	"time"
)

func TestDirPerm(t *testing.T) {
	t.Parallel()
	if DirPerm != 0o700 {
		t.Fatalf("DirPerm = %o, want 700", DirPerm)
	}
}

func TestFilePerm(t *testing.T) {
	t.Parallel()
	if FilePerm != 0o600 {
		t.Fatalf("FilePerm = %o, want 600", FilePerm)
	}
}

func TestHTTPTimeout(t *testing.T) {
	t.Parallel()
	if HTTPTimeout <= 0 {
		t.Fatalf("HTTPTimeout = %v, want positive", HTTPTimeout)
	}
	if HTTPTimeout != 30*time.Second {
		t.Fatalf("HTTPTimeout = %v, want 30s", HTTPTimeout)
	}
}

func TestMaxResponseBodySize(t *testing.T) {
	t.Parallel()
	want := 10 * 1024 * 1024
	if MaxResponseBodySize != want {
		t.Fatalf("MaxResponseBodySize = %d, want %d (10MB)", MaxResponseBodySize, want)
	}
}

func TestDefaultPartition(t *testing.T) {
	t.Parallel()
	if DefaultPartition != "default/default" {
		t.Fatalf("DefaultPartition = %q, want %q", DefaultPartition, "default/default")
	}
}
