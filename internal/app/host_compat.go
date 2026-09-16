package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// msysRoot caches the Windows path of the MSYS "/" mount point.
var msysRoot struct {
	sync.Once
	path string
}

// getMSYSRoot returns the Windows path that MSYS2 maps to "/",
// e.g. "E:/Git" for Git Bash installed at E:\Git.
// Returns empty string if not running under MSYS2.
func getMSYSRoot() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	msysRoot.Do(func() {
		out, err := exec.Command("cygpath", "-w", "/").Output()
		if err != nil {
			return
		}
		msysRoot.path = strings.TrimRight(strings.TrimSpace(string(out)), `\/`)
	})
	return msysRoot.path
}

// normalizeMSYSPath reverses MSYS2 automatic path conversion on Windows.
// Git Bash converts Unix-style paths like "/技术部" to Windows paths like "E:/Git/技术部".
// This function detects such conversions and restores the original Unix path.
// Non-affected paths are returned unchanged.
func normalizeMSYSPath(path string) string {
	if path == "" || runtime.GOOS != "windows" {
		return path
	}

	// Only process paths that look like Windows absolute paths (X:/...)
	msysPathRe := regexp.MustCompile(`^[A-Za-z]:[/\\]`)
	if !msysPathRe.MatchString(path) {
		return path
	}

	// Strategy 1: Use cygpath to find MSYS root and strip it
	root := getMSYSRoot()
	if root != "" {
		normalizedPath := filepath.ToSlash(path)
		normalizedRoot := filepath.ToSlash(root)
		if strings.HasPrefix(normalizedPath, normalizedRoot+"/") {
			result := normalizedPath[len(normalizedRoot):]
			if result == "" {
				return "/"
			}
			return result
		}
		if normalizedPath == normalizedRoot {
			return "/"
		}
	}

	// Strategy 2: Heuristic - extract the last path segment as /{segment}
	// This handles cases where cygpath is unavailable.
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) >= 3 {
		return "/" + parts[len(parts)-1]
	}

	return path
}

// normalizeMSYSPaths normalizes pipe-separated path values (e.g. "/a|/b").
// Each segment is individually normalized via normalizeMSYSPath.
func normalizeMSYSPaths(paths string) string {
	if paths == "" || runtime.GOOS != "windows" {
		return paths
	}
	parts := strings.Split(paths, "|")
	for i, p := range parts {
		parts[i] = normalizeMSYSPath(p)
	}
	return strings.Join(parts, "|")
}

// normalizeMSYSPathsComma normalizes comma-separated path values (e.g. "/a,/b").
// Each segment is individually normalized via normalizeMSYSPath.
func normalizeMSYSPathsComma(paths string) string {
	if paths == "" || runtime.GOOS != "windows" {
		return paths
	}
	parts := strings.Split(paths, ",")
	for i, p := range parts {
		parts[i] = normalizeMSYSPath(p)
	}
	return strings.Join(parts, ",")
}

// expandHomePath expands a leading "~" to the current user's home directory.
// Only "~" and "~/..." forms are expanded; "~user" is left unchanged.
// Returns the original path if the home directory cannot be determined.
func expandHomePath(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
