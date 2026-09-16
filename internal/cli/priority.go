package cli

import (
	"github.com/gokuai/yunku-cli/internal/cobracmd"
	"github.com/spf13/cobra"
)

// SetOverridePriority delegates to cobracmd.SetOverridePriority.
func SetOverridePriority(cmd *cobra.Command, priority int) {
	cobracmd.SetOverridePriority(cmd, priority)
}

// OverridePriority delegates to cobracmd.OverridePriority.
func OverridePriority(cmd *cobra.Command) int {
	return cobracmd.OverridePriority(cmd)
}
