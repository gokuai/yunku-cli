package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureUpgradeDirectories(t *testing.T) {
	// This test verifies the structure is correct without actually modifying $HOME
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	err = EnsureUpgradeDirectories()
	if err != nil {
		t.Fatalf("EnsureUpgradeDirectories() error = %v", err)
	}

	expectedDirs := []string{
		filepath.Join(homeDir, ".ykc"),
		filepath.Join(homeDir, ".ykc", "data"),
		filepath.Join(homeDir, ".ykc", "data", "backups"),
		filepath.Join(homeDir, ".ykc", "cache"),
		filepath.Join(homeDir, ".ykc", "cache", "downloads"),
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("directory %s should exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", d)
		}
	}
}

func TestDownloadCacheDir(t *testing.T) {
	dir := DownloadCacheDir()
	if dir == "" {
		t.Error("DownloadCacheDir() returned empty")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("DownloadCacheDir() = %q, want absolute path", dir)
	}
}

func TestBinaryName(t *testing.T) {
	name := BinaryName()
	if name != "ykc" && name != "ykc.exe" {
		t.Errorf("BinaryName() = %q, want ykc or ykc.exe", name)
	}
}
