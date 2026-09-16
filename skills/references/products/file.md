# file — 用户 API 文件操作命令参考

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc file` | 用户 access_token | `ykc auth login` 登录后自动管理 |

> 需要先 `ykc auth login` 登录，所有命令需要 `--mount-id`（库空间ID）

---

#### 文件列表
```
Usage:
  ykc file ls [flags]
Example:
  ykc file ls --mount-id 1 --fullpath "/项目文档"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件夹路径，空字符串表示根目录
  --dir int           1只返回文件夹, 0都返回
  --order string      排序方式: filename asc|desc, last_dateline asc|desc, filesize asc|desc
  --start int         开始位置
  --size int          返回条数 (默认100)
```

#### 文件最近更新列表
```
Usage:
  ykc file updates [flags]
Example:
  ykc file updates --mount-id 1 --dateline 0
Flags:
  --mount-id int   库空间ID (必需)
  --dateline int   时间戳，获取此时间之后的更新
  --start int      开始位置
  --size int       返回条数 (默认100)
Usage:
  ykc file updates [flags]

Examples:
  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508
  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508 --size 50
  ykc file updates --mount-id 1 --from-datelinems 0 --to-datelinems 1781167091508 --act 1

Flags:
      --mount-id int   库空间ID (必需)
      --act int               操作类型
      --act-member-id int     操作人ID
      --from-datelinems int   开始毫秒 (必需)
  -h, --help                  help for updates
      --size int              数据长度 (default 20)
      --to-datelinems int     结束毫秒 (必需)
```

#### 文件信息
```
Usage:
  ykc file info [flags]

Examples:
  ykc file info --mount-id 1 --fullpath /documents/readme.txt
  ykc file info --mount-id 1 --hash abc123
  ykc file info --mount-id 1 --fullpath /documents/readme.txt --net in --open
  ykc file info --mount-id 1 --fullpath /doc.txt --ignores tag,favorite,property

Flags:
      --mount-id int          库空间ID (必需)
      --fullpath string       文件的路径
      --hash string           路径hash
      --net string            返回下载地址公网/内网(in-内网，不传默认公网)
      --open                  是否打开该文件(默认false)
      --hid string            版本ID
      --nop                 不包含库权限, 只返回文件夹上设置的权限 (不添加此参数则包含库权限)
      --stat                记录下载日志 (不添加此参数则不记录)
      --ignores string        忽略哪些返回信息(,隔开) (tag,favorite,property,permission,lock,url,preview,thumbnail)
      --wf-id int             流程ID(流程审批提交前)
      --wf-approve-id int     流程审批ID(流程审批提交后)
  -h, --help                  help for info
```

#### 获取文件属性
```
Usage:
  ykc file attr [flags]
Example:
  ykc file attr --mount-id 1 --fullpath "文档/readme.pdf"
Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for attr       help for attr
```

#### 搜索文件
```
Usage:
  ykc file search [flags]
Example:
  ykc file search --mount-id 1 --keyword "报告"
Flags:
  --mount-id int     库空间ID (必需)
  --keyword string   搜索关键字 (必需)
  --start int        开始位置
  --size int         返回条数 (默认20)
```

#### 文件预览链接
```
Usage:
  ykc file preview-url [flags]

Examples:
  ykc file preview-url --mount-id 1 --hash abc123
  ykc file preview-url --mount-id 1 --hash abc123 --watermark
  ykc file preview-url --mount-id 1 --hash abc123 --watermark --member-name "张三"
  ykc file preview-url --mount-id 1 --hash abc123 --filename report.pdf
  ykc file preview-url --mount-id 1 --hash abc123 --dialog_id private:123&234&56 --message_id ABC

Flags:
      --mount-id int       库空间ID (必需)
      --hash string        文件唯一标识
      --filename string    强制指定文件名
      --watermark          是否显示水印(默认不显示)
      --member-name string 显示水印时,指定水印上的预览者身份
      --dialog-id string   会话ID
      --message-id string  消息ID
```

#### 获取文件打开URL
```
Usage:
  ykc file open-url [flags]
Example:
  ykc file open-url --mount-id 1 --fullpath "文档/a.docx"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件完整路径 (必需)
```

#### 获取协作编辑URL
```
Usage:
  ykc file cedit-url [flags]
Example:
  ykc file cedit-url --mount-id 1 --fullpath "test.txt"
  ykc file cedit-url --mount-id 1 --fullpath "test.txt" --edit
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件完整路径 (必需)
  --edit              编辑模式 (不添加此参数则为预览)
```

#### 创建文件夹
```
Usage:
  ykc file mkdir [flags]
