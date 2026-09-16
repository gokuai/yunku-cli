package app

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const requiredFlagSuffix = " (必需)"

// annotateRequiredFlagUsages walks the whole command tree and appends
// " (必需)" to the usage text of every flag marked required via
// MarkFlagRequired / MarkPersistentFlagRequired, so help output indicates
// which flags are mandatory without repeating the marker in each definition.
func annotateRequiredFlagUsages(cmd *cobra.Command) {
	// InheritedFlags() triggers cobra's persistent-flag merge, so Flags()
	// afterwards contains both local and inherited flags of the command.
	_ = cmd.InheritedFlags()
	cmd.Flags().VisitAll(appendRequiredMarker)
	for _, child := range cmd.Commands() {
		annotateRequiredFlagUsages(child)
	}
}

func appendRequiredMarker(f *pflag.Flag) {
	values, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	if !ok || len(values) == 0 || values[0] != "true" {
		return
	}
	// Skip hidden flags (e.g. command-specific overrides).
	if f.Hidden {
		return
	}
	// Avoid duplicate markers (half/full-width variants, any position).
	if strings.Contains(f.Usage, "(必需)") || strings.Contains(f.Usage, "（必需）") {
		return
	}
	f.Usage += requiredFlagSuffix
}
