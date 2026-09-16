package app

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/gokuai/yunku-cli/internal/errors"
	"github.com/gokuai/yunku-cli/pkg/gokuai"
	"github.com/spf13/cobra"
)

// newEntFileCommand creates the enterprise file operation command group
func newEntFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "库文件操作",
		Long: `库文件操作，包括文件列表、上传、下载、搜索、权限管理等
库授权获取方式二选一:
  1. 自动绑定: 传 --org-id 或 --mount-id (需企业凭证 GOKUAI_CLIENT_ID/GOKUAI_CLIENT_SECRET)，
     绑定结果缓存 1 小时，期间后续命令可省略这两个参数
  2. 直接配置: 设置环境变量 GOKUAI_ORG_CLIENT_ID 和 GOKUAI_ORG_CLIENT_SECRET`,
	}

	cmd.PersistentFlags().String("org-client-id", "", "库授权client_id (或设置 GOKUAI_ORG_CLIENT_ID 环境变量)")
	cmd.PersistentFlags().String("org-client-secret", "", "库授权client_secret (或设置 GOKUAI_ORG_CLIENT_SECRET 环境变量)")
	cmd.PersistentFlags().String("org-id", "", "库ID, 自动获取库授权时与 mount-id 二选一 (绑定后缓存 1 小时，可省略)")
	cmd.PersistentFlags().Int("mount-id", 0, "库空间ID, 自动获取库授权时与 org-id 二选一 (绑定后缓存 1 小时，可省略)")
	cmd.PersistentFlags().String("title", "", "对接库文件的应用或系统名称, 自动获取库授权时使用")

	// 已直接配置库授权且无企业凭证时，自动绑定不可用，隐藏绑定参数
	if os.Getenv("GOKUAI_ORG_CLIENT_ID") != "" && os.Getenv("GOKUAI_CLIENT_ID") == "" {
		_ = cmd.PersistentFlags().MarkHidden("org-id")
		_ = cmd.PersistentFlags().MarkHidden("mount-id")
	}

	cmd.AddCommand(
		newEntFileLsCommand(),
		newEntFileUpdatesCommand(),
		newEntFileUpdatesCountCommand(),
		newEntFileDownloadCommand(),
		newEntFilePreviewURLCommand(),
		newEntFileExportURLCommand(),
		newEntFileCeditURLCommand(),
		newEntFileInfoCommand(),
		newEntFileSearchCommand(),
		newEntFileCreateFolderCommand(),
		newEntFileCreateFileCommand(),
		newEntFileUploadServersCommand(),
		newEntFileCopyCommand(),
		newEntFileMcopyCommand(),
		newEntFileMoveCommand(),
		newEntFileDelCommand(),
		newEntFileDelCompletelyCommand(),
		newEntFileRecycleCommand(),
		newEntFileRecoverCommand(),
		newEntFileHistoryCommand(),
		newEntFileLinkCommand(),
		newEntFileLinkCloseCommand(),
		newEntFileLinksCommand(),
		newEntFileLockCommand(),
		newEntFileSetPermissionInheritCommand(),
		newEntFileGetAllPermissionCommand(),
		newEntFileFilePermissionCommand(),
		newEntFileResetPermissionCommand(),
		newEntFileGetPermissionCommand(),
		newEntFileAddTagCommand(),
		newEntFileDelTagCommand(),
		newEntFileSetMetadataCommand(),
		newEntFileDelMetadataCommand(),
		newEntFileStatCommand(),
		newEntFileQueueStatusCommand(),
	)

	return cmd
}

// === 文件列表 ===

func newEntFileLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "文件列表",
		Example: `  ykc ent file ls --mount-id 1
  ykc ent file ls --mount-id 1 --fullpath /文档 --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			start, _ := cmd.Flags().GetInt("start")
			if start < 0 {
				return fmt.Errorf("--start 必须是0或正整数, 当前值: %d", start)
			}
			size, _ := cmd.Flags().GetInt("size")
			if size <= 0 {
				return fmt.Errorf("--size 必须是大于0的整数, 当前值: %d", size)
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("tag"); v != "" {
				params["tag"] = v
			}
			if start > 0 {
				params["start"] = fmt.Sprintf("%d", start)
			}
			params["size"] = fmt.Sprintf("%d", size)
			if v, _ := cmd.Flags().GetString("hashs"); v != "" {
				params["hashs"] = v
			}
			resp, err := client.GetFileList(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("tag", "", "标签")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	cmd.Flags().String("hashs", "", "文件hashs")
	return cmd
}

// === 文件最近更新列表 ===

func newEntFileUpdatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updates",
		Short: "文件最近更新列表",
		Example: `  ykc ent file updates --mount-id 1
  ykc ent file updates --mount-id 1 --fetch-dateline 1700000000000
  ykc ent file updates --mount-id 1 --mode compare --fetch-dateline 1700000000000
  ykc ent file updates --mount-id 1 --mode compare --fetch-dateline 0 --dir 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("mode"); v != "" {
				params["mode"] = v
			}
			if cmd.Flags().Changed("fetch-dateline") {
				v, _ := cmd.Flags().GetInt("fetch-dateline")
				params["fetch_dateline"] = strconv.Itoa(v)
			}
			if cmd.Flags().Changed("dir") {
				v, _ := cmd.Flags().GetInt("dir")
				params["dir"] = strconv.Itoa(v)
			}
			resp, err := client.GetFileUpdates(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("mode", "", "获取模式，空(倒序获取)或compare(正序获取，含已删除文件)")
	cmd.Flags().Int("fetch-dateline", 0, "Unix时间戳(毫秒)，获取此时间之后的更新，默认0")
	cmd.Flags().Int("dir", 0, "1只返回文件夹, 0只返回文件, 不传返回全部")
	return cmd
}