Example:
  ykc file mkdir --mount-id 1 --fullpath /documents/新建文件夹
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件夹路径 (必需)
```

#### 复制文件(夹)
```
Usage:
  ykc file copy [flags]
Example:
  ykc file copy --mount-id 1 --fullpath "documents/a.txt" --target-mount-id 1 --target-fullpath "documents/b.txt"
  ykc file copy --mount-id 1 --fullpaths "a.txt|b.txt" --target-mount-id 2 --target-fullpath "backup/"
Flags:
      --mount-id int   库空间ID (必需)
      --check-from string        是否检测源权限
      --fullpath string          要复制文件的路径(单文件复制)
      --fullpaths string         要复制文件的路径(多文件复制,|分隔)
  -h, --help                     help for copy
      --overwrite string         是否覆盖同名文件,0不覆盖,1覆盖,默认0
      --target-fullpath string   复制到的路径 (必需)
      --target-mount-id string   复制到的mount_id (必需)
```

#### 移动文件
```
Usage:
  ykc file move [flags]
Example:
  ykc file move --mount-id 1 --fullpath "a.txt" --target-mount-id 2 --target-fullpath "folder/"
  ykc file move --mount-id 1 --fullpaths "a.txt|b.txt" --target-mount-id 2 --target-fullpath "folder/"
Flags:
      --mount-id int   库空间ID (必需)
      --fullpath string          文件路径(单文件)
      --fullpaths string         多文件路径(分隔)
  -h, --help                     help for move
      --target-fullpath string   移动到的路径 (必需)
      --target-mount-id string   移动到的mount_id (必需)
```

#### 重命名文件
```
Usage:
  ykc file rename [flags]
Example:
  ykc file rename --mount-id 1 --fullpath "a.txt" --newname "b.txt"
Flags:
  --mount-id int    库空间ID (必需)
  --fullpath string   文件路径 (必需)
  --newname string    新的名称 (必需)
```


#### 删除文件(夹)
```
Usage:
  ykc file delete [flags]
Example:
  ykc file delete --mount-id 1 --fullpath documents/a.txt
  ykc file delete --mount-id 1 --fullpath documents/folder
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件路径 (必需)
```

#### 清空回收站
```
Usage:
  ykc file clear [flags]
Example:
  ykc file clear --mount-id 1
Flags:
  --mount-id int   库空间ID (必需)
```

#### 彻底删除文件
```
Usage:
  ykc file delete-completely [flags]
Example:
  ykc file delete-completely --mount-id 1 --fullpath documents/a.txt
  ykc file delete-completely --mount-id 1 --fullpaths "a.txt|b.txt"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   单个文件路径
  --fullpaths string  多个文件路径, |分隔
```

#### 回收站列表
```
Usage:
  ykc file recycle [flags]
Example:
  ykc file recycle --mount-id 1
Flags:
  --mount-id int   库空间ID (必需)
  --start int      开始位置
  --size int       返回条数 (默认100)
```

#### 恢复已删除文件
```
Usage:
  ykc file recover [flags]
Example:
  ykc file recover --mount-id 1 --fullpaths "documents/a.txt|documents/b.txt"
  ykc file recover --mount-id 1 --fullpaths "documents/a.txt" --machine "my-pc"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpaths string   文件路径
  --machine string     操作机器
```

#### 获取文件历史
```
Usage:
  ykc file history [flags]
Example:
  ykc file history --mount-id 1 --fullpath documents/a.txt
  ykc file history --mount-id 1 --hash abc123 --start 0 --size 50
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件路径
  --hash string       文件唯一标识
  --start int         开始位置
  --size int          返回条数 (默认50)
```

#### 文件外链操作
```
# 创建文件外链
Usage:
  ykc file link create [flags]

Examples:
  ykc file link create --mount-id 1 --fullpath documents/a.txt --dir 0 --deadline 10d
  ykc file link create --mount-id 1 --fullpath documents/a.txt --dir 0 --deadline 2d --password 123456 --auth 110 --scope 0

Flags:
      --mount-id int      库空间ID (必需)
      --access-limit string      访问次数限制，默认不限制
      --annotation-view        开启批注显示
      --auth string              权限: 100预览, 110预览+下载, 111预览+下载+上传, 101预览+上传, 001仅上传
      --authems string           scope为3时身份验证的邮箱或手机号，分号分隔
      --content string           发送邮箱时告知的内容
      --deadline string          过期时间戳或字符串(如10d/1w/1m/1y), -1永不失效 (必需)
      --dir string               是否文件夹 (必需)
      --edit                     允许协同编辑
      --emails string            发送到邮箱地址列表，分号分隔
      --fullpath string          文件路径 (必需)
  -h, --help                     help for create
      --keep                     外链不随文件修改而更新 (不添加此参数则随文件更新)
      --password string          外链密码
      --scope string             访问范围: 0所有人, 1仅企业成员, 3需身份验证
      --startline string         生效时间戳，不传则立即生效

