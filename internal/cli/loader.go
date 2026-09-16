package cli

import (
	"github.com/gokuai/yunku-cli/pkg/configmeta"
)

func init() {
	configmeta.Register(configmeta.ConfigItem{
		Name:         "YKC_CACHE_DIR",
		Category:     configmeta.CategoryCore,
		Description:  "覆盖缓存目录",
		DefaultValue: "~/.ykc/cache",
		Example:      "/tmp/ykc-cache",
	})
}

const (
	CacheDirEnv = "YKC_CACHE_DIR"
)

// CatalogFixtureEnv is the environment variable name for specifying a catalog fixture file.
// When set, the CLI loads tools from this file instead of fetching from the network.
const CatalogFixtureEnv = "YKC_CATALOG_FIXTURE"
