package app

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func configureRootHelp(root *cobra.Command) {
	if root == nil {
		return
	}

	defaultHelpFunc := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != root {
			defaultHelpFunc(cmd, args)
			return
		}
		renderRootHelp(root)
	})
}

func renderRootHelp(root *cobra.Command) {
	commands := visibleRootCommands(root)
	w := root.OutOrStdout()
	binName := root.Name()

	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintf(w, "  %s <command> [flags]\n", binName)
	_, _ = fmt.Fprintln(w)
	if len(commands) > 0 {
		_, _ = fmt.Fprintln(w, "Commands:")
		_, _ = fmt.Fprintln(w)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, cmd := range commands {
			_, _ = fmt.Fprintf(tw, "  %s\t%s\n", cmd.Name(), strings.TrimSpace(cmd.Short))
		}
		_ = tw.Flush()
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintf(w, "Use \"%s <command> --help\" for more information about a command.\n", binName)
}

func visibleRootCommands(root *cobra.Command) []*cobra.Command {
	if root == nil {
		return nil
	}

	commands := make([]*cobra.Command, 0)
	for _, cmd := range root.Commands() {
		if cmd == nil || cmd.Hidden {
			continue
		}
		commands = append(commands, cmd)
	}
	return commands
}
