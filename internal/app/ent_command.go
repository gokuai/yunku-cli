package app

import (
	"fmt"
	"regexp"
	"strconv"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

// newEntCommand creates the enterprise management command group
func newEntCommand(flags *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ent",
		Short: "企业管理操作",
		Long: `企业管理操作，包括成员管理、部门管理、同步操作、管理日志等。
企业凭证可通过 --client-id/--client-secret flag 传入，或设置环境变量 GOKUAI_CLIENT_ID / GOKUAI_CLIENT_SECRET`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Apply enterprise client credentials from ent-scoped flags.
			// cobra.EnableTraverseRunHooks=true ensures the root's PersistentPreRunE
			// (log level, bearer token, output sink) has already run before this.
			clientID, _ := cmd.Flags().GetString("client-id")
			clientSecret, _ := cmd.Flags().GetString("client-secret")
			if clientID != "" || clientSecret != "" {
				SetEntClientCredentials(clientID, clientSecret)
			}
			return nil
		},
	}

	// ent 命令树隐藏 --bearer-token（企业凭证通过 client-id/client-secret 或环境变量传入）
	var bearerTokenShadow string
	cmd.PersistentFlags().StringVar(&bearerTokenShadow, "bearer-token", "", "")
	_ = cmd.PersistentFlags().MarkHidden("bearer-token")

	cmd.PersistentFlags().String("client-id", "", "企业应用 Client ID (可使用环境变量 GOKUAI_CLIENT_ID)")
	cmd.PersistentFlags().String("client-secret", "", "企业应用 Client Secret (可使用环境变量 GOKUAI_CLIENT_SECRET)")

	cmd.AddCommand(
		newEntOAuthCommand(),
		newEntMemberCommand(),
		newEntGroupCommand(),
		newEntSyncCommand(),
		newEntLogCommand(),
		newEntRolesCommand(),
		newEntOrgCommand(),
		newEntFileCommand(),
	)

	return cmd
}

// === OAuth 子命令 ===

func newEntOAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oauth",
		Short: "个人授权管理",
	}

	cmd.AddCommand(
		newEntOAuthGetCommand(),
		newEntOAuthDelCommand(),
	)

	return cmd
}

func newEntOAuthGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "获取个人授权",
		Example: `  ykc ent oauth get`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			resp, err := client.GetClientOAuth()
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newEntOAuthDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "删除个人授权",
		Example: `  ykc ent oauth delete`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			if err := client.DelClientOAuth(); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	return cmd
}

// === 成员管理子命令 ===

func newEntMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "成员管理",
	}

	cmd.AddCommand(
		newEntMemberListCommand(),
		newEntMemberInfoCommand(),
		newEntMemberAddCommand(),
		newEntMemberSetCommand(),
		newEntMemberDelCommand(),
		newEntMemberDelGroupCommand(),
	)

	return cmd
}

func newEntMemberListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "成员列表",
		Example: `  ykc ent member list
  ykc ent member list --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			if start < 0 {
				return apperrors.NewValidation("--start 必须是0或正整数")
			}

			params := map[string]string{
				"start": fmt.Sprintf("%d", start),
				"size":  fmt.Sprintf("%d", size),
			}

			resp, err := client.GetMembers(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	return cmd
}

func newEntMemberInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "成员信息",
		Example: `  ykc ent member info --account zhangsan
  ykc ent member info --member-id 123 --show-groups --show-orgs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("member-id"); v != "" {
				params["member_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			if v, _ := cmd.Flags().GetString("email"); v != "" {
				params["email"] = v
			}
			if v, _ := cmd.Flags().GetBool("show-groups"); v {
				params["show_groups"] = "1"
			}
			if v, _ := cmd.Flags().GetBool("show-orgs"); v {
				params["show_orgs"] = "1"
			}

			resp, err := client.GetMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("member-id", "", "成员ID")
	cmd.Flags().String("out-id", "", "外部帐号ID")
	cmd.Flags().String("account", "", "帐号")
	cmd.Flags().String("email", "", "邮箱")
	cmd.Flags().Bool("show-groups", false, "显示所属部门")
	cmd.Flags().Bool("show-orgs", false, "显示所属库")
	return cmd
}

func newEntMemberAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "添加成员",
		Example: `  ykc ent member add --account zhangsan@company.com --password Pass123 --name "张三"
  ykc ent member add --account zhangsan@company.com --password Zs123456 --name "张三" --group-path "/技术部"
  ykc ent member add --account zhangsan@company.com --password Pass123 --name "张三" --create-personal-org`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["password"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				matched, _ := regexp.MatchString(`^[\x{4e00}-\x{9fa5}a-zA-Z0-9_-]+$`, v)
				if !matched {
					return apperrors.NewValidation(fmt.Sprintf("成员名称格式错误，仅支持中文汉字、数字、英文字母、减号或下划线: %s", v))
				}
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetString("phone"); v != "" {
				matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, v)
				if !matched {
					return apperrors.NewValidation(fmt.Sprintf("手机号码格式错误: %s", v))
				}
				params["member_phone"] = v
			}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				params["member_title"] = v
			}
			if v, _ := cmd.Flags().GetString("group-path"); v != "" {
				params["group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetInt("state"); v >= 0 {
				params["state"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetBool("create-personal-org"); v {
				params["create_personal_org"] = "1"
			}

			resp, err := client.AddMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("account", "", "帐号 (必需)")
	cmd.Flags().String("password", "", "初始密码 (必需)")
	cmd.Flags().String("name", "", "成员名称 (必需)")
	cmd.Flags().String("phone", "", "联系电话")
	cmd.Flags().String("title", "", "职位")
	cmd.Flags().String("group-path", "", "成员所在部门路径")
	cmd.Flags().Int("state", -1, "状态: 1启用, 0禁用")
	cmd.Flags().Bool("create-personal-org", false, "初始化个人库 (不添加此参数则不初始化)")
	cmd.MarkFlagRequired("account")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newEntMemberSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "修改成员",
		Example: `  ykc ent member set --account zhangsan --name "张三丰"
  ykc ent member set --account zhangsan --state 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				matched, _ := regexp.MatchString(`^[\x{4e00}-\x{9fa5}a-zA-Z0-9_-]+$`, v)
				if !matched {
					return apperrors.NewValidation(fmt.Sprintf("成员名称格式错误，仅支持中文汉字、数字、英文字母、减号或下划线: %s", v))
				}
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetString("phone"); v != "" {
				matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, v)
				if !matched {
					return apperrors.NewValidation(fmt.Sprintf("手机号码格式错误: %s", v))
				}
				params["member_phone"] = v
			}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				params["member_title"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["password"] = v
			}
			if v, _ := cmd.Flags().GetInt("state"); v >= 0 {
				params["state"] = fmt.Sprintf("%d", v)
			}

			resp, err := client.SetMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("account", "", "帐号 (必需)")
	cmd.Flags().String("name", "", "成员名称")
	cmd.Flags().String("phone", "", "手机号")
	cmd.Flags().String("title", "", "职位")
	cmd.Flags().String("password", "", "密码")
	cmd.Flags().Int("state", -1, "状态: 1启用, 0禁用")
	cmd.MarkFlagRequired("account")
	return cmd
}

func newEntMemberDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "删除成员",
		Example: `  ykc ent member delete --members "zhangsan|lisi"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			members, _ := cmd.Flags().GetString("members")

			if err := client.DelMember(members); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("members", "", "成员帐号，多个用竖号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntMemberDelGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-group",
		Short: "删除成员的所属部门",
		Example: `  ykc ent member del-group --members "zhangsan@company.com"
  ykc ent member del-group --members "zhangsan@company.com,lisi@company.com"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			members, _ := cmd.Flags().GetString("members")

			if members == "" {
				return apperrors.NewValidation("请提供必需参数: --members 成员帐号\n用法: ykc ent member del-group --members \"zhangsan@company.com,lisi@company.com\"")
			}

			if err := client.DelMemberGroup(members); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("members", "", "成员帐号，多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

// === 部门管理子命令 ===

func newEntGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "部门管理",
	}

	cmd.AddCommand(
		newEntGroupListCommand(),
		newEntGroupAddCommand(),
		newEntGroupSetCommand(),
		newEntGroupDelCommand(),
		newEntGroupMemberListCommand(),
		newEntGroupAddMemberCommand(),
		newEntGroupDelMemberCommand(),
	)

	return cmd
}

func newEntGroupListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "部门列表",
		Example: `  ykc ent group list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			resp, err := client.GetGroups()
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newEntGroupAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "添加部门",
		Example: `  ykc ent group add --name "技术部"
  ykc ent group add --name "前端组" --parent-path "/技术部"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["group_name"] = v
			}
			if v, _ := cmd.Flags().GetString("parent-path"); v != "" {
				params["parent_group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetInt("orderby"); v > 0 {
				params["orderby"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetInt("state"); v >= 0 {
				params["state"] = fmt.Sprintf("%d", v)
			}

			resp, err := client.AddGroup(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("name", "", "部门名称 (必需)")
	cmd.Flags().String("parent-path", "", "父部门路径，空表示根部门")
	cmd.Flags().Int("orderby", 0, "排序值 (需大于等于0)")
	cmd.Flags().Int("state", -1, "状态: 1启用, 0禁用")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newEntGroupSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "修改部门",
		Example: `  ykc ent group set --path "/技术部" --name "研发部"
  ykc ent group set --path "/技术部" --parent-path "/产品中心"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("path"); v != "" {
				params["group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["group_name"] = v
			}
			if v, _ := cmd.Flags().GetString("parent-path"); v != "" {
				params["parent_group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetInt("orderby"); v > 0 {
				params["orderby"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetInt("state"); v >= 0 {
				params["state"] = fmt.Sprintf("%d", v)
			}

			if err := client.SetGroup(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("path", "", "部门路径 (必需)")
	cmd.Flags().String("name", "", "部门名称")
	cmd.Flags().String("parent-path", "", "新的父部门路径")
	cmd.Flags().Int("orderby", 0, "排序值 (需大于等于0)")
	cmd.Flags().Int("state", -1, "状态: 1启用, 0禁用")
	cmd.MarkFlagRequired("path")
	return cmd
}

func newEntGroupDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "删除部门",
		Example: `  ykc ent group delete --groups "/技术部"
  ykc ent group delete --groups "/技术部,/产品部"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			groups, _ := cmd.Flags().GetString("groups")

			if err := client.DelGroup(normalizeMSYSPathsComma(groups)); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("groups", "", "部门路径，多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("groups")
	return cmd
}

func newEntGroupMemberListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "部门中成员列表",
		Example: `  ykc ent group members --group-id 1
  ykc ent group members --group-path "/技术部" --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			if start < 0 {
				return apperrors.NewValidation("--start 必须是0或正整数")
			}
			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-path"); v != "" {
				params["group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetBool("show-child"); v {
				params["show_child"] = "1"
			}
			params["start"] = fmt.Sprintf("%d", start)
			params["size"] = fmt.Sprintf("%d", size)

			resp, err := client.GetGroupMembers(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("group-id", "", "部门ID")
	cmd.Flags().String("out-id", "", "部门外部ID")
	cmd.Flags().String("group-path", "", "部门路径")
	cmd.Flags().Bool("show-child", false, "显示子部门成员")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	return cmd
}

func newEntGroupAddMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member",
		Short: "添加部门成员",
		Example: `  ykc ent group add-member --path "/技术部" --members "zhangsan@company.com"
  ykc ent group add-member --path "/技术部" --members "zhangsan@company.com,lisi@company.com"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("path"); v != "" {
				params["group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("members"); v != "" {
				params["members"] = v
			}

			if err := client.AddGroupMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("path", "", "部门路径 (必需)")
	cmd.Flags().String("members", "", "成员帐号，多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("path")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntGroupDelMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-member",
		Short: "删除部门成员",
		Example: `  ykc ent group del-member --path "/技术部" --members "zhangsan@company.com"
  ykc ent group del-member --path "/技术部" --members "zhangsan@company.com,lisi@company.com"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("path"); v != "" {
				params["group_path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("members"); v != "" {
				params["members"] = v
			}

			if err := client.DelGroupMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("path", "", "部门路径 (必需)")
	cmd.Flags().String("members", "", "成员帐号，多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("path")
	cmd.MarkFlagRequired("members")
	return cmd
}

// === 同步操作子命令 ===

func newEntSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "同步操作",
	}

	cmd.AddCommand(
		newEntSyncAddMemberCommand(),
		newEntSyncDelMemberCommand(),
		newEntSyncAddGroupCommand(),
		newEntSyncDelGroupCommand(),
		newEntSyncAddGroupMemberCommand(),
		newEntSyncDelGroupMemberCommand(),
		newEntSyncDelMemberGroupCommand(),
		newEntSyncGetMemberByOutIDCommand(),
		newEntSyncGetGroupByOutIDCommand(),
		newEntSyncAddAdminCommand(),
	)

	return cmd
}

func newEntSyncAddMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member",
		Short: "添加或修改同步成员",
		Example: `  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --group-out-ids "dept001,dept002"
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --password "Zs123456" --state 1
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --leader-out-id "emp002" --group-out-ids "dept001"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			if v, _ := cmd.Flags().GetString("email"); v != "" {
				if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, v); !matched {
					return apperrors.NewValidation("--email 格式不正确")
				}
				params["member_email"] = v
			}
			if v, _ := cmd.Flags().GetString("phone"); v != "" {
				matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, v)
				if !matched {
					return apperrors.NewValidation(fmt.Sprintf("手机号码格式错误: %s", v))
				}
				params["member_phone"] = v
			}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				params["member_title"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["password"] = v
			}
			if v, _ := cmd.Flags().GetInt("state"); v >= 0 {
				params["state"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetString("expire"); v != "" {
				params["expire"] = v
			}
			if v, _ := cmd.Flags().GetString("group-out-ids"); v != "" {
				params["group_out_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("leader-out-id"); v != "" {
				params["leader_out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("leader-account"); v != "" {
				params["leader_account"] = v
			}

			resp, err := client.AddSyncMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("out-id", "", "成员在外部系统的唯一ID (必需)")
	cmd.Flags().String("name", "", "成员显示名称 (必需)")
	cmd.Flags().String("account", "", "成员在外部系统的登录帐号 (必需)")
	cmd.Flags().String("email", "", "邮箱")
	cmd.Flags().String("phone", "", "联系电话")
	cmd.Flags().String("title", "", "职位")
	cmd.Flags().String("password", "", "密码，需要由云库校验帐号密码时传入")
	cmd.Flags().Int("state", -1, "状态: 1启用, 0禁用; 添加时传1启用, 修改时传0禁用")
	cmd.Flags().String("expire", "", "临时帐号过期日期, Unix时间戳(秒), 仅保留日期部分, 传0清除过期日期")
	cmd.Flags().String("group-out-ids", "", "成员所属部门的外部唯一ID, 多个用逗号分隔; 空字符串表示顶层部门; 添加时不传则成员不会出现在企业管理后台的成员管理中")
	cmd.Flags().String("leader-out-id", "", "直属上级在外部系统的唯一ID, 与leader-account二选一; 修改时传空字符串可清除上级属性")
	cmd.Flags().String("leader-account", "", "直属上级在外部系统的登录帐号, 与leader-out-id二选一")
	cmd.MarkFlagRequired("out-id")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("account")
	return cmd
}

func newEntSyncDelMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-member",
		Short: "删除同步成员",
		Example: `  ykc ent sync del-member --members "emp001"
  ykc ent sync del-member --members "emp001,emp002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			members, _ := cmd.Flags().GetString("members")

			if err := client.DelSyncMember(members); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("members", "", "成员在外部系统的唯一ID, 多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntSyncAddGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-group",
		Short: "添加或修改同步部门",
		Example: `  ykc ent sync add-group --out-id "dept001" --name "技术部"
  ykc ent sync add-group --out-id "dept002" --name "前端组" --parent-out-id "dept001"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["name"] = v
			}
			if v, _ := cmd.Flags().GetString("parent-out-id"); v != "" {
				params["parent_out_id"] = v
			}
			if v, _ := cmd.Flags().GetInt("orderby"); v >= 0 {
				params["orderby"] = fmt.Sprintf("%d", v)
			} else if v < -1 {
				return apperrors.NewValidation("--orderby 必须是0或正整数")
			}

			if err := client.AddSyncGroup(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("out-id", "", "外部部门ID (必需)")
	cmd.Flags().String("name", "", "部门名称 (必需)")
	cmd.Flags().String("parent-out-id", "", "父部门外部ID")
	cmd.Flags().Int("orderby", -1, "排序值")
	cmd.MarkFlagRequired("out-id")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newEntSyncDelGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-group",
		Short: "删除同步部门",
		Example: `  ykc ent sync del-group --groups "dept001"
  ykc ent sync del-group --groups "dept001,dept002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			groups, _ := cmd.Flags().GetString("groups")

			if err := client.DelSyncGroup(groups); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("groups", "", "部门在外部系统的唯一ID, 多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("groups")
	return cmd
}

func newEntSyncAddGroupMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-group-member",
		Short:   "添加同步部门的成员",
		Example: `  ykc ent sync add-group-member --group-out-id "dept001" --members "emp001|emp002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("group-out-id"); v != "" {
				params["group_out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("members"); v != "" {
				params["members"] = v
			}

			if err := client.AddSyncGroupMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("group-out-id", "", "部门外部ID")
	cmd.Flags().String("members", "", "成员外部ID，多个用竖号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntSyncDelGroupMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-group-member",
		Short: "删除同步部门的成员",
		Example: `  ykc ent sync del-group-member --group-out-id "dept001" --members "emp001"
  ykc ent sync del-group-member --group-out-id "dept001" --members "emp001,emp002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("group-out-id"); v != "" {
				params["group_out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("members"); v != "" {
				params["members"] = v
			}

			if err := client.DelSyncGroupMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("group-out-id", "", "部门在外部系统的唯一ID, 不传表示顶层部门")
	cmd.Flags().String("members", "", "成员在外部系统的唯一ID, 多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntSyncDelMemberGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-member-group",
		Short: "删除同步成员的所属部门",
		Example: `  ykc ent sync del-member-group --members "emp001"
  ykc ent sync del-member-group --members "emp001,emp002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			members, _ := cmd.Flags().GetString("members")

			if err := client.DelSyncMemberGroup(members); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("members", "", "成员在外部系统的唯一ID, 多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("members")
	return cmd
}

func newEntSyncGetMemberByOutIDCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-member",
		Short: "通过外部帐号获取成员信息",
		Example: `  ykc ent sync get-member --out-ids "emp001"
  ykc ent sync get-member --out-ids "emp001,emp002"
  ykc ent sync get-member --user-ids "123,456"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("out-ids"); v != "" {
				params["out_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("user-ids"); v != "" {
				params["user_ids"] = v
			}

			resp, err := client.GetMemberByOutID(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("out-ids", "", "外部成员ID，多个用逗号分隔，与user-ids二选一")
	cmd.Flags().String("user-ids", "", "外部成员登录帐号，多个用逗号分隔，与out-ids二选一")
	return cmd
}

func newEntSyncGetGroupByOutIDCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-group",
		Short:   "通过外部部门ID获取部门信息",
		Example: `  ykc ent sync get-group --out-ids "dept001"
  ykc ent sync get-group --out-ids "dept001,dept002"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("out-ids"); v != "" {
				params["out_ids"] = v
			}

			resp, err := client.GetGroupByOutID(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("out-ids", "", "部门外部ID，多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("out-ids")
	return cmd
}

func newEntSyncAddAdminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-admin",
		Short: "添加管理员",
		Example: `  ykc ent sync add-admin --out-id "emp001"
  ykc ent sync add-admin --out-id "emp001" --email "admin@example.com" --type 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("email"); v != "" {
				if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, v); !matched {
					return apperrors.NewValidation("--email 格式不正确")
				}
				params["member_email"] = v
			}
			if v, _ := cmd.Flags().GetInt("type"); v >= 0 {
				if v != 0 && v != 1 {
					return apperrors.NewValidation("--type 必须是0或1")
				}
				params["type"] = fmt.Sprintf("%d", v)
			}

			if err := client.AddSyncAdmin(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("out-id", "", "外部帐号ID (必需)")
	cmd.Flags().String("email", "", "管理员邮箱")
	cmd.Flags().Int("type", -1, "管理员类型: 0普通管理员, 1超级管理员")
	cmd.MarkFlagRequired("out-id")
	return cmd
}

// === 日志子命令 ===

func newEntLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "管理日志",
		Example: `  ykc ent log
  ykc ent log --type 2 --start 0 --size 100
  ykc ent log --start-dateline 1700000000 --end-dateline 1700100000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			if start < 0 {
				return apperrors.NewValidation("--start 必须是0或正整数")
			}
			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("type"); v != "" {
				t, err := strconv.Atoi(v)
				if err != nil || t < 1 || t > 9 {
					return apperrors.NewValidation("--type 必须是1~9的整数")
				}
				params["type"] = v
			}
			if v, _ := cmd.Flags().GetString("orderby"); v != "" {
				params["orderby"] = v
			}
			if v, _ := cmd.Flags().GetString("start-dateline"); v != "" {
				params["start_dateline"] = v
			}
			if v, _ := cmd.Flags().GetString("end-dateline"); v != "" {
				params["end_dateline"] = v
			}
			params["start"] = fmt.Sprintf("%d", start)
			params["size"] = fmt.Sprintf("%d", size)

			resp, err := client.GetEntLog(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("type", "", "日志类型: 1子管理员, 2成员信息, 3企业设置, 4组织架构, 5角色, 6成员设备, 7存储点, 8文件库, 9开发授权")
	cmd.Flags().String("orderby", "", "排序: asc顺序, desc倒序(默认)")
	cmd.Flags().String("start-dateline", "", "开始时间, Unix时间戳(秒), 获取操作时间 >= 该值的日志")
	cmd.Flags().String("end-dateline", "", "结束时间, Unix时间戳(秒), 获取操作时间 < 该值的日志")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "获取日志条数, 最多1000")
	return cmd
}

// === 角色子命令 ===

func newEntRolesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "roles",
		Short:   "角色列表",
		Example: `  ykc ent roles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			resp, err := client.GetRoles()
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

// getEntClient 获取企业API客户端，复用getClient但确保凭证存在
func getEntClient() *gokuai.Client {
	return getClient()
}
