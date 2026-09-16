package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

// === File Commands ===

func newFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "够快云库文件操作",
		Long: `够快云库文件操作，支持列表、搜索、上传、下载等。
需要先登录获取 access_token: ykc auth login`,
	}

	cmd.AddCommand(
		newFileListCommand(),
		newFileInfoCommand(),
		newFilePreviewURLCommand(),
		newFileCreateFolderCommand(),
		newFileCreateFileCommand(),
		newFileCopyCommand(),
		newFileDeleteCommand(),
		newFileRecycleCommand(),
		newFileRecoverCommand(),
		newFileHistoryCommand(),
		newFileLinkCommand(),
		newFileLockCommand(),
		newFileKeywordCommand(),
		newFilePermissionCommand(),
		newFileUpdatesCommand(),
		newFileClearCommand(),
		newFileDeleteCompletelyCommand(),
		newFileQueueCommand(),
		// v2 APIs (merged)
		newFileMoveCommand(),
		newFileRenameCommand(),
		newFileV2AttributeCommand(),
		newFileV2OpenURLCommand(),

		//newFileV2SaveCommand(), //暂时用不到
		newFileV2SearchV2Command(),
		newFileV2AddCommentCommand(),
		newFileV2GetCommentCommand(),
		newFileV2CEditURLCommand(),
		newFileV2AddLifecycleCommand(),
		newFileV2DelLifecycleCommand(),
		newFileV2GetLifecycleCommand(),
		newFileV2SetRemindCommand(),
		newFileV2DelRemindCommand(),
		newFileV2ExistCommand(),
		newFileV2DownloadCommand(),
		newFileV2RevertCommand(),
		newFileV2ImageSearchCommand(),
		newFileV2GetRemindCommand(),
		newFileV2GetRemindsCommand(),
	)

	// 添加全局 mount-id 参数
	cmd.PersistentFlags().Int("mount-id", 0, "库空间ID，-1: 代表个人库 (必需)")
	cmd.MarkFlagRequired("mount-id")

	return cmd
}

func newFileListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "文件列表",
		Example: `  ykc file ls --mount-id 1 --fullpath documents
  ykc file ls --mount-id 1 --fullpath "" --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			order, err := getStringFlagInSet(cmd, "order",
				"filename asc", "filename desc",
				"last_dateline asc", "last_dateline desc",
				"filesize asc", "filesize desc")
			if err != nil {
				return err
			}
			dir, _ := cmd.Flags().GetInt("dir")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			params := map[string]string{
				"fullpath": fullpath,
				"start":    fmt.Sprintf("%d", start),
				"size":     fmt.Sprintf("%d", size),
			}
			if order != "" {
				params["order"] = order
			}
			if dir > 0 {
				params["dir"] = fmt.Sprintf("%d", dir)
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileList(mountID, params, secret)
			if err != nil {
				return err
			}

			// Filter out items with mismatched mount_id (API may return items with mount_id=0)
			filtered := make([]gokuai.FileInfoV2, 0, len(resp.List))
			for _, item := range resp.List {
				if int(item.MountID) == mountID {
					filtered = append(filtered, item)
				}
			}
			resp.List = filtered
			resp.Count = gokuai.FlexInt(len(filtered))

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径，空字符串表示根目录")
	cmd.Flags().String("order", "", "排序方式: filename asc|desc, last_dateline asc|desc, filesize asc|desc")
	cmd.Flags().Int("dir", 0, "1只返回文件夹, 0都返回")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	return cmd
}

func newFileInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "获取文件（夹）信息",
		Example: `  ykc file info --mount-id 1 --fullpath documents/readme.txt
  ykc file info --mount-id 1 --hash abc123
  ykc file info --mount-id 1 --fullpath documents/readme.txt --open
  ykc file info --mount-id 1 --fullpath doc.txt --ignores tag,favorite,property`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			hash, _ := cmd.Flags().GetString("hash")
			open, _ := cmd.Flags().GetBool("open")
			hid, _ := cmd.Flags().GetString("hid")
			nop, _ := cmd.Flags().GetBool("nop")
			stat, _ := cmd.Flags().GetBool("stat")
			ignores, _ := cmd.Flags().GetString("ignores")
			wfID, _ := cmd.Flags().GetInt("wf-id")
			wfApproveID, _ := cmd.Flags().GetInt("wf-approve-id")

			params := map[string]string{}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if hash != "" {
				params["hash"] = hash
			}
			if open {
				params["open"] = "1"
			}
			if hid != "" {
				params["hid"] = hid
			}
			if nop {
				params["nop"] = "1"
			}
			if stat {
				params["stat"] = "1"
			}
			if ignores != "" {
				params["ignores"] = ignores
			}
			if wfID > 0 {
				params["wf_id"] = fmt.Sprintf("%d", wfID)
			}
			if wfApproveID > 0 {
				params["wf_approve_id"] = fmt.Sprintf("%d", wfApproveID)
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileInfo(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件的路径")
	cmd.Flags().String("hash", "", "路径hash")
	cmd.Flags().Bool("open", false, "是否打开该文件(默认false)")
	cmd.Flags().String("hid", "", "版本ID")
	cmd.Flags().Bool("nop", false, "不包含库权限, 只返回文件夹上设置的权限 (不添加此参数则包含库权限)")
	cmd.Flags().Bool("stat", false, "记录下载日志 (不添加此参数则不记录)")
	cmd.Flags().String("ignores", "", "忽略哪些返回信息(,隔开) (tag,favorite,property,permission,lock,url,preview,thumbnail)")
	cmd.Flags().Int("wf-id", 0, "流程ID(流程审批提交前)")
	cmd.Flags().Int("wf-approve-id", 0, "流程审批ID(流程审批提交后)")
	return cmd
}

func newFilePreviewURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview-url",
		Short: "获取文件预览地址",
		Long: `获取文件预览地址。
