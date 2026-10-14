// Package browser opens the user's default web browser in a cross-platform
// way (used by the OAuth2 browser login flow).
package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open launches the system default browser at rawURL.
//
// The BROWSER environment variable takes precedence when set (same
// convention as gh/git): it is split into a command plus arguments and the
// URL is appended. This also keeps browser login usable on headless hosts
// (e.g. BROWSER=/bin/true).
func Open(rawURL string) error {
	args, err := browserArgs(rawURL)
	if err != nil {
		return err
	}
	return exec.Command(args[0], args[1:]...).Start()
}

// browserArgs resolves the command (and arguments) used to open rawURL.
// BROWSER takes precedence when set; otherwise the platform default is used.
func browserArgs(rawURL string) ([]string, error) {
	if override := strings.TrimSpace(os.Getenv("BROWSER")); override != "" {
		return append(strings.Fields(override), rawURL), nil
	}
	return openArgs(runtime.GOOS, rawURL)
}

// openArgs returns the command used to open a URL on the given GOOS.
// It is split out from Open so the platform mapping can be unit tested
// without launching anything.
func openArgs(goos, rawURL string) ([]string, error) {
	switch goos {
	case "darwin":
		return []string{"open", rawURL}, nil
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", rawURL}, nil
	case "linux", "freebsd", "netbsd", "openbsd":
		return []string{"xdg-open", rawURL}, nil
	default:
		return nil, fmt.Errorf("当前平台 (%s) 不支持自动打开浏览器", goos)
	}
}