// === 文件更新数量 ===

func newEntFileUpdatesCountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "updates-count",
		Short:   "文件更新数量",
		Example: `  ykc ent file updates-count --mount-id 1 --begin-dateline 1700000000000 --end-dateline 1700100000000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			v1, _ := cmd.Flags().GetInt("begin-dateline")
			params["begin_dateline"] = strconv.Itoa(v1)
			v2, _ := cmd.Flags().GetInt("end-dateline")
			params["end_dateline"] = strconv.Itoa(v2)
			v, _ := cmd.Flags().GetInt("showdel")
			params["showdel"] = strconv.Itoa(v)
			resp, err := client.GetFileUpdatesCount(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("begin-dateline", 0, "开始时间戳，单位毫秒 (必需)")
	cmd.Flags().Int("end-dateline", 0, "结束时间戳，单位毫秒 (必需)")
	cmd.Flags().Int("showdel", 0, "1同时返回删除的文件，默认0不返回")
	cmd.MarkFlagRequired("begin-dateline")
	cmd.MarkFlagRequired("end-dateline")
	return cmd
}

// === 文件下载链接 ===

func newEntFileDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download",
		Aliases: []string{"download-url"},
		Short:   "获取文件下载链接或直接下载文件",
		Example: `  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file download --mount-id 1 --hash abc123 --open
  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt --output ./文件.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt("mount-id"); v > 0 {
				params["mount_id"] = fmt.Sprintf("%d", v)
			}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetString("filehash"); v != "" {
				params["filehash"] = v
			}
			if v, _ := cmd.Flags().GetInt("open"); v == 1 {
				params["open"] = "1"
			} else {
				params["open"] = "0"
			}
			if v, _ := cmd.Flags().GetString("filename"); v != "" {
				params["filename"] = v
			}
			if v, _ := cmd.Flags().GetString("net"); v != "" {
				params["net"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.GetDownloadURL(params)
			if err != nil {
				return err
			}

			outputPath, _ := cmd.Flags().GetString("output")
			if outputPath != "" && len(resp.URLs) > 0 {
				return downloadFile(cmd, resp.URLs[0], outputPath)
			}

			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件hash")
	cmd.Flags().String("filehash", "", "文件内容hash")
	cmd.Flags().Int("open", 0, "是否直接打开: 1打开, 0不打开")
	cmd.Flags().String("filename", "", "文件名")
	cmd.Flags().String("net", "", "网络类型")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.Flags().String("output", "", "下载文件保存路径（指定后直接下载文件）")
	return cmd
}

// === 文件预览链接 ===

func newEntFilePreviewURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview-url",
		Short: "文件预览链接",
		Example: `  ykc ent file preview-url --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file preview-url --mount-id 1 --hash abc123 --watermark`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetBool("watermark"); v {
				params["watermark"] = "1"
			}
			if v, _ := cmd.Flags().GetString("wm-content"); v != "" {
				params["wm_content"] = v
			}
			if v, _ := cmd.Flags().GetString("member-name"); v != "" {
				params["member_name"] = v
			}
			if v, _ := cmd.Flags().GetBool("thumbnail"); v {
				params["thumbnail"] = "1"
			}
			if v, _ := cmd.Flags().GetString("annotation"); v != "" {
				params["annotation"] = v
			}
			if v, _ := cmd.Flags().GetString("annotation-mode"); v != "" {
				params["annotation_mode"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			resp, err := client.GetPreviewURL(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件hash")
	cmd.Flags().Bool("watermark", false, "是否启用水印")
	cmd.Flags().String("wm-content", "", "水印内容")
	cmd.Flags().String("member-name", "", "成员名称")
	cmd.Flags().Bool("thumbnail", false, "是否缩略图")
	cmd.Flags().String("annotation", "", "批注")
	cmd.Flags().String("annotation-mode", "", "批注模式")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("out-id", "", "外部ID")
	cmd.Flags().String("account", "", "帐号")
	return cmd
}

// === 获取文件导出链接 ===

func newEntFileExportURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-url",
		Short: "获取文件导出链接",
		Example: `  ykc ent file export-url --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file export-url --mount-id 1 --hash abc123 --watermark`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetBool("watermark"); v {
				params["watermark"] = "1"
			}
			if v, _ := cmd.Flags().GetString("wm-content"); v != "" {
				params["wm_content"] = v
			}
			resp, err := client.GetExportURL(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件hash")
	cmd.Flags().Bool("watermark", false, "是否启用水印")
	cmd.Flags().String("wm-content", "", "水印内容")
	return cmd
}

// === 文件协同编辑链接 ===

func newEntFileCeditURLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cedit-url",
		Short: "文件协同编辑链接",
		Example: `  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --op-id 1
  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --out-id 1
  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --account 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetBool("readonly"); v {
				params["readonly"] = "1"
			}
			if v, _ := cmd.Flags().GetString("timeout"); v != "" {
				params["timeout"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			resp, err := client.GetCeditURL(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径, 需传 fullpath 或 hash 其中一个")
	cmd.Flags().String("hash", "", "文件唯一标识, 需传 fullpath 或 hash 其中一个")
	cmd.Flags().Bool("readonly", false, "1表示只读打开, 默认允许编辑")
	cmd.Flags().String("timeout", "", "编辑链接过期时间, 单位秒, 默认 3600")
	cmd.Flags().String("op-id", "", "操作人ID, 可以用 out-id 或 account 代替")
	cmd.Flags().String("out-id", "", "操作人外部系统帐号ID")
	cmd.Flags().String("account", "", "操作人外部系统帐号")
	return cmd
}

// === 文件(夹)信息 ===

func newEntFileInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "文件(夹)信息",
		Example: `  ykc ent file info --mount-id 1 --fullpath "/文档/readme.pdf"
  ykc ent file info --mount-id 1 --hash abc123
  ykc ent file info --mount-id 1 --fullpath "/文档" --attribute 1
  ykc ent file info --mount-id 1 --fullpath "/文档/readme.pdf" --ignores "tag,favorite,permission"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetString("net"); v != "" {
				params["net"] = v
			}
			if v, _ := cmd.Flags().GetString("attribute"); v != "" {
				params["attribute"] = v
			}
			if v, _ := cmd.Flags().GetString("hid"); v != "" {
				params["hid"] = v
			}
			if v, _ := cmd.Flags().GetString("ignores"); v != "" {
				params["ignores"] = v
			}
			resp, err := client.GetFileInfo(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径, 需传fullpath或hash其中一个")
	cmd.Flags().String("hash", "", "文件唯一标识, 需传fullpath或hash其中一个")
	cmd.Flags().String("net", "", "in表示获取内网下载链接, 默认空返回公网链接")
	cmd.Flags().String("attribute", "", "1获取额外属性(子文件数量/大小等), 10只返回基础信息, 默认0")
	cmd.Flags().String("hid", "", "版本ID")
	cmd.Flags().String("ignores", "", "忽略返回信息, 逗号隔开, 可选: tag,favorite,property,permission,lock,url,preview,thumbnail")
	return cmd
}

// === 文件搜索 ===

func newEntFileSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "文件搜索",
		Example: `  ykc ent file search --mount-id 1 --keywords "报告"
  ykc ent file search --mount-id 1 --keywords "报告" --path /文档 --size 50
  ykc ent file search --mount-id 1 --keywords "合同" --scope '["filename","content"]'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, _ := cmd.Flags().GetInt("start")
			if start < 0 {
				return apperrors.NewValidation(fmt.Sprintf("--start 必须是0或正整数, 当前值: %d", start))
			}
			size, _ := cmd.Flags().GetInt("size")
			if size <= 0 {
				return apperrors.NewValidation(fmt.Sprintf("--size 必须是大于0的整数, 当前值: %d", size))
			}
			if v, _ := cmd.Flags().GetString("scope"); v != "" {
				if err := validateSearchScope(v); err != nil {
					return err
				}
			}

			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("keywords"); v != "" {
				params["keywords"] = v
			}
			if v, _ := cmd.Flags().GetString("path"); v != "" {
				params["path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("scope"); v != "" {
				params["scope"] = v
			}
			if start > 0 {
				params["start"] = fmt.Sprintf("%d", start)
			}
			params["size"] = fmt.Sprintf("%d", size)
			resp, err := client.SearchFiles(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("keywords", "", "搜索关键字 (必需)")
	cmd.Flags().String("path", "", "需要搜索的文件夹, 默认空搜索整个库")
	cmd.Flags().String("scope", "", `搜索范围, JSON数组字符串, 可选值: filename(文件名)、tag(标签)、content(全文), 默认["filename","tag"], 例如: '["filename","content"]'`)
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	cmd.MarkFlagRequired("keywords")
	return cmd
}

// === 创建文件夹 ===

func newEntFileCreateFolderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create-folder",
		Short:   "创建文件夹",
		Example: `  ykc ent file create-folder --mount-id 1 --fullpath /文档/新文件夹`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				v = normalizeMSYSPath(v)
				if err := ValidateFullpath(v); err != nil {
					return err
				}
				params["fullpath"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.CreateFolder(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 上传文件(分块上传) ===

func newEntFileCreateFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-file",
		Short: "上传文件（支持分块上传）",
		Long: `上传本地文件到够快云库。

当指定 --file 时，自动计算 SHA1 哈希和文件大小，完整执行分块上传流程：
  create_file → upload_init → upload_part(s) → upload_finish

当省略 --file 时，退化为仅调用 create_file API（需手动提供 --filehash 和 --filesize）。`,
		Example: `  # 上传本地文件（推荐，自动分块）
  ykc ent file create-file --mount-id 1 --file ./report.pdf --fullpath /文档/report.pdf

  # 指定分块大小（默认 4MB）
  ykc ent file create-file --mount-id 1 --file ./large.zip --fullpath /归档/large.zip --chunk-size 8388608

  # 仅调用 API（不上传内容，需自行提供哈希）
  ykc ent file create-file --mount-id 1 --fullpath /文档/文件.txt --filehash abc123 --filesize 1024`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}

			localFile, _ := cmd.Flags().GetString("file")
			fullpath, _ := cmd.Flags().GetString("fullpath")
			fullpath = normalizeMSYSPath(fullpath)
			opID, _ := cmd.Flags().GetString("op-id")
			opName, _ := cmd.Flags().GetString("op-name")
			overwrite, _ := cmd.Flags().GetInt("overwrite")
			chunkSizeBytes, _ := cmd.Flags().GetInt64("chunk-size")
			mountID, _ := cmd.Flags().GetInt("mount-id")

			// ── Full upload mode: --file provided ─────────────────────────
			if localFile != "" {
				// Resolve absolute path for clarity in error messages
				absPath, err := filepath.Abs(localFile)
				if err != nil {
					return fmt.Errorf("解析文件路径失败: %w", err)
				}
				if _, err := os.Stat(absPath); err != nil {
					return apperrors.NewValidation(fmt.Sprintf("本地文件不存在: %s", absPath))
				}

				// Derive destination fullpath from filename if not provided
				if fullpath == "" {
					fullpath = "/" + filepath.Base(absPath)
				}
				if err := ValidateFullpath(fullpath); err != nil {
					return err
				}

				extraParams := map[string]string{
					"mount_id": fmt.Sprintf("%d", mountID),
				}
				if opID != "" {
					extraParams["op_id"] = opID
				}
				if opName != "" {
					extraParams["op_name"] = opName
				}
				if overwrite == 1 {
					extraParams["overwrite"] = "1"
				} else {
					extraParams["overwrite"] = "0"
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "正在计算文件哈希: %s\n", absPath)

				resp, err := client.UploadFile(absPath, fullpath, chunkSizeBytes, extraParams)
				if err != nil {
					return fmt.Errorf("上传失败: %w", err)
				}

				stateDesc := "已上传"
				if int(resp.State) == 1 {
					stateDesc = "秒传（哈希命中）"
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "上传完成 (%s): %s\n", stateDesc, resp.Fullpath)

				return outputJSON(cmd, resp)
			}

			// ── API-only mode: manual hash/size ───────────────────────────
			if fullpath == "" {
				return apperrors.NewValidation("请提供 --file（本地文件路径）或同时提供 --fullpath / --filehash / --filesize")
			}
			if err := ValidateFullpath(fullpath); err != nil {
				return err
			}

			filehash, _ := cmd.Flags().GetString("filehash")
			filesizeStr, _ := cmd.Flags().GetString("filesize")

			if filehash == "" || filesizeStr == "" {
				return apperrors.NewValidation("API 模式下 --filehash 和 --filesize 为必需参数（或改用 --file 自动计算）")
			}
			if _, err := strconv.ParseInt(filesizeStr, 10, 64); err != nil {
				return apperrors.NewValidation(fmt.Sprintf("--filesize 必须为整数: %v", err))
			}

			params := map[string]string{
				"fullpath": fullpath,
				"filehash": filehash,
				"filesize": filesizeStr,
				"mount_id": fmt.Sprintf("%d", mountID),
			}
			if opID != "" {
				params["op_id"] = opID
			}
			if opName != "" {
				params["op_name"] = opName
			}
			if overwrite == 1 {
				params["overwrite"] = "1"
			} else {
				params["overwrite"] = "0"
			}

			resp, err := client.CreateFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}

	// Destination path in the library
	cmd.Flags().String("fullpath", "", "库中的目标文件路径（上传模式下默认取本地文件名）")
	// Full upload flags
	cmd.Flags().String("file", "", "本地文件路径（指定后自动计算哈希并执行分块上传）")
	cmd.Flags().Int64("chunk-size", gokuai.DefaultChunkSize, "分块大小（字节），默认 4MB")
	// API-only flags (only meaningful when --file is omitted)
	cmd.Flags().String("filehash", "", "文件 SHA1 哈希（--file 省略时必需）")
	cmd.Flags().String("filesize", "", "文件字节大小（--file 省略时必需）")
	// Common flags
	cmd.Flags().String("op-id", "", "操作者 ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.Flags().Int("overwrite", 0, "是否覆盖同名文件: 1覆盖, 0不覆盖")

	return cmd
}

// === 获取上传服务器 ===

func newEntFileUploadServersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "upload-servers",
		Short:   "获取上传服务器",
		Example: `  ykc ent file upload-servers --mount-id 1 --rand abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			timeout, _ := cmd.Flags().GetInt("timeout")
			if timeout <= 1 {
				return fmt.Errorf("--timeout 必须为大于1的数")
			}
			params["timeout"] = strconv.Itoa(timeout)
			if v, _ := cmd.Flags().GetString("rand"); v != "" {
				params["rand"] = v
			}
			resp, err := client.GetUploadServers(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("timeout", 300, "超时时间(秒)，必须大于1")
	cmd.Flags().String("rand", "", "随机数 (必需)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("rand")
	return cmd
}

// === 复制文件(夹) ===

func newEntFileCopyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy",
		Short:   "复制文件(夹)",
		Example: `  ykc ent file copy --mount-id 1 --from-fullpath /文档/文件.txt --fullpath /备份/文件.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("from-fullpath"); v != "" {
				v = normalizeMSYSPath(v)
				if err := ValidateFullpath(v); err != nil {
					return err
				}
				params["from_fullpath"] = v
			}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				v = normalizeMSYSPath(v)
				if err := ValidateFullpath(v); err != nil {
					return err
				}
				params["fullpath"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			if v, _ := cmd.Flags().GetInt("overwrite"); v == 1 {
				params["overwrite"] = "1"
			} else {
				params["overwrite"] = "0"
			}
			resp, err := client.CopyFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("from-fullpath", "", "源文件路径 (必需)")
	cmd.Flags().String("fullpath", "", "目标文件路径 (必需)")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.Flags().Int("overwrite", 0, "是否覆盖同名文件: 1覆盖, 0不覆盖")
	cmd.MarkFlagRequired("from-fullpath")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 高级复制文件(夹) ===

func newEntFileMcopyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcopy",
		Short: "高级复制文件(夹)",
		Example: `  ykc ent file mcopy --mount-id 1 --from-fullpaths "/a.txt|/b.txt" --paths "/目标文件夹1|/目标文件夹2"
  ykc ent file mcopy --mount-id 1 --from-fullpaths "/a.txt" --paths "/备份" --copy-all 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("from-fullpaths"); v != "" {
				v = normalizeMSYSPaths(v)
				if err := ValidateFullpaths(v); err != nil {
					return err
				}
				params["from_fullpaths"] = v
			}
			if v, _ := cmd.Flags().GetString("paths"); v != "" {
				v = normalizeMSYSPaths(v)
				if err := ValidateFullpaths(v); err != nil {
					return err
				}
				params["paths"] = v
			}
			if v, _ := cmd.Flags().GetInt("copy-all"); v != 0 {
				params["copy_all"] = strconv.Itoa(v)
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.MultiCopyFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("from-fullpaths", "", "源文件路径, 多个用竖号分隔 (必需)")
	cmd.Flags().String("paths", "", "目标文件夹路径(不包含文件名), 多个用竖号分隔 (必需)")
	cmd.Flags().Int("copy-all", 0, "1复制文件的所有属性(包括操作人), 默认0不复制")
	cmd.Flags().String("op-id", "", "操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op-name代替")
	cmd.Flags().String("op-name", "", "操作人名称, 如果指定了op-id, 就不需要op-name")
	cmd.MarkFlagRequired("from-fullpaths")
	cmd.MarkFlagRequired("paths")
	return cmd
}

// === 移动文件(夹) ===

func newEntFileMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move",
		Short:   "移动文件(夹)",
		Example: `  ykc ent file move --mount-id 1 --fullpath /文档/文件.txt --dest-fullpath /备份/文件.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				v = normalizeMSYSPath(v)
				if err := ValidateFullpath(v); err != nil {
					return err
				}
				params["fullpath"] = v
			}
			if v, _ := cmd.Flags().GetString("dest-fullpath"); v != "" {
				v = normalizeMSYSPath(v)
				if err := ValidateFullpath(v); err != nil {
					return err
				}
				params["dest_fullpath"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.MoveFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "源文件路径 (必需)")
	cmd.Flags().String("dest-fullpath", "", "目标文件路径 (必需)")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("dest-fullpath")
	return cmd
}

// === 删除文件(夹) ===

func newEntFileDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del",
		Short: "删除文件(夹)",
		Example: `  ykc ent file del --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file del --mount-id 1 --tag abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("tag"); v != "" {
				params["tag"] = v
			}
			if v, _ := cmd.Flags().GetString("path"); v != "" {
				params["path"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			if v, _ := cmd.Flags().GetBool("destroy"); v {
				params["destroy"] = "1"
			} else {
				params["destroy"] = "0"
			}
			resp, err := client.DeleteFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径, fullpath 和 tag 只需传其中一个")
	cmd.Flags().String("tag", "", "通过标签删除, 多个用分号分隔, fullpath 和 tag 只需传其中一个")
	cmd.Flags().String("path", "", "当使用 tag 方式删除时, 可以指定路径进行删除, 默认空不指定")
	cmd.Flags().String("op-id", "", "操作人ID, 如果操作人不是云库用户, 可以用 op-name 代替")
	cmd.Flags().String("op-name", "", "操作人名称, 如果指定了 op-id, 就不需要 op-name")
	cmd.Flags().Bool("destroy", false, "彻底删除文件不进回收站 (不添加此参数则删除进入回收站)")
	return cmd
}

// === 彻底删除文件(夹) ===

func newEntFileDelCompletelyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del-completely",
		Short:   "彻底删除文件(夹)",
		Example: `  ykc ent file del-completely --mount-id 1 --fullpaths "/a.txt|/b.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpaths"); v != "" {
				params["fullpaths"] = normalizeMSYSPaths(v)
			}
			if v, _ := cmd.Flags().GetString("tag"); v != "" {
				params["tag"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.DeleteCompletely(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpaths", "", "文件路径，多个用竖号分隔")
	cmd.Flags().String("tag", "", "标签")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	return cmd
}

// === 回收站 ===

func newEntFileRecycleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recycle",
		Short: "回收站",
		Example: `  ykc ent file recycle --mount-id 1
  ykc ent file recycle --mount-id 1 --start 0 --size 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			start, _ := cmd.Flags().GetInt("start")
			if start < 0 {
				return fmt.Errorf("--start 必须是0或正整数, 当前值: %d", start)
			}
			size, _ := cmd.Flags().GetInt("size")
			if size <= 0 {
				return fmt.Errorf("--size 必须是大于0的整数, 当前值: %d", size)
			}
			params := map[string]string{}
			if start > 0 {
				params["start"] = fmt.Sprintf("%d", start)
			}
			params["size"] = fmt.Sprintf("%d", size)
			if v, _ := cmd.Flags().GetString("order"); v != "" {
				params["order"] = v
			}
			if v, _ := cmd.Flags().GetString("keyword"); v != "" {
				params["keyword"] = v
			}
			resp, err := client.GetRecycleList(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 100, "返回条数")
	cmd.Flags().String("order", "", "排序")
	cmd.Flags().String("keyword", "", "关键词")
	return cmd
}

// === 恢复已删除文件 ===

func newEntFileRecoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "recover",
		Short:   "恢复已删除文件",
		Example: `  ykc ent file recover --mount-id 1 --fullpaths "/a.txt|/b.txt"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpaths"); v != "" {
				params["fullpaths"] = normalizeMSYSPaths(v)
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.RecoverFiles(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpaths", "", "文件路径，多个用竖号分隔 (必需)")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.MarkFlagRequired("fullpaths")
	return cmd
}

// === 获取文件历史 ===

func newEntFileHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "获取文件历史",
		Example: `  ykc ent file history --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file history --mount-id 1 --fullpath /文档/文件.txt --start 0 --size 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			start, _ := cmd.Flags().GetInt("start")
			if start < 0 {
				return fmt.Errorf("--start 必须是0或正整数")
			}
			size, _ := cmd.Flags().GetInt("size")
			if size <= 0 {
				return fmt.Errorf("--size 必须是大于0的整数")
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			params["start"] = fmt.Sprintf("%d", start)
			params["size"] = fmt.Sprintf("%d", size)
			resp, err := client.GetFileHistory(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().Int("start", 0, "开始位置")
	cmd.Flags().Int("size", 20, "返回条数")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 获取文件外链 ===

func newEntFileLinkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "获取文件外链",
		Example: `  ykc ent file link --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file link --mount-id 1 --fullpath /文档/文件.txt --deadline 1700100000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkIntFlagInRange(cmd, "startline", 1, math.MaxInt64); err != nil {
				return err
			}
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetInt("deadline"); v != 0 {
				params["deadline"] = strconv.Itoa(v)
			}
			if v, _ := cmd.Flags().GetString("auth"); v != "" {
				params["auth"] = v
			}
			if v, _ := cmd.Flags().GetString("password"); v != "" {
				params["password"] = v
			}
			if v, _ := cmd.Flags().GetInt("startline"); v != 0 {
				params["startline"] = strconv.Itoa(v)
			}
			if v, _ := cmd.Flags().GetBool("keep"); v {
				params["keep"] = "1"
			}
			if v, _ := cmd.Flags().GetInt("access-limit"); v != 0 {
				params["access_limit"] = strconv.Itoa(v)
			}
			if v, _ := cmd.Flags().GetInt("op-id"); v != 0 {
				params["op_id"] = strconv.Itoa(v)
			}
			if v, _ := cmd.Flags().GetString("wm-content"); v != "" {
				params["wm_content"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.CreateFileLink(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.Flags().Int("deadline", 0, "到期时间戳, 默认48小时后")
	cmd.Flags().String("auth", "preview", "权限: preview(仅预览)/download(预览+下载)/upload(预览+下载+文件夹上传), 默认preview")
	cmd.Flags().String("password", "", "访问密码")
	cmd.Flags().Int("startline", 0, "生效时间戳, 不传则立即生效")
	cmd.Flags().Bool("keep", false, "不随文件修改更新外链 (不添加此参数则随文件更新)")
	cmd.Flags().Int("access-limit", 0, "访问次数限制，不传则不限制")
	cmd.Flags().Int("op-id", 0, "操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op-name代替")
	cmd.Flags().String("wm-content", "", "自定义水印内容")
	cmd.Flags().String("op-name", "", "操作人名称, 如果指定了op-id, 就不需要op-name")
	cmd.MarkFlagRequired("fullpath")
	cmd.InitDefaultHelpFlag()
	return cmd
}

// === 关闭文件外链 ===

func newEntFileLinkCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "link-close",
		Short:   "关闭文件外链",
		Example: `  ykc ent file link-close --mount-id 1 --fullpath /文档/文件.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("code"); v != "" {
				params["code"] = v
			}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.CloseFileLink(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("code", "", "外链code")
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	return cmd
}

// === 获取开启外链的文件列表 ===

func newEntFileLinksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "links",
		Short: "获取开启外链的文件列表",
		Example: `  ykc ent file links --mount-id 1
  ykc ent file links --mount-id 1 --file 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetInt("file"); v != 0 {
				params["file"] = strconv.Itoa(v)
			}
			resp, err := client.GetFileLinks(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().Int("file", 0, "1仅返回文件, 0返回全部, 默认0")
	return cmd
}

// === 文件锁的操作 ===

func newEntFileLockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "文件锁的操作",
		Example: `  ykc ent file lock --mount-id 1 --fullpath /文档/文件.txt --lock lock
  ykc ent file lock --mount-id 1 --fullpath /文档/文件.txt --lock unlock`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			lockVal, _ := cmd.Flags().GetString("lock")
			if lockVal != "lock" && lockVal != "unlock" {
				return fmt.Errorf("--lock 参数值必须是 lock(上锁) 或 unlock(解锁), 当前值: %q", lockVal)
			}
			params["lock"] = lockVal
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.LockFile(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("lock", "", "锁操作: lock 或 unlock (必需)")
	cmd.Flags().String("op-id", "", "操作者ID")
	cmd.Flags().String("op-name", "", "操作者名称")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("lock")
	return cmd
}

// === 设置文件夹权限继承状态 ===

func newEntFileSetPermissionInheritCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-permission-inherit",
		Short: "设置文件夹权限继承状态",
		Example: `  ykc ent file set-permission-inherit --mount-id 1 --fullpath /文档 --inherit
  ykc ent file set-permission-inherit --mount-id 1 --fullpath /文档`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetBool("inherit"); v {
				params["inherit"] = "1"
			} else {
				params["inherit"] = "0"
			}
			resp, err := client.SetPermissionInherit(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹路径 (必需)")
	cmd.Flags().Bool("inherit", false, "继承权限 (不添加此参数则为不继承)")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 获取文件夹单独设置的权限 ===

func newEntFileGetAllPermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-all-permission",
		Short:   "获取文件夹单独设置的权限",
		Example: `  ykc ent file get-all-permission --mount-id 1 --fullpath /文档`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			resp, err := client.GetAllPermission(params)
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

// === 修改文件夹权限 ===

func newEntFileFilePermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file-permission",
		Short: "修改文件夹权限",
		Example: `  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"user123":["file_preview","file_read"]}'
  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"group456":["file_preview","file_read"]}' --is-group
  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"out_user789":["file_preview"]}' --is-out`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("permissions"); v != "" {
				params["permissions"] = v
			}
			if v, _ := cmd.Flags().GetBool("is-group"); v {
				params["is_group"] = "1"
			} else {
				params["is_group"] = "0"
			}
			if v, _ := cmd.Flags().GetBool("is-out"); v {
				params["is_out"] = "1"
			} else {
				params["is_out"] = "0"
			}
			resp, err := client.SetFilePermission(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹完整路径 (必需)")
	cmd.Flags().String("permissions", "", `权限设置, 如 '{"用户ID":["file_preview","file_read"]}' (必需)`)
	cmd.Flags().Bool("is-group", false, "设置部门权限 (不添加此参数则设置用户权限)")
	cmd.Flags().Bool("is-out", false, "是否外部成员 (不添加此参数则为否)")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("permissions")
	return cmd
}

// === 重置或移除文件夹权限 ===

func newEntFileResetPermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-permission",
		Short: "重置或移除文件夹权限",
		Example: `  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --members 1,2
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --members 1,2 --clear
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --groups 1
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --groups 1 --clear`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("members"); v != "" {
				params["members"] = v
			}
			if v, _ := cmd.Flags().GetString("groups"); v != "" {
				params["groups"] = v
			}
			if v, _ := cmd.Flags().GetBool("clear"); v {
				params["clear"] = "1"
			}
			resp, err := client.ResetPermission(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件夹完整路径 (必需)")
	cmd.Flags().String("members", "", "要重置或移除的成员ID, 多个用逗号分隔")
	cmd.Flags().String("groups", "", "要重置或移除的部门ID, 多个用逗号分隔")
	cmd.Flags().Bool("clear", false, "清除通过部门加入的成员权限, 仅在文件夹非继承状态时生效, 默认不清除")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 获取文件权限 ===

func newEntFileGetPermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-permission",
		Short:   "获取文件权限",
		Example: `  ykc ent file get-permission --mount-id 1 --fullpath /文档 --member-id 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("member-id"); v != "" {
				params["member_id"] = v
			}
			if v, _ := cmd.Flags().GetString("out-id"); v != "" {
				params["out_id"] = v
			}
			if v, _ := cmd.Flags().GetString("account"); v != "" {
				params["account"] = v
			}
			resp, err := client.GetFilePermission(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径 (必需)")
	cmd.Flags().String("member-id", "", "成员ID")
	cmd.Flags().String("out-id", "", "外部ID")
	cmd.Flags().String("account", "", "帐号")
	cmd.MarkFlagRequired("fullpath")
	return cmd
}

// === 添加标签 ===

func newEntFileAddTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add-tag",
		Short:   "添加标签",
		Example: `  ykc ent file add-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("tag"); v != "" {
				params["tag"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.AddTag(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.Flags().String("tag", "", "标签, 多个使用分号;分隔 (必需)")
	cmd.Flags().String("op-id", "", "操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op_name代替")
	cmd.Flags().String("op-name", "", "操作人名称, 如果指定了op_id, 就不需要op_name")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("tag")
	return cmd
}

// === 删除标签 ===

func newEntFileDelTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del-tag",
		Short:   "删除标签",
		Example: `  ykc ent file del-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("tag"); v != "" {
				params["tag"] = v
			}
			if v, _ := cmd.Flags().GetString("op-id"); v != "" {
				params["op_id"] = v
			}
			if v, _ := cmd.Flags().GetString("op-name"); v != "" {
				params["op_name"] = v
			}
			resp, err := client.DelTag(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件完整路径 (必需)")
	cmd.Flags().String("tag", "", "标签, 多个使用分号;分隔 (必需)")
	cmd.Flags().String("op-id", "", "操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op_name代替")
	cmd.Flags().String("op-name", "", "操作人名称, 如果指定了op_id, 就不需要op_name")
	cmd.MarkFlagRequired("fullpath")
	cmd.MarkFlagRequired("tag")
	return cmd
}

// === 添加或修改元数据 ===

func newEntFileSetMetadataCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set-metadata",
		Short:   "添加或修改元数据",
		Example: `  ykc ent file set-metadata --mount-id 1 --fullpath /文档/文件.txt --key key1 --metadata value1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetString("key"); v != "" {
				params["key"] = v
			}
			if v, _ := cmd.Flags().GetString("metadata"); v != "" {
				params["metadata"] = v
			}
			if v, _ := cmd.Flags().GetString("display"); v != "" {
				params["display"] = v
			}
			resp, err := client.SetMetadata(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件hash")
	cmd.Flags().String("key", "", "元数据key (必需)")
	cmd.Flags().String("metadata", "", "元数据内容 (必需)")
	cmd.Flags().String("display", "", "显示名称")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("metadata")
	return cmd
}

// === 删除元数据 ===

func newEntFileDelMetadataCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del-metadata",
		Short:   "删除元数据",
		Example: `  ykc ent file del-metadata --mount-id 1 --fullpath /文档/文件.txt --key key1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpath"); v != "" {
				params["fullpath"] = normalizeMSYSPath(v)
			}
			if v, _ := cmd.Flags().GetString("hash"); v != "" {
				params["hash"] = v
			}
			if v, _ := cmd.Flags().GetString("key"); v != "" {
				params["key"] = v
			}
			resp, err := client.DelMetadata(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpath", "", "文件路径")
	cmd.Flags().String("hash", "", "文件hash")
	cmd.Flags().String("key", "", "元数据key (必需)")
	cmd.MarkFlagRequired("key")
	return cmd
}

// === 统计信息 ===

func newEntFileStatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stat",
		Short:   "统计信息",
		Example: `  ykc ent file stat --mount-id 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			resp, err := client.GetStat(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	return cmd
}

// === 查询队列状态 ===

func newEntFileQueueStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "queue-status",
		Short:   "查询队列状态",
		Example: `  ykc ent file queue-status --mount-id 1 --queue-id abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("queue-id"); v != "" {
				params["queue_id"] = v
			}
			resp, err := client.GetQueueStatus(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("queue-id", "", "队列ID (必需)")
	cmd.MarkFlagRequired("queue-id")
	return cmd
}

// === 批量设置权限 ===

func newEntFileBatchSetPermissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "batch-set-permission",
		Short:   "批量设置权限",
		Example: `  ykc ent file batch-set-permission --mount-id 1 --fullpaths "/a|/b" --type 1 --permissions "..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getOrgClient(cmd)
			if err != nil {
				return err
			}
			params := map[string]string{}
			if v, _ := cmd.Flags().GetString("fullpaths"); v != "" {
				params["fullpaths"] = normalizeMSYSPaths(v)
			}
			if v, _ := cmd.Flags().GetString("type"); v != "" {
				params["type"] = v
			}
			if v, _ := cmd.Flags().GetString("inherit"); v != "" {
				params["inherit"] = v
			}
			if v, _ := cmd.Flags().GetString("permissions"); v != "" {
				params["permissions"] = v
			}
			resp, err := client.BatchSetPermission(params)
			if err != nil {
				return err
			}
			return outputJSON(cmd, resp)
		},
	}
	cmd.Flags().String("fullpaths", "", "文件路径，多个用竖号分隔 (必需)")
	cmd.Flags().String("type", "", "类型 (必需)")
	cmd.Flags().String("inherit", "", "继承状态")
	cmd.Flags().String("permissions", "", "权限设置 (必需)")
	cmd.MarkFlagRequired("fullpaths")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("permissions")
	return cmd
}

// downloadFile downloads a file from the given URL and saves it to outputPath.
// If the file already exists, it will be renamed with a number suffix, e.g., 1.docx -> 1(1).docx
func downloadFile(cmd *cobra.Command, url, outputPath string) error {
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("解析输出路径失败: %w", err)
	}

	// Check if file exists and generate a new name if needed
	absPath = generateUniquePath(absPath)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "开始下载: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return apperrors.NewAPI(fmt.Sprintf("下载请求失败: %v", err), apperrors.WithRetryable(true), apperrors.WithCause(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperrors.NewAPI(fmt.Sprintf("下载失败，HTTP 状态码: %d", resp.StatusCode))
	}

	totalSize := resp.ContentLength
	if totalSize > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "文件大小: %s\n", formatSize(totalSize))
	}

	f, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	var written int64
	buf := make([]byte, 32*1024)
	lastTime := time.Now()
	var lastWritten int64

	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := f.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return fmt.Errorf("写入文件失败: %w", ew)
			}
			if nr != nw {
				return fmt.Errorf("写入文件失败: 短写")
			}
		}
		if er != nil {
			if er.Error() != "EOF" && !isEOF(er) {
				return apperrors.NewAPI(fmt.Sprintf("下载中断: %v", er), apperrors.WithRetryable(true), apperrors.WithCause(er))
			}
			break
		}

		// Print progress every 500ms
		now := time.Now()
		if now.Sub(lastTime) >= 500*time.Millisecond {
			speed := float64(written-lastWritten) / now.Sub(lastTime).Seconds()
			if totalSize > 0 {
				pct := float64(written) / float64(totalSize) * 100
				fmt.Fprintf(cmd.ErrOrStderr(), "下载进度: %s / %s (%.1f%%) 速度: %s/s\n",
					formatSize(written), formatSize(totalSize), pct, formatSize(int64(speed)))
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "已下载: %s  速度: %s/s\n",
					formatSize(written), formatSize(int64(speed)))
			}
			lastTime = now
			lastWritten = written
		}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "下载完成: %s (%s)\n", absPath, formatSize(written))
	return nil
}