# 关闭文件外链
ykc file link close --mount-id 1 --fullpath "/文档/a.pdf"
  --fullpath string   文件路径 (必需)

# 获取文件外链列表
Usage:
  ykc file link list [flags]

Examples:
  ykc file link list --mount-id 1 --fullpath documents/
  ykc file link list --mount-id 1 --fullpath documents/ --state -1

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件路径 (必需)
  -h, --help              help for list
            --state string      外链状态: -1全部, 1开启, 0关闭(默认未关闭)--state string      外链状态: -1全部, 1开启, 0关闭(默认未关闭)

# 更新文件链接设置
Usage:
  ykc file link update [flags]

Examples:
  ykc file link update --mount-id 1 --code abc123 --deadline 1735660800

Flags:
      --mount-id int      库空间ID (必需)
      --access-limit string   访问次数限制
      --auth string           认证方式: 0无, 1密码, 2登录
      --code string           外链code (必需)
      --day string            有效天数
      --deadline string       过期时间戳
  -h, --help                  help for update
      --keep                  外链不随文件修改而更新 (不添加此参数则随文件更新)
      --password string       外链密码
      --scope string          权限: 1预览, 2下载, 3上传, 5编辑

# 获取文件外链详情
获取文件外链详情

Usage:
  ykc file link detail [flags]

Examples:
  ykc file link detail --mount-id 1 --code abc123

Flags:
      --mount-id int      库空间ID (必需)
      --code string   外链code (必需)
  -h, --help          help for detail
```

#### 文件锁操作
```
# 锁定文件
Usage:
  ykc file lock set [flags]

Examples:
  ykc file lock set --mount-id 1 --fullpath documents/a.txt

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件路径 (必需)
  -h, --help              help for set

# 解锁文件
Usage:
  ykc file lock unset [flags]

Examples:
  ykc file lock unset --mount-id 1 --fullpath documents/a.txt

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件路径 (必需)
  -h, --help              help for unset
```

#### 设置文件标签
```
# 获取文件夹成员权限
获取文件夹成员权限 (GET /m-api/1/file/get_member_permissions)。
指定 --format csv 时导出所有用户的权限，CSV 内容为 GBK 编码:
  姓名,邮箱,预览,下载,上传,编辑,删除,外链

Usage:
  ykc file permission member [flags]

Examples:
  ykc file permission member --mount-id 1 --fullpath documents
  ykc file permission member --mount-id 1 --fullpath documents --keyword 张三
  ykc file permission member --mount-id 1 --fullpath documents --format csv
  ykc file permission member --mount-id 1 --fullpath documents --on-group

Flags:
      --format string     为 csv 时导出所有用户的权限(CSV 内容为 GBK 编码)
      --fullpath string   文件路径 (必需)
  -h, --help              help for member
      --is-out            是否外部成员
      --keyword string    成员名关键字
      --on-group          显示继承部门权限的成员(设置文件夹权限不继承情况下)
      --size int          获取数量 (必需) (default 20)
      --start int         开始位置 (必需)

# 获取文件夹部门权限
Usage:
  ykc file permission group [flags]

Examples:
  ykc file permission group --mount-id 1 --fullpath documents

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件夹路径 (必需)
  -h, --help              help for group

# 设置文件夹权限
Usage:
  ykc file permission set [flags]

Examples:
  ykc file permission set --mount-id 1 --fullpath documents --permission '{"1420729":["file_export","admin","file_list"]}'

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string     文件夹路径 (必需)
  -h, --help                help for set
      --is-group            是否部门权限
      --is-out              是否外部成员
      --permission string   权限设置, JSON (必需)

# 设置文件夹权限是否继承

Usage:
  ykc file permission set-inherit [flags]

Examples:
  ykc file permission set-inherit --mount-id 1 --fullpath documents --inherit

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件夹路径 (必需)
  -h, --help              help for set-inherit
      --inherit           继承权限 (不添加此参数则为不继承)

# 重置文件夹权限
Usage:
  ykc file permission reset [flags]

Examples:
  ykc file permission reset --mount-id 1 --fullpath documents --permission '[1420729]'

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string     文件夹路径 (必需)
  -h, --help                help for reset
      --is-group            是否部门权限
      --is-out              是否外部成员
      --permission string   权限设置 (必需)

