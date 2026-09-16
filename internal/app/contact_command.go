package app

import (
	"fmt"
	"regexp"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/spf13/cobra"
)

// newContactCommand creates the contact management command group
func newContactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contact",
		Short: "通讯录管理",
		Long: `通讯录管理，包括部门管理、成员管理等。
需要先登录获取 access_token: ykc auth login`,
	}

	cmd.AddCommand(
		newContactGroupCommand(),
		newContactMemberCommand(),
		newContactRootGroupCommand(),
	)

	return cmd
}

// === 部门管理子命令 ===

func newContactGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "部门管理",
	}

	cmd.AddCommand(
		newContactGroupListCommand(),
		newContactGroupSearchCommand(),
		newContactGroupMemberListCommand(),
		newContactGroupAddCommand(),
		newContactGroupUpdateCommand(),
		newContactGroupDelCommand(),
		newContactGroupAddMemberCommand(),
		newContactGroupRemoveMemberCommand(),
	)

	return cmd
}

func newContactGroupListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "获取部门列表",
		Example: `  ykc contact group list --ent-id 1 --group-id 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			groupID, _ := cmd.Flags().GetString("group-id")
			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("filter"); v != "" {
				params["filter"] = v
			}
			resp, err := userClient.GetContactGroupList(entID, groupID, params, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "0", "部门ID，0返回一级部门")
	cmd.Flags().String("filter", "", "0表示忽略部门禁用和隐藏规则")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}

func newContactGroupSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Short:   "查询部门列表",
		Example: `  ykc contact group search --ent-id 1 --keyword "技术"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			keyword, _ := cmd.Flags().GetString("keyword")
			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}
			if keyword == "" {
				return apperrors.NewValidation("keyword 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			resp, err := userClient.SearchContactGroup(entID, keyword, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("keyword", "", "搜索关键字 (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("keyword")
	return cmd
}

func newContactGroupMemberListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "members",
		Short:   "获取部门成员列表",
		Example: `  ykc contact group members --ent-id 1 --group-id 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")
			if start < 0 {
				return apperrors.NewValidation("--start 必须是 0 或正整数")
			}
			if size < 0 {
				return apperrors.NewValidation("--size 必须是大于 0 的整数")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{"ent_id": entID}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetInt("start"); v > 0 {
				params["start"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetInt("size"); v > 0 {
				params["size"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetString("keyword"); v != "" {
				params["keyword"] = v
			}
			if v, _ := cmd.Flags().GetBool("show-child"); v {
				params["show_child"] = "1"
			}
			if v, _ := cmd.Flags().GetString("filter"); v != "" {
				params["filter"] = v
			}
			resp, err := userClient.GetContactGroupMemberList(params, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 0, "返回条数")
	cmd.Flags().String("keyword", "", "搜索关键字")
	cmd.Flags().Bool("show-child", false, "显示子部门成员")
	cmd.Flags().String("filter", "", "0表示忽略部门禁用和隐藏规则")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}

func newContactGroupAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "添加部门",
		Example: `  ykc contact group add --ent-id 1 --parent-id 0 --name "技术部"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("parent-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["name"] = v
			}
			if v, _ := cmd.Flags().GetString("state"); v != "" {
				if v != "0" && v != "1" {
					return apperrors.NewValidation("--state 的值必须是 0 或 1")
				}
				params["state"] = v
			}
			resp, err := userClient.AddContactGroup(params, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("parent-id", "0", "父部门ID，0为根部门")
	cmd.Flags().String("name", "", "部门名称 (必需)")
	cmd.Flags().String("state", "", "部门状态: 1启用, 0禁用")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newContactGroupUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "更新部门",
		Example: `  ykc contact group update --ent-id 1 --group-id 2 --name "研发部"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["name"] = v
			}
			if v, _ := cmd.Flags().GetString("state"); v != "" {
				if v != "0" && v != "1" {
					return apperrors.NewValidation("--state 的值必须是 0 或 1")
				}
				params["state"] = v
			}
			if err := userClient.UpdateContactGroup(params, secret); err != nil {
				return err
			}
			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.Flags().String("name", "", "部门名称")
	cmd.Flags().String("state", "", "部门状态: 1启用, 0禁用")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("group-id")
	return cmd
}

func newContactGroupDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "删除部门",
		Example: `  ykc contact group delete --ent-id 1 --group-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if err := userClient.DelContactGroup(params, secret); err != nil {
				return err
			}
			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("group-id")
	return cmd
}

func newContactGroupAddMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-member",
		Short:   "添加部门成员",
		Example: `  ykc contact group add-member --ent-id 1 --group-id 2 --member-ids "1,2,3"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["_member_ids"] = v
			}
			resp, err := userClient.AddContactGroupMember(params, secret)
			if err != nil {
				return err
			}
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID")
	cmd.Flags().String("member-ids", "", "成员ID，逗号分隔 (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

func newContactGroupRemoveMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove-member",
		Short:   "移除部门成员",
		Example: `  ykc contact group remove-member --ent-id 1 --group-id 2 --member-ids "1,2,3"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["_member_ids"] = v
			}
			if err := userClient.RemoveContactGroupMember(params, secret); err != nil {
				return err
			}
			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID")
	cmd.Flags().String("member-ids", "", "成员ID，逗号分隔 (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

// === 成员管理子命令 ===

func newContactMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "成员管理",
	}
	cmd.AddCommand(
		newContactMemberInfoCommand(),
		newContactMemberGroupsCommand(),
		newContactMemberAddCommand(),
		newContactMemberUpdateCommand(),
		newContactMemberRemoveCommand(),
	)
	return cmd
}

func newContactMemberInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "获取成员信息",
		Example: `  ykc contact member info --ent-id 1 --member-id 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			memberID, _ := cmd.Flags().GetString("member-id")
			if entID == "" || memberID == "" {
				return apperrors.NewValidation("ent-id 和 member-id 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetBool("with-groups"); v {
				params["with_groups"] = "1"
			}
			if v, _ := cmd.Flags().GetBool("show-space"); v {
				params["show_space"] = "1"
			}
			resp, err := userClient.GetContactMemberInfo(entID, memberID, params, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("member-id", "", "成员ID (必需)")
	cmd.Flags().Bool("with-groups", false, "显示所属部门")
	cmd.Flags().Bool("show-space", false, "显示个人空间使用量")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-id")
	return cmd
}

func newContactMemberGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "groups",
		Short:   "获取成员所属部门列表",
		Example: `  ykc contact member groups --ent-id 1 --member-id 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			memberID, _ := cmd.Flags().GetString("member-id")
			if entID == "" || memberID == "" {
				return apperrors.NewValidation("ent-id 和 member-id 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			resp, err := userClient.GetContactMemberGroups(entID, memberID, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("member-id", "", "成员ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-id")
	return cmd
}

func newContactMemberAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "添加成员",
		Example: `  ykc contact member add --ent-id 1 --name "张三"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			password, _ := cmd.Flags().GetString("password")
			if password == "" {
				return apperrors.NewValidation("--password 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetString("group-id"); v != "" {
				params["group_id"] = v
			}
			if v, _ := cmd.Flags().GetString("group-ids"); v != "" {
				params["group_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("email"); v != "" {
				params["member_email"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["member_account"] = v
			}
			if v, _ := cmd.Flags().GetString("phone"); v != "" {
				if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(v) {
					return apperrors.NewValidation("--phone 必须是有效的中国大陆手机号")
				}
				params["member_phone"] = v
			}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				params["member_title"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["member_password"] = v
			}
			if v, _ := cmd.Flags().GetString("expire"); v != "" {
				params["expire"] = v
			}
			if v, _ := cmd.Flags().GetString("leader-id"); v != "" {
				params["leader_id"] = v
			}
			resp, err := userClient.AddContactMember(params, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("name", "", "成员名称 (必需)")
	cmd.Flags().String("group-id", "", "部门ID，默认根部门，-1不加入部门")
	cmd.Flags().String("group-ids", "", "多个部门ID，逗号分隔")
	cmd.Flags().String("email", "", "邮箱地址")
	cmd.Flags().String("account", "", "帐号")
	cmd.Flags().String("phone", "", "手机号")
	cmd.Flags().String("title", "", "职位")
	cmd.Flags().String("password", "", "登录密码 (必需)")
	cmd.Flags().String("expire", "", "临时成员过期日期，格式: 1970-01-01")
	cmd.Flags().String("leader-id", "", "直属上级用户ID")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("password")
	return cmd
}

func newContactMemberUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "修改成员",
		Example: `  ykc contact member update --ent-id 1 --member-ids "123" --name "张三丰"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["_member_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetString("group-ids"); v != "" {
				params["group_ids"] = v
			}
			if v, _ := cmd.Flags().GetString("email"); v != "" {
				params["member_email"] = v
			}
			if v, _ := cmd.Flags().GetString("phone"); v != "" {
				if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(v) {
					return apperrors.NewValidation("--phone 必须是有效的中国大陆手机号")
				}
				params["member_phone"] = v
			}
			if v, _ := cmd.Flags().GetString("title"); v != "" {
				params["member_title"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["member_password"] = v
			}
			if v, _ := cmd.Flags().GetString("state"); v != "" {
				if v != "0" && v != "1" {
					return apperrors.NewValidation("--state 的值必须是 0 或 1")
				}
				params["state"] = v
			}
			if v, _ := cmd.Flags().GetString("expire"); v != "" {
				params["expire"] = v
			}
			if v, _ := cmd.Flags().GetString("leader-id"); v != "" {
				params["leader_id"] = v
			}
			if err := userClient.UpdateContactMember(params, secret); err != nil {
				return err
			}
			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID，逗号分隔 (必需)")
	cmd.Flags().String("name", "", "成员名称")
	cmd.Flags().String("group-ids", "", "所有部门ID，逗号分隔")
	cmd.Flags().String("email", "", "邮箱地址")
	cmd.Flags().String("phone", "", "手机号")
	cmd.Flags().String("title", "", "职位")
	cmd.Flags().String("password", "", "登录密码")
	cmd.Flags().String("state", "", "成员状态: 1启用, 0禁用")
	cmd.Flags().String("expire", "", "临时成员过期日期")
	cmd.Flags().String("leader-id", "", "直属上级用户ID")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

func newContactMemberRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "移除成员",
		Example: `  ykc contact member remove --ent-id 1 --member-ids "123,456" --to-member-id 789`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("member-ids"); v != "" {
				params["_member_ids"] = v
			} else {
				return apperrors.NewValidation("member-ids 不能为空")
			}
			if v, _ := cmd.Flags().GetString("to-member-id"); v != "" {
				params["_to_member_id"] = v
			} else {
				return apperrors.NewValidation("to-member-id 不能为空")
			}
			if v, _ := cmd.Flags().GetString("ent-id"); v != "" {
				params["ent_id"] = v
			}
			if err := userClient.RemoveContactMember(params, secret); err != nil {
				return err
			}
			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID，逗号分隔 (必需)")
	cmd.Flags().String("to-member-id", "", "转移库给此成员ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("member-ids")
	cmd.MarkFlagRequired("to-member-id")
	return cmd
}

// === 其他子命令 ===

func newContactRootGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "root-group",
		Short:   "获取根部门属性",
		Example: `  ykc contact root-group --ent-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			if entID == "" {
				return apperrors.NewValidation("ent-id 不能为空")
			}
			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}
			resp, err := userClient.GetContactRootGroup(entID, secret)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}