func isEOF(err error) bool {
	return err == io.EOF
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return strconv.FormatInt(bytes, 10) + " B"
	}
}

// generateUniquePath returns a unique file path by adding a number suffix if the file exists.
// e.g., /path/1.docx -> /path/1(1).docx -> /path/1(2).docx
func generateUniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	for i := 1; ; i++ {
		newPath := filepath.Join(dir, fmt.Sprintf("%s(%d)%s", base, i, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
}

// validSearchScopeValues enumerates the allowed scope tokens for file search.
var validSearchScopeValues = map[string]bool{
	"filename": true,
	"tag":      true,
	"content":  true,
}

// validateSearchScope checks that --scope is a JSON array of strings drawn
// exclusively from validSearchScopeValues, e.g. '["filename","content"]'.
func validateSearchScope(v string) error {
	var scopes []string
	if err := json.Unmarshal([]byte(v), &scopes); err != nil {
		return apperrors.NewValidation(fmt.Sprintf("--scope 必须是 JSON 数组字符串 (如 '[\"filename\",\"content\"]'): %v", err))
	}
	if len(scopes) == 0 {
		return apperrors.NewValidation("--scope 不能为空数组")
	}
	for _, s := range scopes {
		if !validSearchScopeValues[s] {
			return apperrors.NewValidation(fmt.Sprintf("--scope 包含无效值 %q，允许值为: filename, tag, content", s))
		}
	}
	return nil
}