# 重置文件夹所有权限
Usage:
  ykc file permission reset-all [flags]

Examples:
  ykc file permission reset-all --mount-id 1 --fullpath documents --type member
  ykc file permission reset-all --mount-id 1 --fullpath documents --type group
  ykc file permission reset-all --mount-id 1 --fullpath documents --type out

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件夹路径 (必需)
  -h, --help              help for reset-all
      --type string       重置哪类权限: member成员, group部门, out外部成员 (必需)
```

#### 文件生命周期
```
# 添加文件生命周期
Usage:
  ykc file add-lifecycle [flags]

Examples:
  ykc file add-lifecycle --mount-id 1 --fullpath /documents/ --type daysago --rule 30
  ykc file add-lifecycle --mount-id 1 --fullpath /test/ --type expire --rule 2019-10-10
  ykc file add-lifecycle --mount-id 1 --fullpath /documents/ --type time --rule day-6 --retainfolder --destroy

Flags:
      --mount-id int      库空间ID (必需)
      --destroy               彻底删除 (不添加此参数则移入回收站)
      --fullpath string       文件夹路径(以 / 结尾) (必需)
  -h, --help                  help for add-lifecycle
      --retainfolder          保留文件夹结构
      --rule string           规则: daysago时为天数, expire时为日期, time时为频率(格式 day-N/week-N/month-N) (必需)
      --type string           类型: daysago自动清理, expire到期清理, time定时清理 (必需)

# 获取文件生命周期
Usage:
  ykc file get-lifecycle [flags]

Examples:
  ykc file get-lifecycle --mount-id 1 --fullpath "test.txt"

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for get-lifecycle

# 删除文件生命周期
Usage:
  ykc file del-lifecycle [flags]

Examples:
  ykc file del-lifecycle --mount-id 1 --fullpath "test.txt"

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for del-lifecycle
```

#### 文件提醒
```
# 设置文件提醒
Usage:
  ykc file set-remind [flags]

Examples:
  ykc file set-remind --mount-id 1 --fullpath "test.txt" --expire 1735660800 --reminder "1,2,3"
  ykc file set-remind --mount-id 1 --fullpath "test.txt" --expire 1735660800 --timer "day-9,11-30" --reminder "1,2,3"

Flags:
      --mount-id int      库空间ID (必需)
      --file-props stringSlice   文件属性字段, 可多个(逗号分隔): fullpath,last_dateline,last_member_id,filesize,create_dateline,create_member_id
      --expire string     到期时间戳(int类型) (必需)
      --fullpath string   文件路径 (必需)
  -h, --help              help for set-remind
      --machine string    操作机器
      --remark string     备注说明
      --reminder string   被提醒人(,隔开的member_id) (必需)
      --timer string      定时提醒, 格式: day-9,11-30;week-3,7-14,17-00;month-1,11,21-10,16-00

# 删除文件提醒
Usage:
  ykc file del-remind [flags]

Examples:
  ykc file del-remind --mount-id 1 --fullpath "test.txt"

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for del-remind
```

#### 文件评论
```
# 添加文件评论
Usage:
  ykc file add-comment [flags]

Examples:
  ykc file add-comment --mount-id 1 --fullpath "test.txt" --message "[@ id=all]@所有人[/@]开会"
  ykc file add-comment --mount-id 1 --fullpath "文档/a.pdf" --message "[@ id=1234567]@1122[/@]你好"

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for add-comment
      --message string    评论内容 (必需)，提及须写成 [@ id=<id>]@<名称>[/@]，例如 [@ id=all]@所有人[/@]；也支持输入 @成员 自动转换

# 获取文件评论
Usage:
  ykc file get-comment [flags]

Examples:
  ykc file get-comment --mount-id 1 --fullpath "test.txt"

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件完整路径 (必需)
  -h, --help              help for get-comment
```

#### 检测文件执行队列状态
```
Usage:
  ykc file queue [flags]
Example:
  ykc file queue --mount-id 1 --queue-id abc123 --type 1
Flags:
  --mount-id int      库空间ID (必需)
  --queue-id string   队列ID
  --type string       队列类型
```


#### 设置文件标签
```
Usage:
  ykc file keyword [flags]

Examples:
  ykc file keyword --mount-id 1 --fullpath documents/a.txt --keywords '秋天;很美'

Flags:
      --mount-id int      库空间ID (必需)
      --fullpath string   文件路径
      --hash string       文件唯一标识
  -h, --help              help for keyword
      --keywords string   标签, 分号分隔, 如 '秋天;很美' (必需)
```

#### 文件是否存在
```
Usage:
  ykc file exist [flags]
