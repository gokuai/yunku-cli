package app

import (
	"sort"

	"github.com/gokuai/yunku-cli/internal/cobracmd"
	"github.com/gokuai/yunku-cli/internal/executor"
	"github.com/gokuai/yunku-cli/internal/helpers"
	"github.com/gokuai/yunku-cli/pkg/edition"
	"github.com/spf13/cobra"
)

func newLegacyPublicCommands(_ interface{}, runner executor.Runner) []*cobra.Command {
	if fn := edition.Get().StaticServers; fn != nil {
		// Static servers provided by the edition hook — commands are registered via
		// RegisterExtraCommands; we only add the open-source helpers here.
		commands := helpers.NewPublicCommands(runner)
		return mergeTopLevelCommands(commands)
	}

	commands := helpers.NewPublicCommands(runner)
	return mergeTopLevelCommands(commands)
}

func newLegacyHiddenCommands(_ executor.Runner) []*cobra.Command {
	return nil
}

func mergeTopLevelCommands(commands []*cobra.Command) []*cobra.Command {
	byName := make(map[string]*cobra.Command, len(commands))
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		name := cmd.Name()
		if name == "" {
			continue
		}
		if existing, ok := byName[name]; ok {
			cobracmd.MergeCommandTree(existing, cmd)
			continue
		}
		byName[name] = cmd
	}

	out := make([]*cobra.Command, 0, len(byName))
	for _, cmd := range byName {
		out = append(out, cmd)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Use < out[j].Use
	})
	return out
}
