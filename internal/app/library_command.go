package app

import (
	"fmt"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/spf13/cobra"
)

// newLibraryCommand creates the library management command group
func newLibraryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "library",
		Short: "库管理",
		Long: `库管理，包括库信息、部门管理、成员管理等。
需要先登录获取 access_token: ykc auth login`,
	}

	cmd.AddCommand(
		newLibraryInfoCommand(),
		newLibraryMembersCommand(),
		newLibraryGroupsCommand(),
		newLibraryGroupAddCommand(),
		newLibraryGroupUpdateCommand(),
		newLibraryMemberAddCommand(),
		newLibraryMemberUpdateCommand(),
		newLibraryMemberRemoveCommand(),

		newLibraryDeleteCommand(),
		newLibraryGroupRemoveCommand(),


		newLibraryUpdateCommand(),

		newLibraryGroupSearchCommand(),
		newLibraryListCommand(),
		newLibraryCreateCommand(),


	)

	return cmd
}

func newLibraryInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "获取库信息",
		Example: `  ykc library info --org-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetInt("org-id")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetLibraryInfo(orgID, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("org-id", 0, "库ID (必需)")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

func newLibraryMembersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "获取库成员列表",
		Example: `  ykc library members --org-id 919427
  ykc library members --org-id 919427 --start 0 --size 20
  ykc library members --org-id 919427 --state 1
  ykc library members --org-id 919427 --keyword test
  ykc library members --org-id 919427 --with-info --with-group
  ykc library members --org-id 919427 --group-fullpath
  ykc library members --org-id 919427 --role-ids 17042
  ykc library members --org-id 919427 --is-out`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetInt("org-id")

			params := map[string]string{}

			start, _ := cmd.Flags().GetInt("start")
			if start > 0 {
				params["start"] = fmt.Sprintf("%d", start)
			}
			size, _ := cmd.Flags().GetInt("size")
			if size > 0 {
				params["size"] = fmt.Sprintf("%d", size)
			}
			state, _ := cmd.Flags().GetInt("state")
			if state > 0 {
				params["state"] = fmt.Sprintf("%d", state)
			}
			keyword, _ := cmd.Flags().GetString("keyword")
			if keyword != "" {
				params["keyword"] = keyword
			}
			withInfo, _ := cmd.Flags().GetBool("with-info")
			if withInfo {
				params["with_info"] = "1"
			}
			withGroup, _ := cmd.Flags().GetBool("with-group")
			if withGroup {
				params["with_group"] = "1"
			}
			groupFullpath, _ := cmd.Flags().GetBool("group-fullpath")
			if groupFullpath {
				params["group_fullpath"] = "1"
			}
			roleIDs, _ := cmd.Flags().GetString("role-ids")
			if roleIDs != "" {
				params["role_ids"] = roleIDs
			}
			isOut, _ := cmd.Flags().GetBool("is-out")
			if isOut {
				params["is_out"] = "1"
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetLibraryMembers(orgID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("org-id", 0, "库ID (必需)")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 0, "数量，不传拿全部")
	cmd.Flags().Int("state", 0, "成员状态: 1启用, 0禁用")
	cmd.Flags().String("keyword", "", "搜索关键字")
	cmd.Flags().Bool("with-info", false, "返回成员详细信息")
	cmd.Flags().Bool("with-group", false, "返回成员分组信息")
	cmd.Flags().Bool("group-fullpath", false, "返回成员部门路径")
	cmd.Flags().String("role-ids", "", "角色ID")
	cmd.Flags().Bool("is-out", false, "是否外部成员")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

func newLibraryGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "获取库部门列表",
		Example: `  ykc library groups --org-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			withInfo, _ := cmd.Flags().GetBool("with-info")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetLibraryGroups(orgID, withInfo, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().Bool("with-info", false, "返回详细信息")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

func newLibraryGroupAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-group",
		Short: "添加库部门",
		Example: `  ykc library add-group --ent-id 1 --org-id 2 --group-id 3 --role-id 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			orgID, _ := cmd.Flags().GetString("org-id")
			groupID, _ := cmd.Flags().GetString("group-id")
			roleID, _ := cmd.Flags().GetString("role-id")

			params := map[string]string{
				"ent_id":   entID,
				"org_id":   orgID,
				"group_id": groupID,
				"role_id":  roleID,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.AddLibraryGroup(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

func newLibraryGroupUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-group",
		Short: "更新库部门",
		Example: `  ykc library update-group --ent-id 1 --org-id 2 --group-id 3 --role-id 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			orgID, _ := cmd.Flags().GetString("org-id")
			groupID, _ := cmd.Flags().GetString("group-id")
			roleID, _ := cmd.Flags().GetString("role-id")

			params := map[string]string{
				"ent_id":   entID,
				"org_id":   orgID,
				"group_id": groupID,
				"role_id":  roleID,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.UpdateLibraryGroup(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

func newLibraryMemberAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-member",
		Short: "添加库成员",
		Example: `  ykc library add-member --org-id 1 --member-ids "123,456" --role-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			memberIDs, _ := cmd.Flags().GetString("member-ids")
			roleID, _ := cmd.Flags().GetString("role-id")

			params := map[string]string{
				"org_id":      orgID,
				"_member_ids": memberIDs,
				"role_id":     roleID,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.AddLibraryMember(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID,以逗号分隔 (必需)")
	cmd.Flags().String("role-id", "", "角色ID (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	cmd.MarkFlagRequired("role-id")
	return cmd
}

func newLibraryMemberUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-member",
		Short: "更新库成员",
		Example: `  ykc library update-member --org-id 1 --member-ids "123,456" --role-id 2
  ykc library update-member --org-id 1 --member-ids "123" --state 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			memberIDs, _ := cmd.Flags().GetString("member-ids")
			roleID, _ := cmd.Flags().GetString("role-id")
			state, _ := cmd.Flags().GetString("state")

			params := map[string]string{
				"org_id":      orgID,
				"_member_ids": memberIDs,
			}
			if roleID != "" {
				params["role_id"] = roleID
			}
			if state != "" {
				if state != "0" && state != "1" {
					return apperrors.NewValidation("state 的值必须是 0 或 1")
				}
				params["state"] = state
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.UpdateLibraryMember(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID,以逗号分隔 (必需)")
	cmd.Flags().String("role-id", "", "角色ID")
	cmd.Flags().String("state", "", "成员状态: 0禁用 1正常")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

func newLibraryMemberRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-member",
		Short: "移除库成员",
		Example: `  ykc library remove-member --org-id 1 --member-ids "123,456"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			memberIDs, _ := cmd.Flags().GetString("member-ids")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.RemoveLibraryMember(orgID, memberIDs, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("member-ids", "", "成员ID列表,逗号分隔 (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("member-ids")
	return cmd
}

func newLibraryDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "删除库",
		Example: `  ykc library delete --org-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DeleteLibrary(orgID, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.MarkFlagRequired("org-id")
	return cmd
}

func newLibraryGroupRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-group",
		Short: "移除库部门",
		Example: `  ykc library remove-group --ent-id 1 --org-id 2 --group-id 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			entID, _ := cmd.Flags().GetString("ent-id")
			orgID, _ := cmd.Flags().GetString("org-id")
			groupID, _ := cmd.Flags().GetString("group-id")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.RemoveLibraryGroup(entID, orgID, groupID, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("ent-id", "", "企业ID (必需)")
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("group-id", "", "部门ID (必需)")
	cmd.MarkFlagRequired("ent-id")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("group-id")
	return cmd
}

func newLibraryUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新库信息",
		Example: `  ykc library update --org-id 1 --name "新名称"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			name, _ := cmd.Flags().GetString("name")

			params := map[string]string{}
			if orgID != "" {
				params["org_id"] = orgID
			}
			if name != "" {
				params["name"] = name
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.UpdateLibrary(params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("org-id", "", "库ID")
	cmd.Flags().String("name", "", "库名称")
	return cmd
}



func newLibraryGroupSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search-group",
		Short: "搜索库部门",
		Example: `  ykc library search-group --org-id 1 --keyword "销售"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			orgID, _ := cmd.Flags().GetString("org-id")
			keyword, _ := cmd.Flags().GetString("keyword")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.SearchLibraryGroup(orgID, keyword, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("org-id", "", "库ID (必需)")
	cmd.Flags().String("keyword", "", "搜索关键字 (必需)")
	cmd.MarkFlagRequired("org-id")
	cmd.MarkFlagRequired("keyword")
	return cmd
}

func newLibraryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "库列表",
		Example: `  ykc library list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			userClient, err := getUserClient()
			if err != nil {
				return err
			}

			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetMountList(secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newLibraryCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "创建库",
		Example: `  ykc library create --name "新库" --ent-id 0
  ykc library create --name "新库" --ent-id 0 --capacity 1073741824`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			entID, _ := cmd.Flags().GetInt("ent-id")
			capacity, _ := cmd.Flags().GetInt64("capacity")
			storagePoint, _ := cmd.Flags().GetString("storage-point")

			params := map[string]string{
				"name":   name,
				"ent_id": fmt.Sprintf("%d", entID),
			}
			if capacity > 0 {
				params["capacity"] = fmt.Sprintf("%d", capacity)
			}
			if storagePoint != "" {
				params["storage_point"] = storagePoint
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.CreateLibrary(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("name", "", "库名称 (必需)")
	cmd.Flags().Int("ent-id", 0, "企业ID (必需，0表示个人)")
	cmd.Flags().Int64("capacity", -1, "空间上限(字节)，-1表示不限制")
	cmd.Flags().String("storage-point", "", "存储点名称")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("ent-id")
	return cmd
}


