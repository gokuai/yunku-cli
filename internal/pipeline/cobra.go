package pipeline

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RunPreParse resolves the target command from the raw args, extracts
// flag names from the Cobra command tree, and runs all PreParse
// handlers. The corrected args are set back on the root command via
// SetArgs so that Cobra's subsequent ExecuteC uses the corrected
// values.
//
// If the target command cannot be resolved (e.g. the user typed a
// non-existent command), PreParse is skipped silently and Cobra will
// handle the error.
func RunPreParse(root *cobra.Command, engine *Engine) {
	if engine == nil || !engine.HasHandlers(PreParse) {
		return
	}

	rawArgs := os.Args[1:]
	if len(rawArgs) == 0 {
		return
	}

	// Traverse the command tree to find the target command.
	target, _, err := root.Traverse(rawArgs)
	if err != nil || target == nil {
		return
	}

	// Build FlagInfo from the target command's registered flags.
	flagInfos := FlagInfoFromCommand(target)
	if len(flagInfos) == 0 {
		return
	}

	ctx := &Context{
		Args:      append([]string{}, rawArgs...),
		FlagSpecs: flagInfos,
	}

	if err := engine.RunPhase(PreParse, ctx); err != nil {
		slog.Debug("pipeline pre-parse", "error", err)
		return
	}

	// Only set corrected args if PreParse actually changed something.
	if len(ctx.Corrections) > 0 {
		root.SetArgs(ctx.Args)
		for _, c := range ctx.Corrections {
			slog.Debug("pipeline correction",
				"handler", c.Handler,
				"kind", c.Kind,
				"field", c.Field,
				"original", c.Original,
				"corrected", c.Corrected,
			)
		}
	}
}

// FlagInfoFromCommand extracts FlagInfo entries from a Cobra
// command's registered flags (both local and inherited).
func FlagInfoFromCommand(cmd *cobra.Command) []FlagInfo {
	if cmd == nil {
		return nil
	}

	seen := make(map[string]bool)
	var infos []FlagInfo

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		infos = append(infos, FlagInfo{
			Name:         f.Name,
			PropertyName: f.Name,
			Type:         f.Value.Type(),
		})
	})

	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		infos = append(infos, FlagInfo{
			Name:         f.Name,
			PropertyName: f.Name,
			Type:         f.Value.Type(),
		})
	})

	return infos
}