注意: 该接口会记录预览日志`,
		Example: `  ykc file preview-url --mount-id 1 --hash abc123
  ykc file preview-url --mount-id 1 --hash abc123 --watermark
  ykc file preview-url --mount-id 1 --hash abc123 --watermark --member-name "张三"
  ykc file preview-url --mount-id 1 --hash abc123 --filename report.pdf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			hash, _ := cmd.Flags().GetString("hash")
			filename, _ := cmd.Flags().GetString("filename")
			watermark, _ := cmd.Flags().GetBool("watermark")
			memberName, _ := cmd.Flags().GetString("member-name")
			dialogID, _ := cmd.Flags().GetString("dialog-id")
			messageID, _ := cmd.Flags().GetString("message-id")

			params := map[string]string{}
			if hash != "" {
				params["hash"] = hash
			}
			if filename != "" {
				params["filename"] = filename
			}
			if watermark {
				params["watermark"] = "1"
			}
			if memberName != "" {
				params["member_name"] = memberName
			}
			if dialogID != "" {
				params["dialog_id"] = dialogID
			}
			if messageID != "" {
				params["message_id"] = messageID
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetPreviewURL(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("hash", "", "文件唯一标识 (必需)")
	cmd.Flags().String("filename", "", "强制指定文件名")
	cmd.Flags().Bool("watermark", false, "是否显示水印(默认不显示)")
	cmd.Flags().String("member-name", "", "显示水印时,指定水印上的预览者身份")
	cmd.Flags().String("dialog-id", "", "会话ID")
	cmd.Flags().String("message-id", "", "消息ID")
	cmd.MarkFlagRequired("hash")
	return cmd
}

func newFileCreateFolderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mkdir",
		Short:   "创建文件夹",
		Example: `  ykc file mkdir --mount-id 1 --fullpath documents/新建文件夹`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.CreateFolder(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileCreateFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-file",
		Short: "上传文件（支持分块上传）",
		Long: `上传本地文件到够快云库（用户 API）。

当指定 --file 时，自动计算 SHA1 哈希和文件大小，完整执行分块上传流程：
  create_file → upload_init → upload_part(s) → upload_finish

create_file 调用签名时自动排除 filehash / filesize（协议要求）。
upload_init 通过 x-gk-token 请求头传递 access_token（无需 org_client_id）。`,
		Example: `  # 上传本地文件（推荐，自动分块）
  ykc file create-file --mount-id 1 --file ./report.pdf --fullpath 文档/report.pdf

  # 指定分块大小（默认 4MB）
  ykc file create-file --mount-id 1 --file ./large.zip --fullpath 归档/large.zip --chunk-size 8388608

  # 覆盖同名文件
  ykc file create-file --mount-id 1 --file ./data.csv --fullpath data/data.csv --overwrite`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			localFile, _ := cmd.Flags().GetString("file")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			chunkSize, _ := cmd.Flags().GetInt64("chunk-size")

			if localFile == "" {
				return apperrors.NewValidation("请提供 --file（本地文件路径）")
			}

			absPath, err := filepath.Abs(expandHomePath(localFile))
			if err != nil {
				return fmt.Errorf("解析文件路径失败: %w", err)
			}
			if _, err := os.Stat(absPath); err != nil {
				return apperrors.NewValidation(fmt.Sprintf("本地文件不存在: %s", absPath))
			}
			if fullpath == "" {
				fullpath = "/" + filepath.Base(absPath)
			}
			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			extraParams := map[string]string{}
			if overwrite {
				extraParams["overwrite"] = "1"
			} else {
				extraParams["overwrite"] = "0"
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "正在计算文件哈希: %s\n", absPath)

			resp, err := userClient.UploadFile(mountID, absPath, fullpath, chunkSize, secret, extraParams)
			if err != nil {
				return fmt.Errorf("上传失败: %w", err)
			}

			stateDesc := "已上传"
			if int(resp.State) == 1 {
				stateDesc = "秒传（哈希命中）"
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "上传完成 (%s): %s\n", stateDesc, resp.Fullpath)

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("file", "", "本地文件路径（必需）")
	cmd.Flags().String("fullpath", "", "库中的目标路径（默认取本地文件名）")
	cmd.Flags().Bool("overwrite", false, "覆盖已存在的同名文件")
	cmd.Flags().Int64("chunk-size", gokuai.DefaultChunkSize, "分块大小（字节），默认 4MB")
	cmd.MarkFlagRequired("file")
	return cmd
}

func newFileCopyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "复制文件(夹)",
		Example: `  ykc file copy --mount-id 1 --fullpath "documents/a.txt" --target-mount-id 1 --target-fullpath "documents2/"
  ykc file copy --mount-id 1 --fullpaths "backup/a.txt|backup/b.txt" --target-mount-id 2 --target-fullpath "backup2/"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			fullpaths, _ := cmd.Flags().GetString("fullpaths")
			fullpaths = normalizeMSYSPaths(fullpaths)
			if fullpaths != "" {
				parts := strings.Split(fullpaths, "|")
				for i, p := range parts {
					if p != "" && !strings.HasPrefix(p, "/") {
						parts[i] = "/" + p
					}
				}
				fullpaths = strings.Join(parts, "|")
			}
			targetMountID, _ := cmd.Flags().GetString("target-mount-id")
			targetFullpath, _ := cmd.Flags().GetString("target-fullpath")
			targetFullpath = normalizeMSYSPath(targetFullpath)
			if targetFullpath != "" && !strings.HasPrefix(targetFullpath, "/") {
				targetFullpath = "/" + targetFullpath
			}
			overwrite, _ := cmd.Flags().GetString("overwrite")

			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}
			if err := ValidateFullpaths(fullpaths); err != nil {
				return err
			}
			if err := ValidateFullpath(targetFullpath); err != nil {
				return err
			}

			params := map[string]string{
				"target_mount_id": targetMountID,
				"target_fullpath": targetFullpath,
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if fullpaths != "" {
				params["fullpaths"] = fullpaths
			}
			if overwrite != "" {
				if overwrite != "0" && overwrite != "1" {
					return apperrors.NewValidation("overwrite 的值必须是 0 或 1")
				}
				params["overwrite"] = overwrite
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.CopyFile(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "要复制文件的路径(单文件复制)")
	cmd.Flags().String("fullpaths", "", "要复制文件的路径(多文件复制,|分隔)")
	cmd.Flags().String("target-mount-id", "", "复制到的mount_id")
	cmd.Flags().String("target-fullpath", "", "复制到的目录路径(不含文件名)")
	cmd.Flags().String("overwrite", "", "是否覆盖同名文件,0不覆盖,1覆盖,默认0")
	cmd.MarkFlagRequired("target-mount-id")
	cmd.MarkFlagRequired("target-fullpath")
	return cmd
}

func newFileDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "删除文件(夹)",
		Example: `  ykc file delete --mount-id 1 --fullpath documents/a.txt
  ykc file delete --mount-id 1 --fullpath documents/folder`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			if fullpath == "" {
				return apperrors.NewValidation("请提供 --fullpath 文件路径")
			}

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DeleteFile(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径(以 / 结尾) (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileRecycleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "recycle",
		Short:   "回收站列表",
		Example: `  ykc file recycle --mount-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			params := map[string]string{
				"start": fmt.Sprintf("%d", start),
				"size":  fmt.Sprintf("%d", size),
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetRecycleList(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	return cmd
}

func newFileRecoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "恢复已删除文件",
		Example: `  ykc file recover --mount-id 1 --fullpaths "documents/a.txt|documents/b.txt"
  ykc file recover --mount-id 1 --fullpaths "documents/a.txt" --machine "my-pc"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpaths, _ := cmd.Flags().GetString("fullpaths")
			fullpaths = normalizeMSYSPaths(fullpaths)
			if fullpaths != "" {
				parts := strings.Split(fullpaths, "|")
				for i, p := range parts {
					if p != "" && !strings.HasPrefix(p, "/") {
						parts[i] = "/" + p
					}
				}
				fullpaths = strings.Join(parts, "|")
			}
			machine, _ := cmd.Flags().GetString("machine")

			params := map[string]string{}
			if fullpaths != "" {
				params["fullpaths"] = fullpaths
			}
			if machine != "" {
				params["machine"] = machine
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.RecoverFiles(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpaths", "", "文件路径, |分隔 (必需)")
	cmd.Flags().String("machine", "", "操作机器")
	cmd.MarkFlagRequired("fullpaths")
	return cmd
}

func newFileHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "获取文件历史",
		Example: `  ykc file history --mount-id 1 --fullpath documents/a.txt
  ykc file history --mount-id 1 --hash abc123 --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			hash, _ := cmd.Flags().GetString("hash")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			params := map[string]string{
				"start": fmt.Sprintf("%d", start),
				"size":  fmt.Sprintf("%d", size),
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if hash != "" {
				params["hash"] = hash
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileHistoryV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件唯一标识")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 50, "返回条数")
	return cmd
}

// === File Lock Commands ===

func newFileLockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "文件锁操作",
	}

	cmd.AddCommand(
		newFileLockSetCommand(),
		newFileLockUnsetCommand(),
	)

	return cmd
}

func newFileLockSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set",
		Short:   "锁定文件",
		Example: `  ykc file lock set --mount-id 1 --fullpath documents/a.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			req := gokuai.FileLockRequest{
				MountID:  mountID,
				Fullpath: fullpath,
				Lock:     "lock",
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.LockFile(req, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileLockUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unset",
		Short:   "解锁文件",
		Example: `  ykc file lock unset --mount-id 1 --fullpath documents/a.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			req := gokuai.FileLockRequest{
				MountID:  mountID,
				Fullpath: fullpath,
				Lock:     "unlock",
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.LockFile(req, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileUpdatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updates",
		Short: "文件最近更新列表",
		Example: `  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508
  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508 --size 50
  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508 --act 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			if mountID <= 0 {
				return fmt.Errorf("--mount-id 必须为正整数")
			}
			size, _ := cmd.Flags().GetInt("size")
			fromDatelineMS, _ := cmd.Flags().GetInt64("from-datelinems")
			toDatelineMS, _ := cmd.Flags().GetInt64("to-datelinems")
			act, _ := cmd.Flags().GetInt("act")
			actMemberID, _ := cmd.Flags().GetInt("act-member-id")

			params := map[string]string{
				"from_datelinems": fmt.Sprintf("%d", fromDatelineMS),
				"to_datelinems":   fmt.Sprintf("%d", toDatelineMS),
				"size":            fmt.Sprintf("%d", size),
			}
			if cmd.Flags().Changed("act") {
				params["act"] = fmt.Sprintf("%d", act)
			}
			if actMemberID > 0 {
				params["act_member_id"] = fmt.Sprintf("%d", actMemberID)
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileUpdates(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int64("from-datelinems", 0, "开始毫秒 (必需)")
	cmd.Flags().Int64("to-datelinems", 0, "结束毫秒 (必需)")
	cmd.Flags().Int("size", 20, "数据长度")
	cmd.Flags().Int("act", 0, "操作类型")
	cmd.Flags().Int("act-member-id", 0, "操作人ID")
	cmd.MarkFlagRequired("from-datelinems")
	cmd.MarkFlagRequired("to-datelinems")
	return cmd
}

// === File V2 Commands (merged from filev2) ===

func newFileMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move",
		Short: "移动文件(夹)",
		Example: `  ykc file move --mount-id 1 --fullpath "a.txt" --target-mount-id 2 --target-fullpath "folder/"
  ykc file move --mount-id 1 --fullpaths "a.txt|b.txt" --target-mount-id 2 --target-fullpath "folder/"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			fullpaths, _ := cmd.Flags().GetString("fullpaths")
			fullpaths = normalizeMSYSPaths(fullpaths)
			if fullpaths != "" {
				parts := strings.Split(fullpaths, "|")
				for i, p := range parts {
					if p != "" && !strings.HasPrefix(p, "/") {
						parts[i] = "/" + p
					}
				}
				fullpaths = strings.Join(parts, "|")
			}
			targetMountID, _ := cmd.Flags().GetString("target-mount-id")
			targetFullpath, _ := cmd.Flags().GetString("target-fullpath")
			targetFullpath = normalizeMSYSPath(targetFullpath)
			if targetFullpath != "" && !strings.HasPrefix(targetFullpath, "/") {
				targetFullpath = "/" + targetFullpath
			}

			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}
			if err := ValidateFullpaths(fullpaths); err != nil {
				return err
			}
			if err := ValidateFullpath(targetFullpath); err != nil {
				return err
			}

			params := map[string]string{
				"target_mount_id": targetMountID,
				"target_fullpath": targetFullpath,
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if fullpaths != "" {
				params["fullpaths"] = fullpaths
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.MoveFile(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径(单文件)")
	cmd.Flags().String("fullpaths", "", "多文件路径(分隔)")
	cmd.Flags().String("target-mount-id", "", "移动到的mount_id (必需)")
	cmd.Flags().String("target-fullpath", "", "移动到的路径 (必需)")
	cmd.MarkFlagRequired("target-mount-id")
	cmd.MarkFlagRequired("target-fullpath")
	return cmd
}

func newFileRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename",
		Short:   "重命名文件(夹)",
		Example: `  ykc file rename --mount-id 1 --fullpath "a.txt" --newname "b.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}
			newname, _ := cmd.Flags().GetString("newname")
			if err := ValidateFileName(newname); err != nil {
				return err
			}

			params := map[string]string{
				"fullpath": fullpath,
				"newname":  newname,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.RenameFile(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("newname", "", "新的名称 (必需)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("newname")
	return cmd
}

func newFileV2AttributeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attr",
		Short:   "获取文件夹额外属性",
		Example: `  ykc file attr --mount-id 1 --fullpath "test"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")

			// Normalize for Windows: convert backslashes and ensure leading "/".
			fullpath = filepath.ToSlash(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileAttributeV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2OpenURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open-url",
		Short:   "获取文件打开URL",
		Example: `  ykc file open-url --mount-id 1 --fullpath "test.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileOpenURLV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2SaveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "save",
		Short:   "转存文件",
		Example: `  ykc file save --mount-id 1 --fullpath "test.txt" --target-mount-id 2 --target-fullpath "backup/"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			targetMountID, _ := cmd.Flags().GetString("target-mount-id")
			targetFullpath, _ := cmd.Flags().GetString("target-fullpath")
			targetFullpath = normalizeMSYSPath(targetFullpath)
			dialogID, _ := cmd.Flags().GetString("dialog-id")

			params := map[string]string{
				"fullpath":        fullpath,
				"target_mount_id": targetMountID,
				"target_fullpath": targetFullpath,
			}
			if dialogID != "" {
				params["dialog_id"] = dialogID
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.SaveFileV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "要转存文件fullpath(转存文件夹时)")
	cmd.Flags().String("target-mount-id", "", "转存到的mount_id (必需)")
	cmd.Flags().String("target-fullpath", "", "转存到的路径 (必需)")
	cmd.Flags().String("dialog-id", "", "会话ID(转存文件夹时)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("target-mount-id")
	cmd.MarkFlagRequired("target-fullpath")
	return cmd
}

func newFileV2SearchV2Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Short:   "搜索文件",
		Example: `  ykc file search --mount-id 1 --keyword "文档"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			keyword, _ := cmd.Flags().GetString("keyword")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			params := map[string]string{
				"keyword": keyword,
				"start":   fmt.Sprintf("%d", start),
				"size":    fmt.Sprintf("%d", size),
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.SearchFileV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("keyword", "", "搜索关键字 (必需)")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	cmd.MarkFlagRequired("keyword")
	return cmd
}

func newFileV2AddCommentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-comment",
		Short: "添加文件评论",
		Example: `  ykc file add-comment --mount-id 1 --fullpath "test.txt" --message "[@ id=all]@所有人[/@]开会"
  ykc file add-comment --mount-id 1 --fullpath "文档/a.pdf" --message "[@ id=1234567]@1122[/@]你好"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			message, _ := cmd.Flags().GetString("message")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			// 解析 message 中的 @成员 提及并替换为 [@ id=member_id]@成员名[/@]
			message = resolveMentions(message, func(name string) (string, bool) {
				if name == "所有人" || strings.ToLower(name) == "all" {
					return "all", true
				}
				resp, err := userClient.SearchLibraryMembersByKeyword(mountID, name, secret)
				if err != nil {
					return "", false
				}
				for _, m := range resp.List {
					if m.MemberName == name {
						return fmt.Sprintf("%d", int(m.MemberID)), true
					}
				}
				return "", false
			})

			if err := validateRemarkMessage(message); err != nil {
				return err
			}

			params := map[string]string{
				"fullpath": fullpath,
				"message":  message,
			}

			if err := userClient.AddFileRemarkV2(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.Flags().String("message", "", "评论内容 (必需)，提及须写成 [@ id=<id>]@<名称>[/@]，例如 [@ id=all]@所有人[/@] 或 [@ id=1234567]@1122[/@]；也支持输入 @成员 自动转换")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("message")
	return cmd
}

// resolveMentions 解析并替换 message 中的 @成员 提及。
// 匹配格式: @成员 后跟空格，或 @成员 位于字符串结尾。
// 替换结果: [@ id=<member_id>]@成员名[/@]；未找到的成员保持原样。
func resolveMentions(message string, lookup func(name string) (memberID string, found bool)) string {
	var sb strings.Builder
	i := 0
	n := len(message)
	for i < n {
		if message[i] == '@' && (i == 0 || isSpaceByte(message[i-1])) {
			j := i + 1
			for j < n && !isSpaceByte(message[j]) {
				j++
			}
			name := message[i+1 : j]
			if name != "" {
				if id, found := lookup(name); found {
					sb.WriteString("[@ id=")
					sb.WriteString(id)
					sb.WriteString("]@")
					sb.WriteString(name)
					sb.WriteString("[/@]")
					i = j
					continue
				}
			}
		}
		sb.WriteByte(message[i])
		i++
	}
	return sb.String()
}

func isSpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// remarkMentionRe 匹配合法的提及格式: [@ id=<id>]@<名称>[/@]
var remarkMentionRe = regexp.MustCompile(`\[@ id=[^\]]+\]@[^\[]*\[/@\]`)

// validateRemarkMessage 校验评论内容的提及格式。
// 允许纯文本；若包含提及，必须是 [@ id=<id>]@<名称>[/@] 格式
// （例如 [@ id=all]@所有人[/@] 或 [@ id=1234567]@1122[/@]）。
// 其它普通文本中含有 @ 不会触发校验。
func validateRemarkMessage(message string) error {
	// 移除所有合法提及后，若仍有 "[@" 或 "[/@]" 残片，说明存在格式错误的提及。
	stripped := remarkMentionRe.ReplaceAllString(message, "")
	if strings.Contains(stripped, "[@") || strings.Contains(stripped, "[/@]") {
		return fmt.Errorf("--message 提及格式不正确，正确格式如: [@ id=all]@所有人[/@] 或 [@ id=1234567]@1122[/@]")
	}
	return nil
}

func newFileV2GetCommentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-comment",
		Short:   "获取文件评论",
		Example: `  ykc file get-comment --mount-id 1 --fullpath "test.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileRemarkV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2CEditURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cedit-url",
		Short: "获取协同编辑链接",
		Example: `  ykc file cedit-url --mount-id 1 --fullpath "test.txt"
  ykc file cedit-url --mount-id 1 --fullpath "test.txt" --edit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			edit, _ := cmd.Flags().GetBool("edit")

			// Normalize for Windows: convert backslashes and ensure leading "/".
			fullpath = filepath.ToSlash(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}

			params := map[string]string{
				"fullpath": fullpath,
			}
			if edit {
				params["edit"] = "1"
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetCEditURLV2(mountID, params, secret)
			if err != nil {
				return err
			}

			if urlVal, ok := resp["url"]; ok {
				if u, ok := urlVal.(string); ok && u != "" {
					if !strings.HasPrefix(u, "http") {
						u = gokuai.GetAPIHost() + u
					}
					ssoURL, err := gokuai.BuildSSOURL(userClient.AccessToken, u, secret)
					if err != nil {
						return err
					}
					resp["url"] = ssoURL
				}
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.Flags().Bool("edit", false, "编辑模式 (不添加此参数则为预览)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2AddLifecycleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-lifecycle",
		Short: "设置文件生命周期",
		Example: `  ykc file add-lifecycle --mount-id 1 --fullpath /documents/ --type daysago --rule 30
  ykc file add-lifecycle --mount-id 1 --fullpath /test/ --type expire --rule 2019-10-10
  ykc file add-lifecycle --mount-id 1 --fullpath /documents/ --type time --rule day-6 --retainfolder --destroy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath == "/" {
				return apperrors.NewValidation("--fullpath 不支持根目录")
			}
			if !strings.HasSuffix(fullpath, "/") {
				return apperrors.NewValidation(fmt.Sprintf("--fullpath 必须是文件夹路径(以 / 结尾)，不能是文件路径: %s", fullpath))
			}
			lifecycleType, _ := cmd.Flags().GetString("type")
			rule, _ := cmd.Flags().GetString("rule")
			rule = strings.ToLower(rule)
			retainfolder, _ := cmd.Flags().GetBool("retainfolder")
			destroy, _ := cmd.Flags().GetBool("destroy")

			if err := ValidateLifecycleRule(lifecycleType, rule); err != nil {
				return err
			}

			params := map[string]string{
				"type": lifecycleType,
				"rule": rule,
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if retainfolder {
				params["retainfolder"] = "1"
			}
			if destroy {
				params["destroy"] = "1"
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.AddLifecycleV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("type", "", "类型: daysago自动清理, expire到期清理, time定时清理 (必需)")
	cmd.Flags().String("rule", "", "规则: daysago时为天数, expire时为日期, time时为频率(格式 day-N/week-N/month-N) (必需)")
	cmd.Flags().Bool("retainfolder", false, "保留文件夹结构")
	cmd.Flags().Bool("destroy", false, "彻底删除 (不添加此参数则移入回收站)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("rule")
	return cmd
}

func newFileV2DelLifecycleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del-lifecycle",
		Short:   "删除文件生命周期",
		Example: `  ykc file del-lifecycle --mount-id 1 --fullpath "test.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DelLifecycleV2(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2GetLifecycleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-lifecycle",
		Short:   "获取文件生命周期",
		Example: `  ykc file get-lifecycle --mount-id 1 --fullpath "test.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetLifecycleV2(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2SetRemindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-remind",
		Short: "设置文件到期提醒",
		Example: `  ykc file set-remind --mount-id 1 --fullpath "test.txt" --expire 1735660800 --reminder "1,2,3"
  ykc file set-remind --mount-id 1 --fullpath "test.txt" --expire 1735660800 --timer "day-9,11-30" --reminder "1,2,3"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath == "" {
				return apperrors.NewValidation("请提供 --fullpath 文件路径")
			}
			expire, _ := cmd.Flags().GetString("expire")
			timer, _ := cmd.Flags().GetString("timer")
			reminder, _ := cmd.Flags().GetString("reminder")
			machine, _ := cmd.Flags().GetString("machine")
			fileProps, _ := cmd.Flags().GetStringSlice("file-props")
			remark, _ := cmd.Flags().GetString("remark")

			expireVal, err := strconv.ParseInt(expire, 10, 64)
			if err != nil || expireVal <= 0 {
				return apperrors.NewValidation("--expire 必须是大于 0 的时间戳")
			}
			if err := ValidateRemindTimer(timer); err != nil {
				return err
			}

			params := map[string]string{
				"fullpath": fullpath,
				"reminder": reminder,
				"expire":   expire,
			}
			if timer != "" {
				params["timer"] = timer
			}
			if machine != "" {
				params["machine"] = machine
			}
			if len(fileProps) > 0 {
				data, err := buildRemindData(fileProps)
				if err != nil {
					return err
				}
				params["data"] = data
			}
			if remark != "" {
				params["remark"] = remark
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.SetFileRemindV2(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("expire", "", "到期时间戳(秒级)")
	cmd.Flags().String("timer", "", "定时提醒, 格式: day-9,11-30;week-3,7-14,17-00;month-1,11,21-10,16-00")
	cmd.Flags().String("reminder", "", "被提醒人(,隔开的member_id)，每次设置都会覆盖")
	cmd.Flags().String("machine", "", "操作机器")
	cmd.Flags().StringSlice("file-props", nil, "文件属性字段, 可多个(逗号分隔): fullpath,last_dateline,last_member_id,filesize,create_dateline,create_member_id")
	cmd.Flags().String("remark", "", "备注说明")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("reminder")
	cmd.MarkFlagRequired("expire")
	return cmd
}

// filePropsAllowed 是 set-remind 命令 --file-props 允许的值。
var filePropsAllowed = []string{
	"fullpath", "last_dateline", "last_member_id", "filesize", "create_dateline", "create_member_id",
}

// buildRemindData 根据 --file-props 组装接口 data 参数 JSON 字符串：
// {"file":["fullpath",...],"metadata":{}}
func buildRemindData(fileProps []string) (string, error) {
	for _, p := range fileProps {
		valid := false
		for _, a := range filePropsAllowed {
			if p == a {
				valid = true
				break
			}
		}
		if !valid {
			return "", apperrors.NewValidation(fmt.Sprintf("--file-props 的值 %q 不合法，允许值: %s", p, strings.Join(filePropsAllowed, ", ")))
		}
	}

	dataStruct := map[string]interface{}{
		"file":     fileProps,
		"metadata": map[string]interface{}{},
	}
	dataBytes, err := json.Marshal(dataStruct)
	if err != nil {
		return "", fmt.Errorf("组装 data 参数失败: %w", err)
	}
	return string(dataBytes), nil
}

func newFileV2DelRemindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del-remind",
		Short:   "删除文件提醒",
		Example: `  ykc file del-remind --mount-id 1 --fullpath "test.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.DelFileRemindV2(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === New v1 Commands ===

func newFileClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clear",
		Short:   "清空回收站",
		Example: `  ykc file clear --mount-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.ClearRecycle(mountID, map[string]string{}, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

func newFileDeleteCompletelyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-completely",
		Short: "彻底删除文件",
		Example: `  ykc file delete-completely --mount-id 1 --fullpath documents/a.txt
  ykc file delete-completely --mount-id 1 --fullpaths "a.txt|b.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			fullpaths, _ := cmd.Flags().GetString("fullpaths")
			fullpaths = normalizeMSYSPaths(fullpaths)
			if fullpaths != "" {
				parts := strings.Split(fullpaths, "|")
				for i, p := range parts {
					if p != "" && !strings.HasPrefix(p, "/") {
						parts[i] = "/" + p
					}
				}
				fullpaths = strings.Join(parts, "|")
			}

			params := map[string]string{}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if fullpaths != "" {
				params["fullpaths"] = fullpaths
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.DeleteCompletely(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "单个文件路径")
	cmd.Flags().String("fullpaths", "", "多个文件路径, |分隔")
	return cmd
}

func newFileQueueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "queue",
		Short:   "检测文件执行队列状态",
		Example: `  ykc file queue --mount-id 1 --queue-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			queueID, _ := cmd.Flags().GetString("queue-id")
			queueType, _ := cmd.Flags().GetString("type")

			params := map[string]string{}
			if queueID != "" {
				params["queue_id"] = queueID
			}
			if queueType != "" {
				params["type"] = queueType
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetQueueStatus(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("queue-id", "", "队列ID,多个以逗号隔开")
	cmd.Flags().String("type", "", "队列类型：不传：文件批处理；metadata：元数据批处理；zip：压缩包批处理")
	return cmd
}

func newFileKeywordCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keyword",
		Short:   "设置文件标签",
		Example: `  ykc file keyword --mount-id 1 --fullpath documents/a.txt --keywords '秋天;很美'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			hash, _ := cmd.Flags().GetString("hash")
			keywords, _ := cmd.Flags().GetString("keywords")

			params := map[string]string{
				"keywords": keywords,
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if hash != "" {
				params["hash"] = hash
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.SetFileKeyword(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件唯一标识")
	cmd.Flags().String("keywords", "", "标签, 分号分隔, 如 '秋天;很美' (必需)")
	cmd.MarkFlagRequired("keywords")
	return cmd
}

// === File Permission Commands ===

func newFilePermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission",
		Short: "文件夹权限操作",
	}

	cmd.AddCommand(
		newFilePermissionMemberCommand(),
		newFilePermissionGroupCommand(),
		newFilePermissionSetCommand(),
		newFilePermissionSetInheritCommand(),
		newFilePermissionResetCommand(),
		newFilePermissionResetAllCommand(),
	)

	return cmd
}

func newFilePermissionMemberCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "获取文件夹成员权限",
		Long: `获取文件夹成员权限 (GET /m-api/1/file/get_member_permissions)。
指定 --format csv 时导出所有用户的权限，CSV 内容为 GBK 编码:
  姓名,邮箱,预览,下载,上传,编辑,删除,外链`,
		Example: `  ykc file permission member --mount-id 1 --fullpath documents
  ykc file permission member --mount-id 1 --fullpath documents --keyword 张三
  ykc file permission member --mount-id 1 --fullpath documents --format csv
  ykc file permission member --mount-id 1 --fullpath documents --on-group`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")
			keyword, _ := cmd.Flags().GetString("keyword")
			format, _ := cmd.Flags().GetString("format")
			onGroup, _ := cmd.Flags().GetBool("on-group")
			isOut, _ := cmd.Flags().GetBool("is-out")

			params := map[string]string{
				"fullpath": fullpath,
				"start":    fmt.Sprintf("%d", start),
				"size":     fmt.Sprintf("%d", size),
			}
			if keyword != "" {
				params["keyword"] = keyword
			}
			if format != "" {
				params["format"] = format
			}
			if onGroup {
				params["on_group"] = "1"
			}
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

			resp, err := userClient.GetMemberPermissions(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().Int("start", 0, "开始位置 (必需)")
	cmd.Flags().Int("size", 20, "获取数量 (必需)")
	cmd.Flags().String("keyword", "", "成员名关键字")
	cmd.Flags().String("format", "", "为 csv 时导出所有用户的权限(CSV 内容为 GBK 编码)")
	cmd.Flags().Bool("on-group", false, "显示继承部门权限的成员(设置文件夹权限不继承情况下)")
	cmd.Flags().Bool("is-out", false, "是否外部成员")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFilePermissionGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "group",
		Short:   "获取文件夹部门权限",
		Example: `  ykc file permission group --mount-id 1 --fullpath documents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetGroupPermissions(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFilePermissionSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set",
		Short:   "设置文件夹权限",
		Example: `  ykc file permission set --mount-id 1 --fullpath documents --permission '{"1420729":["file_export","admin","file_list"]}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			permission, _ := cmd.Flags().GetString("permission")
			isGroup, _ := cmd.Flags().GetBool("is-group")
			isOut, _ := cmd.Flags().GetBool("is-out")

			params := map[string]string{
				"fullpath":   fullpath,
				"permission": permission,
			}
			if isGroup {
				params["is_group"] = "1"
			}
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

			if err := userClient.SetPermission(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().String("permission", "", "要重置权限的成员ID数组, 如 '[1420729]' (必需)")
	cmd.Flags().Bool("is-group", false, "是否部门权限")
	cmd.Flags().Bool("is-out", false, "是否外部成员")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("permission")
	return cmd
}

func newFilePermissionSetInheritCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-inherit",
		Short:   "设置文件夹权限是否继承",
		Example: `  ykc file permission set-inherit --mount-id 1 --fullpath documents --inherit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			inherit, _ := cmd.Flags().GetBool("inherit")

			inheritVal := "0"
			if inherit {
				inheritVal = "1"
			}
			params := map[string]string{
				"fullpath": fullpath,
				"inherit":  inheritVal,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			if err := userClient.SetPermissionInherit(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().Bool("inherit", false, "继承权限 (不添加此参数则为不继承)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFilePermissionResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset",
		Short:   "重置文件夹权限",
		Example: `  ykc file permission reset --mount-id 1 --fullpath documents --permission '[1420729]'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			permission, _ := cmd.Flags().GetString("permission")
			if strings.TrimSpace(permission) == "" {
				return apperrors.NewValidation("请提供 --permission 权限设置")
			}
			isGroup, _ := cmd.Flags().GetBool("is-group")
			isOut, _ := cmd.Flags().GetBool("is-out")

			params := map[string]string{
				"fullpath":   fullpath,
				"permission": permission,
			}
			if isGroup {
				params["is_group"] = "1"
			}
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

			if err := userClient.ResetPermission(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().String("permission", "", "权限设置 (必需)")
	cmd.Flags().Bool("is-group", false, "是否部门权限")
	cmd.Flags().Bool("is-out", false, "是否外部成员")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("permission")
	return cmd
}

func newFilePermissionResetAllCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-all",
		Short: "重置文件夹所有权限",
		Example: `  ykc file permission reset-all --mount-id 1 --fullpath documents --type member
  ykc file permission reset-all --mount-id 1 --fullpath documents --type group
  ykc file permission reset-all --mount-id 1 --fullpath documents --type out`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if err := checkStringFlagInSet(cmd, "type", "member", "group", "out"); err != nil {
				return err
			}
			permType, _ := cmd.Flags().GetString("type")

			params := map[string]string{
				"fullpath": fullpath,
			}
			switch permType {
			case "member":
				params["is_group"] = "0"
			case "group":
				params["is_group"] = "1"
			case "out":
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

			if err := userClient.ResetAllPermission(mountID, params, secret); err != nil {
				return err
			}

			return outputJSON(cmd, map[string]string{"result": "success"})
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().String("type", "", "重置哪类权限: member成员, group部门, out外部成员 (必需)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("type")
	return cmd
}

// === New V2 Commands ===

func newFileV2ExistCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exist",
		Short:   "文件是否存在",
		Example: `  ykc file exist --mount-id 1 --fullpath documents/a.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			hash, _ := cmd.Flags().GetString("hash")

			params := map[string]string{}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if hash != "" {
				params["hash"] = hash
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.CheckFileExist(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件唯一标识")
	return cmd
}

func newFileV2DownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "获取文件地址并下载",
		Example: `  ykc file download --mount-id 1 --hash abc123
  ykc file download --mount-id 1 --hash abc123 --output ./file.pdf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			hash, _ := cmd.Flags().GetString("hash")
			fullpath, _ := cmd.Flags().GetString("fullpath")

			params := map[string]string{}
			if hash != "" {
				params["hash"] = hash
			}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileURL(mountID, params, secret)
			if err != nil {
				return err
			}

			outputPath, _ := cmd.Flags().GetString("output")
			if outputPath != "" {
				outputPath = expandHomePath(outputPath)
				// Extract download URL from response
				downloadURL := ""
				if urls, ok := resp["uris"]; ok {
					if arr, ok := urls.([]interface{}); ok && len(arr) > 0 {
						downloadURL, _ = arr[0].(string)
					}
				}
				if downloadURL == "" {
					return apperrors.NewAPI("响应中未找到下载地址")
				}
				return downloadFile(cmd, downloadURL, outputPath)
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("hash", "", "文件唯一标识")
	cmd.Flags().String("fullpath", "", "文件完整路径")
	cmd.Flags().String("output", "", "下载文件保存路径（指定后直接下载文件）")
	return cmd
}

func newFileV2RevertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revert",
		Short:   "还原历史版本",
		Example: `  ykc file revert --mount-id 1 --fullpath documents/a.txt --hid 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			if fullpath != "" && !strings.HasPrefix(fullpath, "/") {
				fullpath = "/" + fullpath
			}
			hid, _ := cmd.Flags().GetString("hid")

			params := map[string]string{}
			if fullpath != "" {
				params["fullpath"] = fullpath
			}
			if hid != "" {
				params["hid"] = hid
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.RevertFileHistory(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("hid", "", "历史版本ID (必需)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("hid")
	return cmd
}

func newFileV2ImageSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-search",
		Short: "图片搜索",
		Example: `  ykc file image-search --mount-id 1 --keyword "风景"
  ykc file image-search --mount-id 1 --file ./photo.jpg
  ykc file image-search --mount-id 1 --image-mount-id 1 --image-hash abc123
  ykc file image-search --mount-id 1 --file ./photo.jpg --keyword "风景"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			filePath, _ := cmd.Flags().GetString("file")
			imageMountID, _ := cmd.Flags().GetString("image-mount-id")
			imageHash, _ := cmd.Flags().GetString("image-hash")
			keyword, _ := cmd.Flags().GetString("keyword")
			ext, _ := cmd.Flags().GetString("ext")
			size, _ := cmd.Flags().GetInt("size")
			createMemberID, _ := cmd.Flags().GetString("create-member-id")
			lastMemberID, _ := cmd.Flags().GetString("last-member-id")

			hasKeyword := keyword != ""
			hasFile := filePath != ""
			hasImageRef := imageMountID != "" && imageHash != ""
			if !hasKeyword && !hasFile && !hasImageRef {
				return apperrors.NewValidation("必须提供 --keyword、--file 或 --image-mount-id 与 --image-hash 之一")
			}
			if imageMountID != "" && imageHash == "" {
				return apperrors.NewValidation("--image-mount-id 与 --image-hash 需同时提供")
			}
			if imageMountID == "" && imageHash != "" {
				return apperrors.NewValidation("--image-mount-id 与 --image-hash 需同时提供")
			}
			if hasFile && hasImageRef {
				return apperrors.NewValidation("--file 与 --image-mount-id/--image-hash 不能同时提供")
			}

			params := map[string]string{}
			if keyword != "" {
				params["keyword"] = keyword
			}
			if ext != "" {
				params["ext"] = ext
			}
			if size > 0 {
				params["size"] = fmt.Sprintf("%d", size)
			}
			if createMemberID != "" {
				params["create_member_id"] = createMemberID
			}
			if lastMemberID != "" {
				params["last_member_id"] = lastMemberID
			}
			if imageMountID != "" {
				params["image_mount_id"] = imageMountID
			}
			if imageHash != "" {
				params["image_hash"] = imageHash
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			var resp map[string]interface{}
			if hasFile {
				absPath, err := filepath.Abs(expandHomePath(filePath))
				if err != nil {
					return fmt.Errorf("解析图片路径失败: %w", err)
				}
				if err := validateImageFile(absPath); err != nil {
					return err
				}
				fileBytes, err := os.ReadFile(absPath)
				if err != nil {
					return apperrors.NewValidation(fmt.Sprintf("读取图片文件失败: %v", err))
				}
				resp, err = userClient.ImageSearchUpload(mountID, params, fileBytes, filepath.Base(absPath), secret)
			} else {
				resp, err = userClient.ImageSearch(mountID, params, secret)
			}
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("file", "", "本地图片文件路径 (仅支持 jpeg/jpg/png/webp/bmp/tif/tiff，单张最大 20MB)")
	cmd.Flags().String("image-mount-id", "", "库中图片所在库ID (与 --image-hash 一起使用，限制同 --file)")
	cmd.Flags().String("image-hash", "", "库中图片的 hash (与 --image-mount-id 一起使用，限制同 --file)")
	cmd.Flags().String("keyword", "", "搜索关键字")
	cmd.Flags().String("ext", "", "图片扩展名")
	cmd.Flags().Int("size", 20, "返回条数")
	cmd.Flags().String("create-member-id", "", "文件创建者的 member_id")
	cmd.Flags().String("last-member-id", "", "文件修改者的 member_id")
	return cmd
}

// validateImageFile 校验本地图片文件格式与大小。
// 仅支持 jpeg/jpg/png/webp/bmp/tif/tiff，单张最大 20MB。
func validateImageFile(path string) error {
	allowedExts := map[string]bool{
		".jpeg": true,
		".jpg":  true,
		".png":  true,
		".webp": true,
		".bmp":  true,
		".tif":  true,
		".tiff": true,
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !allowedExts[ext] {
		return apperrors.NewValidation(fmt.Sprintf("不支持的图片格式: %s，仅支持 jpeg/jpg/png/webp/bmp/tif/tiff", ext))
	}
	info, err := os.Stat(path)
	if err != nil {
		return apperrors.NewValidation(fmt.Sprintf("图片文件不存在: %s", path))
	}
	if info.Size() > 20*1024*1024 {
		return apperrors.NewValidation("图片大小超过限制 (单张最大 20MB)")
	}
	return nil
}

func newFileV2GetRemindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-remind",
		Short:   "获取文件到期提醒",
		Example: `  ykc file get-remind --mount-id 1 --fullpath documents/a.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mountID, _ := cmd.Flags().GetInt("mount-id")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)

			params := map[string]string{
				"fullpath": fullpath,
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileRemind(mountID, params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

func newFileV2GetRemindsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-reminds",
		Short: "获取文件到期提醒列表",
		Example: `  ykc file get-reminds
  ykc file get-reminds --is-expire 1 --start 0 --size 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			isExpire, _ := cmd.Flags().GetInt("is-expire")
			start, _ := cmd.Flags().GetInt("start")
			size, _ := cmd.Flags().GetInt("size")

			params := map[string]string{}
			if isExpire != 0 {
				params["is_expire"] = fmt.Sprintf("%d", isExpire)
			}
			if start > 0 {
				params["start"] = fmt.Sprintf("%d", start)
			}
			if size > 0 {
				params["size"] = fmt.Sprintf("%d", size)
			}

			userClient, err := getUserClient()
			if err != nil {
				return err
			}
			secret, err := getSecret()
			if err != nil {
				return err
			}

			resp, err := userClient.GetFileReminds(params, secret)
			if err != nil {
				return err
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("is-expire", 0, "是否过期: -1未过期, 1已过期")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	// get-reminds 不区分库空间，用本地同名 flag 覆盖父命令的持久化 mount-id，
	// 使其既不必填也不显示在帮助中。
	cmd.Flags().Int("mount-id", 0, "")
	cmd.Flags().MarkHidden("mount-id")
	return cmd
}
