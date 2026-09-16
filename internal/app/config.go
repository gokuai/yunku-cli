package app

import (
	"os"
	"path/filepath"

	"github.com/gokuai/yunku-cli/pkg/configmeta"
	"github.com/gokuai/yunku-cli/pkg/edition"
)

func init() {
	configmeta.Register(configmeta.ConfigItem{
		Name:         "YKC_CONFIG_DIR",
		Category:     configmeta.CategoryCore,
		Description:  "覆盖默认配置目录 (~/.ykc)",
		DefaultValue: "~/.ykc",
		Example:      "/opt/ykc/config",
	})
}

// Build-time variables injected via ldflags when available.
var (
	buildTime = "unknown"
	gitCommit = "unknown"
)

func defaultConfigDir() string {
	if envDir := os.Getenv("YKC_CONFIG_DIR"); envDir != "" {
		return envDir
	}
	if fn := edition.Get().ConfigDir; fn != nil {
		return fn()
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return exeRelativeConfigDir()
	}
	return filepath.Join(homeDir, ".ykc")
}

func exeRelativeConfigDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ".ykc"
	}
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		realPath = exePath
	}
	return filepath.Join(filepath.Dir(realPath), ".ykc")
}
