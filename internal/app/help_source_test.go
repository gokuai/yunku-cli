package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootHelpListsVisibleCommands(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(--help) error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"Usage:", "Commands:", "auth", "file", "ent", "doctor", "version"} {
		if !strings.Contains(got, want) {
			t.Fatalf("root help missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "MCP") {
		t.Fatalf("root help should not mention MCP:\n%s", got)
	}
}

func TestRootHelpCustomizationDoesNotAffectSubcommandHelp(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"auth", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(auth --help) error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Usage:") || !strings.Contains(got, "Available Commands:") || !strings.Contains(got, "Flags:") {
		t.Fatalf("subcommand help should still use cobra default sections:\n%s", got)
	}
}

func TestRootCommandRegistersUpgradeCommand(t *testing.T) {
	root := NewRootCommand()
	if cmd := lookupCommand(root, "upgrade"); cmd == nil {
		t.Fatal("upgrade command should be registered on root, but was not found")
	}
}

func lookupCommand(root *cobra.Command, path string) *cobra.Command {
	if root == nil || path == "" {
		return root
	}

	cmd := root
	for _, part := range strings.Fields(path) {
		found := false
		for _, child := range cmd.Commands() {
			if child.Name() == part {
				cmd = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return cmd
}
