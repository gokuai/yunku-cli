package app

import (
	"github.com/gokuai/yunku-cli/internal/cobracmd"
	"github.com/spf13/cobra"
)

func newPlaceholderParent(use, short string, children ...*cobra.Command) *cobra.Command {
	return cobracmd.NewPlaceholderParent(use, short, children...)
}
