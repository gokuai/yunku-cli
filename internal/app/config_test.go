package app

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigDirUsesHomeDirectoryInOSSMode(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", homeDir)
	t.Setenv("YKC_CONFIG_DIR", "")

	got := defaultConfigDir()
	want := filepath.Join(homeDir, ".ykc")
	if got != want {
		t.Fatalf("defaultConfigDir() = %q, want %q", got, want)
	}
}
