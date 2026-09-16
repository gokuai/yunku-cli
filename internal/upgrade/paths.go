package upgrade

import (
	"os"
	"path/filepath"
	"runtime"
)

// Permission constants following Unix best practices.
const (
	dirPermSecure  os.FileMode = 0o700
	dirPermShared  os.FileMode = 0o755
	filePermBinary os.FileMode = 0o755
	filePermConfig os.FileMode = 0o644
)

// EnsureUpgradeDirectories creates the directories needed for upgrade operations.
func EnsureUpgradeDirectories() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dirs := []struct {
		path string
		perm os.FileMode
	}{
		{filepath.Join(homeDir, ".ykc"), dirPermSecure},
		{filepath.Join(homeDir, ".ykc", "data"), dirPermSecure},
		{filepath.Join(homeDir, ".ykc", "data", "backups"), dirPermSecure},
		{filepath.Join(homeDir, ".ykc", "cache"), dirPermSecure},
		{filepath.Join(homeDir, ".ykc", "cache", "downloads"), dirPermSecure},
	}

	for _, d := range dirs {
		if err := ensureDir(d.path, d.perm); err != nil {
			return err
		}
	}
	return nil
}

// DownloadCacheDir returns the path for temporary downloads during upgrade.
func DownloadCacheDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".ykc", "cache", "downloads")
}

// CurrentBinaryPath returns the resolved path of the currently running binary.
func CurrentBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// BinaryName returns the platform-specific binary name.
func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "ykc.exe"
	}
	return "ykc"
}

func ensureDir(path string, perm os.FileMode) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, perm)
	}
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != perm {
		if info.Mode().Perm()&^perm != 0 {
			return os.Chmod(path, perm)
		}
	}
	return nil
}
