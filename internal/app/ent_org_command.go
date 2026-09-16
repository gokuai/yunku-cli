package app

import (
	"fmt"
	"strconv"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/spf13/cobra"
)

// newEntOrgCommand creates the enterprise library management command group
func newEntOrgCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "库操作",
		Long: `库操作，包括创建库、修改库、库信息、库列表、库授权、库成员管理、库部门管理等。
需要设置环境变量 GOKUAI_CLIENT_ID 和 GOKUAI_CLIENT_SECRET`,
	}

	cmd.AddCommand(
		newEntOrgCreateCommand(),
		newEntOrgSetCommand(),
		newEntOrgInfoCommand(),
		newEntOrgSearchCommand(),
		newEntOrgInfoByMemberCommand(),
		newEntOrgSetByMemberCommand(),
		newEntOrgListCommand(),
		newEntOrgBindCommand(),
		newEntOrgUnbindCommand(),
		newEntOrgMembersCommand(),
		newEntOrgMemberCommand(),
		newEntOrgSetOwnerCommand(),
		newEntOrgAddMemberCommand(),
		newEntOrgSetMemberRoleCommand(),
		newEntOrgDelMemberCommand(),
		newEntOrgGroupsCommand(),
		newEntOrgAddGroupCommand(),
		newEntOrgDelGroupCommand(),
		newEntOrgSetGroupRoleCommand(),
		newEntOrgDestroyCommand(),
		newEntOrgLogCommand(),
	)

	return cmd
}

// === 创建库 ===

func newEntOrgCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建库",
		Example: `  ykc ent org create --name "新库"
  ykc ent org create --name "新库" --capacity 1073741824 --out-id "my-lib-001"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("参数 --name 不能为空")
			}

			client := getOpenClient()

			params := map[string]string{}
			params["org_name"] = name
			if v, _ := cmd.Flags().GetString("logo"); v != "" {
				params["org_logo"] = v
			}
			if v, _ := cmd.Flags().GetInt64("capacity"); v > 0 {
				params["org_capacity"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetString("storage-point"); v != "" {
				params["storage_point_name"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}

			resp, err := client.CreateOrg(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("name", "", "库名称 (必需)")
	cmd.Flags().String("logo", "", "库图标URL")
	cmd.Flags().Int64("capacity", -1, "库空间上限(字节), -1表示不限制")
	cmd.Flags().String("storage-point", "", "库归属存储点名称")
	cmd.Flags().String("out-id", "", "库外部ID")
	cmd.MarkFlagRequired("name")
	return cmd
}

// === 修改库 ===

func newEntOrgSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "修改库",
		Example: `  ykc ent org set --org-id 1 --name "新名称"
  ykc ent org set --mount-id 2 --capacity 1073741824`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("mount-id"); v != "" {
				params["mount_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["org_name"] = v
			}
			if v, _ := cmd.Flags().GetString("logo"); v != "" {
				params["org_logo"] = v
			}
			if v, _ := cmd.Flags().GetString("capacity"); v != "" {
				params["org_capacity"] = v
			}

			if err := client.SetOrg(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("mount-id", "", "库空间ID")
	cmd.Flags().String("name", "", "库名称")
	cmd.Flags().String("logo", "", "库图标URL")
	cmd.Flags().String("capacity", "", "库空间上限(字节), 空字符串表示不设置上限")
	return cmd
}

// === 库信息 ===

func newEntOrgInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "库信息",
		Example: `  ykc ent org info --org-id 1
  ykc ent org info --mount-id 2
  ykc ent org info --out-id "my-lib-001"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("mount-id"); v != "" {
				params["mount_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}

			resp, err := client.GetOrgInfo(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("mount-id", "", "库空间ID")
	cmd.Flags().String("out-id", "", "库外部ID")
	return cmd
}

// === 搜索库 ===

func newEntOrgSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "搜索库",
		Example: `  ykc ent org search --name "技术部"
  ykc ent org search --prefix "技术" --size 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			size, _ := cmd.Flags().GetInt("size")
			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["name"] = v
			}
			if v, _ := cmd.Flags().GetString("prefix"); v != "" {
				params["prefix"] = v
			}
			params["size"] = fmt.Sprintf("%d", size)

			resp, err := client.SearchOrg(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("name", "", "库名称, 精确匹配")
	cmd.Flags().String("prefix", "", "库名称前缀, 模糊匹配")
	cmd.Flags().Int("size", 10, "返回条数, 不超过1000")
	return cmd
}

// === 个人文件库信息 ===

func newEntOrgInfoByMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info-by-member",
		Short: "个人文件库信息",
		Example: `  ykc ent org info-by-member --member-id 123
  ykc ent org info-by-member --account zhangsan`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

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

			resp, err := client.GetOrgInfoByMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("member-id", "", "成员ID")
	cmd.Flags().String("out-id", "", "成员外部系统唯一ID")
	cmd.Flags().String("account", "", "成员外部系统登录帐号")
	cmd.Flags().String("email", "", "登录邮箱")
	return cmd
}

// === 设置个人文件库 ===

func newEntOrgSetByMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-by-member",
		Short: "设置个人文件库",
		Example: `  ykc ent org set-by-member --member-id 123 --capacity 1073741824`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

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
			if v, _ := cmd.Flags().GetString("capacity"); v != "" {
				params["capacity"] = v
			}

			if err := client.SetOrgByMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("member-id", "", "成员ID")
	cmd.Flags().String("out-id", "", "成员外部系统唯一ID")
	cmd.Flags().String("account", "", "成员外部系统登录帐号")
	cmd.Flags().String("email", "", "登录邮箱")
	cmd.Flags().String("capacity", "", "库空间(字节)(必需)")
	cmd.MarkFlagRequired("capacity")
	return cmd
}

// === 获取库列表 ===

func newEntOrgListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "获取库列表",
		Example: `  ykc ent org list
  ykc ent org list --type 1 --member-id 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("type"); v != "" {
				if v != "0" && v != "1" && v != "2" {
					return apperrors.NewValidation("--type 必须是 0(所有)、1(非个人文件库) 或 2(个人文件库)")
				}
				params["type"] = v
			}
			if v, _ := cmd.Flags().GetString("member-id"); v != "" {
				params["member_id"] = v
			}

			resp, err := client.GetOrgList(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("type", "", "1非个人文件库, 2个人文件库, 默认0所有")
	cmd.Flags().String("member-id", "", "只返回该成员参与的库")
	return cmd
}

// === 获取库授权 ===

func newEntOrgBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind",
		Short: "获取库授权",
		Example: `  ykc ent org bind --org-id 1 --title "我的应用"
  ykc ent org bind --mount-id 2 --title "我的应用"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			title, _ := cmd.Flags().GetString("title")
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("参数 --title 不能为空")
			}

			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("mount-id"); v != "" {
				params["mount_id"] = v
			}
			params["title"] = title

			resp, err := client.BindOrg(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("mount-id", "", "库空间ID")
	cmd.Flags().String("title", "", "对接库文件的应用或系统名称 (必需)")
	cmd.MarkFlagRequired("title")
	return cmd
}

// === 取消库授权 ===

func newEntOrgUnbindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbind",
		Short: "取消库授权",
		Example: `  ykc ent org unbind --org-client-id "abc123"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			orgClientID, _ := cmd.Flags().GetString("org-client-id")

			if err := client.UnbindOrg(orgClientID); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-client-id", "", "库授权client_id (必需)")
	cmd.MarkFlagRequired("org-client-id")
	return cmd
}

// === 获取库成员列表 ===

func newEntOrgMembersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "获取库成员列表",
		Example: `  ykc ent org members --org-id 1
  ykc ent org members --org-id 1 --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			if start < 0 {
				return apperrors.NewValidation("--start 必须是0或正整数")
			}
			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			params["start"] = fmt.Sprintf("%d", start)
			params["size"] = fmt.Sprintf("%d", size)

			resp, err := client.GetOrgMembers(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

// === 查询库成员信息 ===

func newEntOrgMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "查询库成员信息",
		Example: `  ykc ent org member --org-id 1 --ids "123,456" --type member_id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("ids"); v != "" {
				params["ids"] = v
			}
			if v, _ := cmd.Flags().GetString("type"); v != "" {
				params["type"] = v
			}

			resp, err := client.GetOrgMember(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("ids", "", "成员唯一ID或帐号, 多个用逗号分隔 (必需)")
	cmd.Flags().String("type", "", "ids的类型: account, out_id或member_id (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("ids")
	cmd.MarkFlagRequired("type")
	return cmd
}

// === 设置库拥有者 ===

func newEntOrgSetOwnerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-owner",
		Short: "设置库拥有者",
		Example: `  ykc ent org set-owner --org-id 1 --member-id 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-id"); v != "" {
				params["member_id"] = v
			}
			if v, _ := cmd.Flags().GetString("role-id"); v != "" {
				params["role_id"] = v
			}

			if err := client.SetOrgOwner(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-id", "", "设置为库拥有者的成员ID (必需)")
	cmd.Flags().String("role-id", "", "原库拥有者更换角色ID, 不传、传0或传空表示移除原拥有者")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-id")
	return cmd
}

// === 添加库成员 ===

func newEntOrgAddMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member",
		Short: "添加库成员",
		Example: `  ykc ent org add-member --org-id 1 --member-ids "123,456" --role-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["member_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("role-id"); v != "" {
				params["role_id"] = v
			}

			if err := client.AddOrgMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID, 多个用逗号分隔 (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

// === 修改库成员角色 ===

func newEntOrgSetMemberRoleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-member-role",
		Short: "修改库成员角色",
		Example: `  ykc ent org set-member-role --org-id 1 --member-ids "123,456" --role-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["member_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("role-id"); v != "" {
				if _, err := strconv.Atoi(v); err != nil {
					return apperrors.NewValidation("--role-id 必须是整数")
				}
				params["role_id"] = v
			}

			if err := client.SetOrgMemberRole(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID, 多个用逗号分隔 (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

// === 删除库成员 ===

func newEntOrgDelMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-member",
		Short: "删除库成员",
		Example: `  ykc ent org del-member --org-id 1 --member-ids "123,456"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["member_ids"] = v
			}

			if err := client.DelOrgMember(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID, 多个用逗号分隔 (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

// === 获取库部门列表 ===

func newEntOrgGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "获取库部门列表",
		Example: `  ykc ent org groups --org-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			orgID, _ := cmd.Flags().GetString("org-id")

			resp, err := client.GetOrgGroups(orgID)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

// === 库上添加部门 ===

func newEntOrgAddGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-group",
		Short: "库上添加部门",
		Example: `  ykc ent org add-group --org-id 1 --group-id 10 --role-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("role-id"); v != "" {
				params["role_id"] = v
			}

			if err := client.AddOrgGroup(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

// === 删除库上的部门 ===

func newEntOrgDelGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-group",
		Short: "删除库上的部门",
		Example: `  ykc ent org del-group --org-id 1 --group-id 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}

			if err := client.DelOrgGroup(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	return cmd
}

// === 修改库上部门的角色 ===

func newEntOrgSetGroupRoleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-group-role",
		Short: "修改库上部门的角色",
		Example: `  ykc ent org set-group-role --org-id 1 --group-id 10 --role-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("role-id"); v != "" {
				if _, err := strconv.Atoi(v); err != nil {
					return apperrors.NewValidation("--role-id 必须是整数")
				}
				params["role_id"] = v
			}

			if err := client.SetOrgGroupRole(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

// === 删除库 ===

func newEntOrgDestroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "删除库",
		Example: `  ykc ent org destroy --org-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("org-client-id"); v != "" {
				params["org_client_id"] = v
			}

			if err := client.DestroyOrg(params); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), `{"result": "success"}`)
			return nil
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("org-client-id", "", "库授权client_id")
	return cmd
}

// === 库日志 ===

func newEntOrgLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "库日志",
		Example: `  ykc ent org log --org-id 1
  ykc ent org log --org-id 1 --act "1,2" --size 100
  ykc ent org log --mount-id 2 --start-dateline 1700000000 --end-dateline 1700100000
  ykc ent org log --org-id 1 --act "20,21" --orderby asc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getOpenClient()

			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			if start < 0 {
				return apperrors.NewValidation("--start 必须是0或正整数")
			}
			if size <= 0 {
				return apperrors.NewValidation("--size 必须是大于0的整数")
			}

			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("org-id"); v != "" {
				params["org_id"] = v
			}
			if v, _ := cmd.Flags().GetString("mount-id"); v != "" {
				params["mount_id"] = v
			}
			if v, _ := cmd.Flags().GetString("act"); v != "" {
				validActs := map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "12": true, "13": true, "20": true, "21": true, "1014": true, "1015": true, "1016": true, "1017": true, "1018": true}
				for _, a := range strings.Split(v, ",") {
					if !validActs[a] {
						return apperrors.NewValidation(fmt.Sprintf("--act 包含无效值 %q, 允许值: 0,1,2,3,4,5,6,12,13,20,21,1014,1015,1016,1017,1018", a))
					}
				}
				params["act"] = v
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

			resp, err := client.GetOrgLog(params)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("mount-id", "", "库空间ID")
	cmd.Flags().String("act", "", "过滤操作类型, 多个用逗号分隔: 0删除, 1创建/上传, 2重命名, 3编辑, 4移动, 5还原已删除项, 6版本还原, 12锁定, 13解锁, 20下载, 21预览, 1014生成外链, 1015访问外链, 1016外链下载, 1017外链存库, 1018外链上传")
	cmd.Flags().String("orderby", "", "排序: asc顺序, desc倒序(默认)")
	cmd.Flags().String("start-dateline", "", "开始时间, Unix时间戳(秒)")
	cmd.Flags().String("end-dateline", "", "结束时间, Unix时间戳(秒)")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "获取日志条数, 最多1000")
	return cmd
}

