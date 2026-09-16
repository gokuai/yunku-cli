package app

import (
	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

// newFavoriteCommand creates the favorite management command group
func newFavoriteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "favorite",
		Short: "收藏管理",
		Long: `收藏管理，包括添加收藏、取消收藏、获取收藏列表等。
需要先登录获取 access_token: ykc auth login`,
	}

	cmd.AddCommand(
		newFavoriteAddCommand(),
		newFavoriteRemoveCommand(),
		newFavoriteListCommand(),
	)

	return cmd
}

func newFavoriteAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "添加文件到收藏夹",
		Example: `  ykc favorite add --mount-id 1 --fullpath /foo/bar.txt
  ykc favorite add --mount-id 1 --fullpath /foo/bar.txt --fav-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			favID, _ := cmd.Flags().GetInt("fav-id")

			if mountID == 0 {
				return apperrors.NewValidation("mount-id 无效")
			}
			if fullpath == "" {
				return apperrors.NewValidation("fullpath 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			req := gokuai.FavoritesRequest{
				MountID:  mountID,
				Fullpath: fullpath,
				FavID:    favID,
			}

			if err := userClient.AddToFavorites(req, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().Int("mount-id", 0, "文件库ID (必需)")
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().Int("fav-id", -1, "收藏夹ID，-1表示默认收藏夹")
	cmd.MarkFlagRequired("mount-id")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFavoriteRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "从收藏夹移除文件",
		Example: `  ykc favorite remove --mount-id 1 --fullpath /foo/bar.txt
  ykc favorite remove --mount-id 1 --fullpath /foo/bar.txt --fav-id 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			favID, _ := cmd.Flags().GetInt("fav-id")

			if mountID == 0 {
				return apperrors.NewValidation("mount-id 无效")
			}
			if fullpath == "" {
				return apperrors.NewValidation("fullpath 不能为空")
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			req := gokuai.FavoritesRequest{
				MountID:  mountID,
				Fullpath: fullpath,
				FavID:    favID,
			}

			if err := userClient.RemoveFromFavorites(req, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().Int("mount-id", 0, "文件库ID (必需)")
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().Int("fav-id", -1, "收藏夹ID，-1表示默认收藏夹")
	cmd.MarkFlagRequired("mount-id")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFavoriteListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "获取收藏夹文件列表",
		Example: `  ykc favorite list --fav-id 1
  ykc favorite list --fav-id 1 --start 0 --size 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			favID, _ := cmd.Flags().GetInt("fav-id")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFavorites(favID, start, size, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("fav-id", -1, "收藏夹ID，-1表示默认收藏夹")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	return cmd
}
