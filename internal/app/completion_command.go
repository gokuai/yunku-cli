package app

import (
	"github.com/spf13/cobra"
)

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "生成 Shell 自动补全脚本",
		Long: `生成指定 Shell 的自动补全脚本。

Zsh (推荐):
  # 将补全脚本写入 fpath 目录
  ykc completion zsh > "${fpath[1]}/_ykc"
  # 或者加入 .zshrc
  source <(ykc completion zsh)

Bash:
  # Linux
  ykc completion bash > /etc/bash_completion.d/ykc
  # macOS (需安装 bash-completion)
  ykc completion bash > $(brew --prefix)/etc/bash_completion.d/ykc

Fish:
  ykc completion fish > ~/.config/fish/completions/ykc.fish`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return nil
			}
		},
	}
}
