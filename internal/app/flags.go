package app

import (
	"github.com/spf13/cobra"
)

// GlobalFlags contains the root-level persistent flags shared across the CLI.
type GlobalFlags struct {
	Debug   bool
	DryRun  bool
	Fields  string
	Format  string
	JQ      string
	Mock    bool
	Output  string
	Timeout int
	Token   string
	Verbose bool
	Yes     bool
}

func bindPersistentFlags(cmd *cobra.Command, flags *GlobalFlags) {
	cmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "显示调试日志")
	cmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "预览操作内容，不实际执行")
	_ = cmd.PersistentFlags().MarkHidden("dry-run")
	cmd.PersistentFlags().StringVar(&flags.Fields, "fields", "", "筛选输出字段 (逗号分隔, 如: name,id,status)")
	cmd.PersistentFlags().StringVarP(&flags.Format, "format", "f", "json", "输出格式: json|table|raw")
	cmd.PersistentFlags().StringVar(&flags.JQ, "jq", "", "jq 表达式过滤输出 (如: '.items[] | .name')")
	_ = cmd.PersistentFlags().MarkHidden("jq")
	cmd.PersistentFlags().BoolVar(&flags.Mock, "mock", false, "使用 Mock 数据 (开发调试用)")
	_ = cmd.PersistentFlags().MarkHidden("mock")
	cmd.PersistentFlags().StringVarP(&flags.Output, "output", "o", "", "Write command output to a file")
	_ = cmd.PersistentFlags().MarkHidden("output")
	cmd.PersistentFlags().IntVar(&flags.Timeout, "timeout", 30, "HTTP 请求超时时间 (秒)")
	cmd.PersistentFlags().StringVar(&flags.Token, "bearer-token", "", "Bearer Token 认证 (可使用环境变量 GOKUAI_BEARER_TOKEN)")
	cmd.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "显示详细日志")
	_ = cmd.PersistentFlags().MarkHidden("verbose")
	cmd.PersistentFlags().BoolVarP(&flags.Yes, "yes", "y", false, "跳过确认提示 (AI Agent 模式)")
}