Example:
  ykc file exist --mount-id 1 --fullpath "文档/a.pdf"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件路径
  --hash string       文件唯一标识
```

#### 获取文件地址并下载
```
Usage:
  ykc file download [flags]

Examples:
  ykc file download --mount-id 1 --hash abc123
  ykc file download --mount-id 1 --hash abc123 --output ./file.pdf

Flags:
      --mount-id int      库空间ID (必需)
      --filehash string   文件内容哈希
      --hash string       文件唯一标识
  -h, --help              help for download
      --output string     下载文件保存路径（指定后直接下载文件）
```

#### 还原历史版本
```
Usage:
  ykc file revert [flags]
Example:
  ykc file revert --mount-id 1 --fullpath "文档/a.pdf" --hid 100
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件路径
  --hid string        历史版本ID (必需)
```

#### 图片搜索
```
Usage:
  ykc file image-search [flags]
Example:
  ykc file image-search --mount-id 1 --keyword "风景"
  ykc file image-search --mount-id 1 --file ./photo.jpg
  ykc file image-search --mount-id 1 --image-mount-id 1 --image-hash abc123
Flags:
  --mount-id int            库空间ID (必需)
  --keyword string          搜索关键字 (与 --file / --image-mount-id+--image-hash 至少传一个)
  --file string             本地图片文件路径 (仅支持 jpeg/jpg/png/webp/bmp/tif/tiff, 单张最大 20MB)
  --image-mount-id string   库中图片所在库ID (与 --image-hash 一起使用)
  --image-hash string       库中图片的 hash (与 --image-mount-id 一起使用)
  --ext string              图片扩展名
  --size int                返回条数 (默认20)
  --create-member-id string   文件创建者的 member_id
  --last-member-id string     文件修改者的 member_id
```

#### 获取文件到期提醒
```
Usage:
  ykc file get-remind [flags]
Example:
  ykc file get-remind --mount-id 1 --fullpath "文档/a.pdf"
Flags:
  --mount-id int      库空间ID (必需)
  --fullpath string   文件完整路径 (必需)
```

#### 获取文件到期提醒列表
```
Usage:
  ykc file get-reminds [flags]
Example:
  ykc file get-reminds
Flags:
  --is-expire string  是否过期: 0未过期, 1已过期
  --start int         开始位置
  --size int          返回条数 (默认20)
```

#### 上传本地文件到够快云库（用户 API）
```
当指定 --file 时，自动计算 SHA1 哈希和文件大小，完整执行分块上传流程：
  create_file → upload_init → upload_part(s) → upload_finish

create_file 调用签名时自动排除 filehash / filesize（协议要求）。
upload_init 通过 x-gk-token 请求头传递 access_token（无需 org_client_id）。

Usage:
  ykc file create-file [flags]

Examples:
  # 上传本地文件（推荐，自动分块）
  ykc file create-file --mount-id 1 --file ./report.pdf --fullpath /文档/report.pdf

  # 指定分块大小（默认 4MB）
  ykc file create-file --mount-id 1 --file ./large.zip --fullpath /归档/large.zip --chunk-size 8388608

  # 覆盖同名文件
  ykc file create-file --mount-id 1 --file ./data.csv --fullpath /data/data.csv --overwrite

Flags:
      --chunk-size int    分块大小（字节），默认 4MB (default 4194304)
      --file string       本地文件路径（必需）
      --fullpath string   库中的目标路径（默认取本地文件名）
  -h, --help              help for create-file
      --overwrite         覆盖已存在的同名文件
```


---

## 意图判断

### 用户文件操作
- 说"查看文件/列出文件" → `file ls`
- 说"搜索文件" → `file search`
- 说"搜索图片" → `file image-search`
- 说"预览文件" → `file preview-url`
- 说"创建文件夹" → `file mkdir`
- 说"复制/移动/删除文件" → `file copy` / `file move` / `file delete`
- 说"彻底删除文件" → `file delete-completely`
- 说"清空回收站" → `file clear`
- 说"文件是否存在" → `file exist`
- 说"获取文件地址" → `file download`
- 说"还原历史版本" → `file revert`
- 说"外链" → `file link create/close/list/update/detail`
- 说"锁定文件" → `file lock set/unset`
- 说"标签" → `file keyword`
- 说"权限" → `file permission member/group/set/set-inherit/reset/reset-all`
- 说"文件到期提醒" → `file set-remind/del-remind/get-remind/get-reminds`

## 上下文传递表

| 操作 | 提取字段 | 用于 |
|------|---------|------|
| `account mount` | `mount_id` | `file` 命令的 `--mount-id` |
