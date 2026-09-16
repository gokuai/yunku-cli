# ent — 企业管理命令参考

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc ent`（oauth/file 除外） | 企业 client_id + secret 签名 | `GOKUAI_CLIENT_ID` `GOKUAI_CLIENT_SECRET` |
| `ykc ent file` | 库授权 client_id + secret 签名 | 推荐传 `--org-id`/`--mount-id` 自动绑定（依赖企业凭证）；或直接配置 `GOKUAI_ORG_CLIENT_ID` `GOKUAI_ORG_CLIENT_SECRET` |

---

### ent member — 成员管理

#### 添加成员
```
Usage:
  ykc ent member add [flags]
Example:
  ykc ent member add --account zhangsan@company.com --password Pass123 --name "张三"
  ykc ent member add --account zhangsan@company.com --password Zs123456 --name "张三" --group-path "/技术部"
  ykc ent member add --account zhangsan@company.com --password Pass123 --name "张三" --create-personal-org
Flags:
  --account string              帐号 (必需)
  --password string             初始密码 (必需)
  --name string                 成员名称 (必需)
  --phone string                联系电话
  --title string                职位
  --group-path string           成员所在部门路径
  --state int                   状态: 1启用, 0禁用
  --create-personal-org         初始化个人库 (不添加此参数则不初始化)
```

#### 修改成员
```
Usage:
  ykc ent member set [flags]
Example:
  ykc ent member set --account zhangsan@company.com --name "张三新" --state 1
Flags:
  --account string    帐号 (必需)
  --name string       成员名称
  --password string   密码
  --phone string      手机号
  --title string      职位
  --state int         状态: 1启用, 0禁用
```

#### 删除成员
```
Usage:
  ykc ent member delete [flags]
Example:
  ykc ent member delete --members "zhangsan@company.com|wangwu@company.com"
Flags:
  --members string   成员帐号，多个用竖号分隔 (必需)
```

#### 成员信息
```
Usage:
  ykc ent member info [flags]
Example:
  ykc ent member info --account zhangsan@company.com --show-groups --show-orgs
Flags:
  --account string     帐号
  --email string       邮箱
  --member-id string   成员ID
  --out-id string      外部帐号ID
  --show-groups        显示所属部门
  --show-orgs          显示所属库
```

#### 成员列表
```
Usage:
  ykc ent member list [flags]
Example:
  ykc ent member list --start 0 --size 50
Flags:
  --start int   开始位置
  --size int    返回条数 (默认20)
```

#### 删除成员的所属部门
```
Usage:
  ykc ent member del-group [flags]
Example:
  ykc ent member del-group --members "zhangsan@company.com"
  ykc ent member del-group --members "zhangsan@company.com,lisi@company.com"
Flags:
  --members string   成员帐号，多个用逗号分隔 (必需)
```

---

### ent group — 部门管理

#### 添加部门
```
Usage:
  ykc ent group add [flags]
Example:
  ykc ent group add --name "销售部" --parent-path "北京分公司"
Flags:
  --name string          部门名称 (必需)
  --parent-path string   父部门路径，空表示根部门
  --orderby int          排序值
  --state int            状态: 1启用, 0禁用
```

#### 修改部门
```
Usage:
  ykc ent group set [flags]
Example:
  ykc ent group set --path "北京分公司/销售部" --name "销售一部"
Flags:
  --path string          部门路径 (必需)
  --name string          部门名称
  --parent-path string   新的父部门路径
  --orderby int          排序值
  --state int            状态: 1启用, 0禁用
```

#### 删除部门
```
Usage:
  ykc ent group delete [flags]
Examples:
  ykc ent group delete --groups "/技术部"
  ykc ent group delete --groups "/技术部,/产品部"

Flags:
      --groups string   部门路径，多个用逗号分隔 (必需)
  -h, --help            help for delete
```

#### 部门列表
```
Usage:
  ykc ent group list [flags]
Example:
  ykc ent group list
```

#### 部门成员列表
```
Usage:
  ykc ent group members [flags]
Example:
  ykc ent group members --group-path "北京分公司/销售部" --show-child
Flags:
  --group-id string     部门ID
  --group-path string   部门路径
  --out-id string       部门外部ID
  --show-child          显示子部门成员
  --start int           开始位置
  --size int            返回条数 (默认100)
```

#### 添加部门成员
```
Usage:
  ykc ent group add-member [flags]
Examples:
  ykc ent group add-member --path "/技术部" --members "zhangsan@company.com"
  ykc ent group add-member --path "/技术部" --members "zhangsan@company.com,lisi@company.com"

Flags:
  -h, --help             help for add-member
      --members string   成员帐号，多个用逗号分隔 (必需)
      --path string      部门路径 (必需)
```

#### 删除部门成员
```
Usage:
  ykc ent group del-member [flags]
Examples:
  ykc ent group del-member --path "/技术部" --members "zhangsan@company.com"
  ykc ent group del-member --path "/技术部" --members "zhangsan@company.com,lisi@company.com"

Flags:
  -h, --help             help for del-member
      --members string   成员帐号，多个用逗号分隔 (必需)
      --path string      部门路径 (必需)
```

---

### ent sync — 同步操作

同步 API 用于将外部系统的组织架构同步到够快云库，需先开通企业帐号同步功能。

#### 添加或修改同步成员
```
Usage:
  ykc ent sync add-member [flags]
Examples:
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --group-out-ids "dept001,dept002"
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --password "Zs123456" --state 1
  ykc ent sync add-member --out-id "emp001" --name "张三" --account zhangsan --leader-out-id "emp002" --group-out-ids "dept001"
Flags:
  --out-id string           成员在外部系统的唯一ID (必需)
  --name string             成员显示名称 (必需)
  --account string          成员在外部系统的登录帐号 (必需)
  --email string            邮箱
  --phone string            联系电话
  --title string            职位
  --password string         密码，需要由云库校验帐号密码时传入
  --state int               状态: 1启用, 0禁用; 添加时传1启用, 修改时传0禁用
  --expire string           临时帐号过期日期, Unix时间戳(秒), 仅保留日期部分, 传0清除过期日期
  --group-out-ids string    成员所属部门的外部唯一ID, 多个用逗号分隔; 空字符串表示顶层部门; 添加时不传则成员不会出现在企业管理后台的成员管理中
  --leader-out-id string    直属上级在外部系统的唯一ID, 与leader-account二选一; 修改时传空字符串可清除上级属性
  --leader-account string   直属上级在外部系统的登录帐号, 与leader-out-id二选一
```

#### 删除同步成员
```
Usage:
  ykc ent sync del-member [flags]
Examples:
  ykc ent sync del-member --members "emp001"
  ykc ent sync del-member --members "emp001,emp002"
Flags:
  --members string   成员在外部系统的唯一ID, 多个用逗号分隔 (必需)
```

#### 删除同步成员的所属部门
```
Usage:
  ykc ent sync del-member-group [flags]
Examples:
  ykc ent sync del-member-group --members "emp001"
  ykc ent sync del-member-group --members "emp001,emp002"
Flags:
  --members string   成员在外部系统的唯一ID, 多个用逗号分隔 (必需)
```

#### 通过外部帐号获取成员信息
```
Usage:
  ykc ent sync get-member [flags]
Examples:
  ykc ent sync get-member --out-ids "emp001"
  ykc ent sync get-member --out-ids "emp001,emp002"
  ykc ent sync get-member --user-ids "123,456"
Flags:
  --out-ids string    外部成员ID，多个用逗号分隔，与user-ids二选一
  --user-ids string   外部成员登录帐号，多个用逗号分隔，与out-ids二选一
```

#### 添加或修改同步部门
```
Usage:
  ykc ent sync add-group [flags]
Example:
  ykc ent sync add-group --out-id dept001 --name "销售部"
Flags:
  --out-id string          外部部门ID (必需)
  --name string            部门名称 (必需)
  --parent-out-id string   父部门外部ID
  --orderby int            排序值
```

#### 删除同步部门
```
Usage:
  ykc ent sync del-group [flags]
Examples:
  ykc ent sync del-group --groups "dept001"
  ykc ent sync del-group --groups "dept001,dept002"
Flags:
  --groups string   部门在外部系统的唯一ID, 多个用逗号分隔 (必需)
```

#### 添加同步部门的成员
```
Usage:
  ykc ent sync add-group-member [flags]
Example:
  ykc ent sync add-group-member --group-out-id dept001 --members "user001|user002"
Flags:
  --group-out-id string   部门外部ID
  --members string        成员外部ID，多个用竖号分隔 (必需)
```

#### 删除同步部门的成员
```
Usage:
  ykc ent sync del-group-member [flags]
Examples:
  ykc ent sync del-group-member --group-out-id "dept001" --members "emp001"
  ykc ent sync del-group-member --group-out-id "dept001" --members "emp001,emp002"
Flags:
  --group-out-id string   部门在外部系统的唯一ID, 不传表示顶层部门
  --members string        成员在外部系统的唯一ID, 多个用逗号分隔 (必需)
```

#### 通过外部部门ID获取部门信息
```
Usage:
  ykc ent sync get-group [flags]
Examples:
  ykc ent sync get-group --out-ids "dept001"
  ykc ent sync get-group --out-ids "dept001,dept002"
Flags:
  --out-ids string   部门外部ID，多个用逗号分隔 (必需)
```

#### 添加管理员
```
Usage:
  ykc ent sync add-admin [flags]
Example:
  ykc ent sync add-admin --out-id user001 --type 1
Flags:
  --out-id string   外部帐号ID (必需)
  --email string    管理员邮箱
  --type int        管理员类型: 0普通管理员, 1超级管理员
```

---

### ent org — 企业库管理

#### 创建库
```
Usage:
  ykc ent org create [flags]
Example:
  ykc ent org create --name "项目文档库"
Flags:
  --name string            库名称 (必需)
  --capacity int           库空间上限(字节), -1表示不限制
  --logo string            库图标URL
  --out-id string          库外部ID
  --storage-point string   库归属存储点名称
```

#### 修改库
```
Usage:
  ykc ent org set [flags]
Example:
  ykc ent org set --org-id 123 --name "新库名称"
Flags:
  --org-id string     库ID
  --mount-id string   库空间ID
  --name string       库名称
  --capacity string   库空间上限(字节), 空字符串表示不设置上限
  --logo string       库图标URL
```

#### 库信息
```
Usage:
  ykc ent org info [flags]
Example:
  ykc ent org info --org-id 123
Flags:
  --org-id string     库ID
  --mount-id string   库空间ID
  --out-id string     库外部ID
```

#### 搜索库
```
Usage:
  ykc ent org search [flags]
Example:
  ykc ent org search --prefix "项目"
Flags:
  --name string     库名称，精确匹配
  --prefix string   库名称前缀，模糊匹配
  --size int        返回条数，不超过1000 (默认10)
```

#### 获取库列表
```
Usage:
  ykc ent org list [flags]
Example:
  ykc ent org list --type 1
Flags:
  --type string      1非个人文件库, 2个人文件库, 默认0所有
  --member-id string 只返回该成员参与的库
```

#### 删除库
```
Usage:
  ykc ent org destroy [flags]
Example:
  ykc ent org destroy --org-id 123
Flags:
  --org-id string   库ID
```

#### 库日志
```
Usage:
  ykc ent org log [flags]
Example:
  ykc ent org log --org-id 123 --size 100
  ykc ent org log --org-id 123 --act "1,2" --size 100
  ykc ent org log --mount-id 2 --start-dateline 1700000000 --end-dateline 1700100000
  ykc ent org log --org-id 123 --act "20,21" --orderby asc
Flags:
  --org-id string           库ID
  --mount-id string         库空间ID
  --act string              过滤操作类型, 多个用逗号分隔: 0删除, 1创建/上传, 2重命名, 3编辑, 4移动, 5还原已删除项, 6版本还原, 12锁定, 13解锁, 20下载, 21预览, 1014生成外链, 1015访问外链, 1016外链下载, 1017外链存库, 1018外链上传
  --orderby string          排序: asc顺序, desc倒序(默认)
  --start-dateline string   开始时间, Unix时间戳(秒), 获取操作时间 >= 该值的日志
  --end-dateline string     结束时间, Unix时间戳(秒), 获取操作时间 < 该值的日志
  --start int               开始位置
  --size int                获取日志条数, 最多1000 (默认100)
```

#### 获取库成员列表
```
Usage:
  ykc ent org members [flags]
Example:
  ykc ent org members --org-id 123
Flags:
  --org-id string   库ID (必需)
  --start int       开始位置
  --size int        返回条数 (默认20)
```

#### 查询库成员信息
```
Usage:
  ykc ent org member [flags]
Example:
  ykc ent org member --org-id 123 --ids "user001,user002" --type out_id
Flags:
  --org-id string   库ID (必需)
  --ids string      成员唯一ID或帐号, 多个用逗号分隔 (必需)
  --type string     ids的类型: account, out_id或member_id (必需)
```

#### 添加库成员
```
Usage:
  ykc ent org add-member [flags]
Example:
  ykc ent org add-member --org-id 123 --member-ids "456,789" --role-id 1
Flags:
  --org-id string       库ID (必需)
  --member-ids string   成员ID, 多个用逗号分隔 (必需)
  --role-id string      角色ID (必需)
```

#### 修改库成员角色
```
Usage:
  ykc ent org set-member-role [flags]
Example:
  ykc ent org set-member-role --org-id 123 --member-ids "456" --role-id 2
Flags:
  --org-id string       库ID (必需)
  --member-ids string   成员ID, 多个用逗号分隔 (必需)
  --role-id string      角色ID (必需)
```

#### 删除库成员
```
Usage:
  ykc ent org del-member [flags]
Example:
  ykc ent org del-member --org-id 123 --member-ids "456,789"
Flags:
  --org-id string       库ID (必需)
  --member-ids string   成员ID, 多个用逗号分隔 (必需)
```

#### 设置库拥有者
```
Usage:
  ykc ent org set-owner [flags]
Example:
  ykc ent org set-owner --org-id 123 --member-id 456
Flags:
  --org-id string    库ID (必需)
  --member-id string 设置为库拥有者的成员ID (必需)
  --role-id string   原库拥有者更换角色ID
```

#### 获取库部门列表
```
Usage:
  ykc ent org groups [flags]
Example:
  ykc ent org groups --org-id 123
Flags:
  --org-id string   库ID (必需)
```

#### 库上添加部门
```
Usage:
  ykc ent org add-group [flags]
Example:
  ykc ent org add-group --org-id 123 --group-id 1 --role-id 2
Flags:
  --org-id string    库ID (必需)
  --group-id string  部门ID (必需)
  --role-id string   角色ID (必需)
```

#### 修改库上部门的角色
```
Usage:
  ykc ent org set-group-role [flags]
Example:
  ykc ent org set-group-role --org-id 123 --group-id 1 --role-id 3
Flags:
  --org-id string    库ID (必需)
  --group-id string  部门ID (必需)
  --role-id string   角色ID (必需)
```

#### 删除库上的部门
```
Usage:
  ykc ent org del-group [flags]
Example:
  ykc ent org del-group --org-id 123 --group-id 1
Flags:
  --org-id string    库ID (必需)
  --group-id string  部门ID (必需)
```

#### 获取库授权
```
Usage:
  ykc ent org bind [flags]
Example:
  ykc ent org bind --org-id 123 --title "OA系统"
Flags:
  --org-id string     库ID
  --mount-id string   库空间ID
  --title string      对接库文件的应用或系统名称 (必需)
```

#### 取消库授权
```
Usage:
  ykc ent org unbind [flags]
Example:
  ykc ent org unbind --org-client-id "abc123"
Flags:
  -h, --help                   help for unbind
      --org-client-id string   库授权client_id (必需)
```

#### 个人文件库信息
```
Usage:
  ykc ent org info-by-member [flags]
Example:
  ykc ent org info-by-member --member-id 123
  ykc ent org info-by-member --account zhangsan@company.com
Flags:
  --account string     成员外部系统登录帐号
  --email string       登录邮箱
  --member-id string   成员ID
  --out-id string      成员外部系统唯一ID
```

#### 设置个人文件库
```
Usage:
  ykc ent org set-by-member [flags]
Example:
  ykc ent org set-by-member --member-id 123 --capacity 1073741824
  ykc ent org set-by-member --account zhangsan@company.com --capacity 10737418240
Flags:
  --account string     成员外部系统登录帐号
  --email string       登录邮箱
  --member-id string   成员ID
  --out-id string      成员外部系统唯一ID
  --capacity string    库空间(字节), -1表示不限制 (必需)
```

---

### ent roles — 角色列表
```
Usage:
  ykc ent roles [flags]
Example:
  ykc ent roles
Flags:
  -h, --help   help for roles
```

---

### ent log — 管理日志
```
Usage:
  ykc ent log [flags]
Example:
  ykc ent log
  ykc ent log --type 2 --start 0 --size 100
  ykc ent log --start-dateline 1700000000 --end-dateline 1700100000
Flags:
  --type string             日志类型: 1子管理员, 2成员信息, 3企业设置, 4组织架构, 5角色, 6成员设备, 7存储点, 8文件库, 9开发授权
  --orderby string          排序: asc顺序, desc倒序(默认)
  --start-dateline string   开始时间, Unix时间戳(秒), 获取操作时间 >= 该值的日志
  --end-dateline string     结束时间, Unix时间戳(秒), 获取操作时间 < 该值的日志
  --start int               开始位置
  --size int                获取日志条数, 最多1000 (默认100)
```

---

### ent oauth — 个人授权管理

#### 获取个人授权
```
Usage:
  ykc ent oauth get [flags]
Example:
  ykc ent oauth get
Flags:
  -h, --help   help for get
```

#### 删除个人授权
```
Usage:
  ykc ent oauth delete [flags]
Example:
  ykc ent oauth delete
Flags:
  -h, --help   help for delete
```

---

### ent file — 企业库文件操作

> 库授权获取方式二选一：
> 1. **自动绑定（推荐）**：命令带 `--org-id` 或 `--mount-id`（依赖企业凭证 `GOKUAI_CLIENT_ID`/`GOKUAI_CLIENT_SECRET`），CLI 自动换取库授权并缓存 1 小时，期间后续 `ent file` 命令可省略这两个参数
> 2. **直接配置**：设置环境变量 `GOKUAI_ORG_CLIENT_ID` 和 `GOKUAI_ORG_CLIENT_SECRET`（可通过 `ykc ent org bind` 获取）

#### 文件列表
```
Usage:
  ykc ent file ls [flags]
Example:
  ykc ent file ls --mount-id 1 --fullpath "/项目文档"
Flags:
  --fullpath string   文件夹路径，空字符串表示根目录
  --start int         开始位置
  --size int          返回条数 (默认100)
  --tag string        按标签过滤
```

#### 文件最近更新列表
```
Usage:
  ykc ent file updates [flags]
Example:
  ykc ent file updates --mount-id 1
  ykc ent file updates --mount-id 1 --fetch-dateline 1700000000000
  ykc ent file updates --mount-id 1 --mode compare --fetch-dateline 1700000000000
  ykc ent file updates --mount-id 1 --mode compare --fetch-dateline 0 --dir 1
Flags:
  --fetch-dateline int      Unix时间戳(毫秒)，获取此时间之后的更新，默认0
  --dir int                 1只返回文件夹, 0只返回文件, 不传返回全部
  --mode string             获取模式，空(倒序获取)或compare(正序获取，含已删除文件)
```

#### 文件更新数量
```
Usage:
  ykc ent file updates-count [flags]
Example:
  ykc ent file updates-count --mount-id 1 --begin-dateline 1700000000000 --end-dateline 1700100000000
Flags:
  --begin-dateline int   开始时间戳，单位毫秒 (必需)
  --end-dateline int     结束时间戳，单位毫秒 (必需)
  --showdel int          1同时返回删除的文件，默认0不返回
```

#### 文件信息
```
Usage:
  ykc ent file info [flags]

Examples:
  ykc ent file info --mount-id 1 --fullpath "/文档/readme.pdf"
  ykc ent file info --mount-id 1 --hash abc123
  ykc ent file info --mount-id 1 --fullpath "/文档" --attribute 1
  ykc ent file info --mount-id 1 --fullpath "/文档/readme.pdf" --ignores "tag,favorite,permission"

Flags:
      --attribute string   1获取额外属性(子文件数量/大小等), 10只返回基础信息, 默认0
      --fullpath string    文件路径, 需传fullpath或hash其中一个
      --hash string        文件唯一标识, 需传fullpath或hash其中一个
  -h, --help               help for info
      --hid string         版本ID
      --ignores string     忽略返回信息, 逗号隔开, 可选: tag,favorite,property,permission,lock,url,preview,thumbnail
      --net string         in表示获取内网下载链接, 默认空返回公网链接
```

#### 文件搜索
```
Usage:
  ykc ent file search [flags]

Examples:
  ykc ent file search --mount-id 1 --keywords "报告"
  ykc ent file search --mount-id 1 --keywords "报告" --path /文档 --size 50
  ykc ent file search --mount-id 1 --keywords "合同" --scope '["filename","content"]'

Flags:
      --keywords string   搜索关键字 (必需)
      --path string       需要搜索的文件夹, 默认空搜索整个库
      --scope string      搜索范围, JSON字符串, 默认["filename","tag"], filename按文件名、tag按标签、content按全文检索
      --start int         开始位置
      --size int          返回条数 (默认100)
  -h, --help              help for search
```

#### 文件下载链接
```
Usage:
  ykc ent file download [flags]

Aliases:
  download, download-url

Examples:
  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file download --mount-id 1 --hash abc123 --open
  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file download --mount-id 1 --fullpath /文档/文件.txt --output ./文件.txt

Flags:
      --filehash string   文件内容hash
      --filename string   文件名
      --fullpath string   文件路径
      --hash string       文件hash
  -h, --help              help for download
      --net string        网络类型
      --op-id string      操作者ID
      --op-name string    操作者名称
      --open int          是否直接打开: 1打开, 0不打开
      --output string     下载文件保存路径（指定后直接下载文件）
```

#### 文件预览链接
```
Usage:
  ykc ent file preview-url [flags]
Example:
  ykc ent file preview-url --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file preview-url --mount-id 1 --hash abc123 --watermark
Flags:
  --fullpath string      文件路径
  --hash string          文件唯一标识
  --watermark int        1显示水印
  --wm-content string    自定义水印内容
  --member-name string   水印显示的查看人姓名
  --thumbnail int        1返回缩略图链接
  --op-id int            操作人ID
  --op-name string       操作人名称
```

#### 获取文件导出链接
```
Usage:
  ykc ent file export-url [flags]
Example:
  ykc ent file export-url --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file export-url --mount-id 1 --hash abc123 --watermark
Flags:
      --fullpath string     文件路径
      --hash string         文件hash
  -h, --help                help for export-url
      --watermark           是否启用水印
      --wm-content string   水印内容
```

#### 文件协同编辑链接
```
Usage:
  ykc ent file cedit-url [flags]
Example:
  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --op-id 1
  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --out-id 1
  ykc ent file cedit-url --mount-id 1 --fullpath /文档/文件.docx --account 1
Flags:
      --account string    操作人外部系统帐号
      --fullpath string   文件路径, 需传 fullpath 或 hash 其中一个
      --hash string       文件唯一标识, 需传 fullpath 或 hash 其中一个
  -h, --help              help for cedit-url
      --op-id string      操作人ID, 可以用 out-id 或 account 代替
      --out-id string     操作人外部系统帐号ID
      --readonly          1表示只读打开, 默认允许编辑
      --timeout string    编辑链接过期时间, 单位秒, 默认 3600
```

#### 创建文件夹
```
Usage:
  ykc ent file create-folder [flags]
Example:
  ykc ent file create-folder --mount-id 1 --fullpath "/项目文档/新文件夹"
  ykc ent file create-folder --org-id 1 --fullpath "/项目文档/新文件夹"
Flags:
  --fullpath string   文件夹路径 (必需)
  --op-id int         操作人ID
  --op-name string    操作人名称
```

#### 上传文件（请求上传）
```
Usage:
  ykc ent file create-file [flags]
Example:
  # 上传本地文件（推荐，自动分块）
  ykc ent file create-file --mount-id 1 --file ./report.pdf --fullpath /文档/report.pdf

  # 指定分块大小（默认 4MB）
  ykc ent file create-file --mount-id 1 --file ./large.zip --fullpath /归档/large.zip --chunk-size 8388608

  # 仅调用 API（不上传内容，需自行提供哈希）
  ykc ent file create-file --mount-id 1 --fullpath /文档/文件.txt --filehash abc123 --filesize 1024
Flags:
  --fullpath string   目标文件路径 (必需)
```

#### 获取上传服务器
```
Usage:
  ykc ent file upload-servers [flags]
Example:
  ykc ent file upload-servers --mount-id 1 --fullpath /文档/文件.txt --rand abc123
Flags:
      --fullpath string   文件路径 (必需)
  -h, --help              help for upload-servers
      --rand string       随机数 (必需)
      --timeout int       超时时间(秒)，必须大于1 (default 300)
```

#### 复制文件(夹)
```
Usage:
  ykc ent file copy [flags]
Example:
  ykc ent file copy --mount-id 1 --from-fullpath "/文档/a.pdf" --fullpath "/备份/a.pdf"
Flags:
  --from-fullpath string   源文件路径 (必需)
  --fullpath string        目标完整路径 (必需)
  --overwrite int          1覆盖同名文件
  --op-id int              操作人ID
  --op-name string         操作人名称
```

#### 高级复制文件(夹)
```
Usage:
  ykc ent file mcopy [flags]

Examples:
  ykc ent file mcopy --mount-id 1 --from-fullpaths "/a.txt|/b.txt" --paths "/目标文件夹1|/目标文件夹2"
  ykc ent file mcopy --mount-id 1 --from-fullpaths "/a.txt" --paths "/备份" --copy-all 1

Flags:
      --copy-all int            1复制文件的所有属性(包括操作人), 默认0不复制
      --from-fullpaths string   源文件路径, 多个用竖号分隔 (必需)
  -h, --help                    help for mcopy
      --op-id string            操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op-name代替
      --op-name string          操作人名称, 如果指定了op-id, 就不需要op-name
      --paths string            目标文件夹路径(不包含文件名), 多个用竖号分隔 (必需)
```

#### 移动文件(夹)
```
Usage:
  ykc ent file move [flags]
Example:
  ykc ent file move --mount-id 1 --fullpath /文档/文件.txt --dest-fullpath /备份/文件.txt
Flags:
  --fullpath string        要移动的文件路径 (必需)
  --dest-fullpath string   移动后的路径 (必需)
  --op-id int              操作人ID
  --op-name string         操作人名称
```

#### 删除文件(夹)
```
Usage:
  ykc ent file del [flags]
Example:
  ykc ent file del --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file del --mount-id 1 --tag "tag1;tag2"
  ykc ent file del --mount-id 1 --fullpath /文档/文件.txt --destroy
Flags:
  --fullpath string   文件完整路径, fullpath 和 tag 只需传其中一个
  --tag string        通过标签删除, 多个用分号分隔, fullpath 和 tag 只需传其中一个
  --path string       当使用 tag 方式删除时, 可以指定路径进行删除, 默认空不指定
  --destroy           彻底删除文件不进回收站 (不添加此参数则删除进入回收站)
  --op-id string      操作人ID, 如果操作人不是云库用户, 可以用 op-name 代替
  --op-name string    操作人名称, 如果指定了 op-id, 就不需要 op-name
```

#### 彻底删除文件(夹)
```
Usage:
  ykc ent file del-completely [flags]
Example:
  ykc ent file del-completely --mount-id 1 --fullpaths "/a.txt|/b.txt"
Flags:
      --fullpaths string   文件路径，多个用竖号分隔
  -h, --help               help for del-completely
      --op-id string       操作者ID
      --op-name string     操作者名称
      --tag string         标签
```

#### 回收站
```
Usage:
  ykc ent file recycle [flags]
Example:
  ykc ent file recycle --mount-id 1
  ykc ent file recycle --mount-id 1 --start 0 --size 50
Flags:
  --keyword string   搜索关键词
  --order string     排序字段
  --start int        开始位置
  --size int         返回条数 (默认100)
```

#### 恢复已删除文件
```
Usage:
  ykc ent file recover [flags]
Example:
  ykc ent file recover --mount-id 1 --fullpaths "/a.txt|/b.txt"
Flags:
  --fullpaths string   文件路径，多个用竖号分隔 (必需)
  --op-id int          操作人ID
  --op-name string     操作人名称
```

#### 获取文件历史
```
Usage:
  ykc ent file history [flags]
Example:
  ykc ent file history --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file history --mount-id 1 --fullpath /文档/文件.txt --start 0 --size 10
Flags:
  --fullpath string   文件完整路径 (必需)
  --start int         开始位置
  --size int          返回条数 (默认20)
```

#### 获取文件外链
```
Usage:
  ykc ent file link [flags]

Examples:
  ykc ent file link --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file link --mount-id 1 --fullpath /文档/文件.txt --deadline 1700100000

Flags:
      --access-limit int    访问次数限制，不传则不限制
      --auth string         权限: preview(仅预览)/download(预览+下载)/upload(预览+下载+文件夹上传), 默认preview (default "preview")
      --deadline int        到期时间戳, 默认48小时后
      --fullpath string     文件完整路径 (必需)
  -h, --help                help for link
      --keep                不随文件修改更新外链 (不添加此参数则随文件更新)
      --op-id int           操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op-name代替
      --op-name string      操作人名称, 如果指定了op-id, 就不需要op-name
      --password string     访问密码
      --startline int       生效时间戳, 不传则立即生效
      --wm-content string   自定义水印内容
```

#### 关闭文件外链
```
Usage:
  ykc ent file link-close [flags]
Example:
  ykc ent file link-close --mount-id 1 --fullpath /文档/文件.txt
  ykc ent file link-close --mount-id 1 --code "abc123"
Flags:
  --code string        外链码
  --fullpath string    文件完整路径
  --op-id int          操作人ID
  --op-name string     操作人名称
```

#### 获取开启外链的文件列表
```
Usage:
  ykc ent file links [flags]
Example:
  ykc ent file links --mount-id 1
  ykc ent file links --mount-id 1 --file 1
Flags:
      --file int   1仅返回文件, 0返回全部, 默认0
  -h, --help       help for links
```

#### 文件锁操作
```
Usage:
  ykc ent file lock [flags]
Example:
  ykc ent file lock --mount-id 1 --fullpath /文档/文件.txt --lock lock
  ykc ent file lock --mount-id 1 --fullpath /文档/文件.txt --lock unlock
Flags:
  --fullpath string   文件完整路径 (必需)
  --lock string       lock上锁, unlock解锁 (必需)
  --op-id int         操作人ID
  --op-name string    操作人名称
```

#### 添加标签
```
Usage:
  ykc ent file add-tag [flags]
Example:
  ykc ent file add-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要"
  ykc ent file add-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要;紧急"
Flags:
  --fullpath string   文件完整路径 (必需)
  --tag string        标签, 多个使用分号;分隔 (必需)
  --op-id string      操作人ID, 个人库默认是库拥有人ID, 如果操作人不是云库用户, 可以用op_name代替
  --op-name string    操作人名称, 如果指定了op_id, 就不需要op_name
```

#### 删除标签
```
Usage:
  ykc ent file del-tag [flags]
Example:
  ykc ent file del-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要"
  ykc ent file del-tag --mount-id 1 --fullpath /文档/文件.txt --tag "重要;紧急"
Flags:
  --fullpath string   文件完整路径 (必需)
  --tag string        标签, 多个使用分号;分隔 (必需)
  --op-id string      操作人ID, 如果操作人不是云库用户, 可以用 op-name 代替
  --op-name string    操作人名称, 如果指定了 op-id, 就不需要 op-name
```

#### 添加或修改元数据
```
Usage:
  ykc ent file set-metadata [flags]
Example:
  ykc ent file set-metadata --mount-id 1 --fullpath "/文档/a.pdf" --key "template_key" --metadata '{"field_key":"值"}'
Flags:
  --fullpath string    文件路径
  --hash string        文件唯一标识
  --key string         元数据模版key (必需)
  --metadata string    JSON字符串 (必需)
  --display string     展示的属性，逗号分隔
```

#### 删除元数据
```
Usage:
  ykc ent file del-metadata [flags]
Example:
  ykc ent file del-metadata --mount-id 1 --fullpath "/文档/a.pdf" --key "template_key"
Flags:
  --fullpath string   文件路径
  --hash string       文件唯一标识
  --key string        元数据模版key (必需)
```

#### 统计信息
```
Usage:
  ykc ent file stat [flags]
Example:
  ykc ent file stat --mount-id 1
Flags:
  -h, --help   help for stat
```

#### 查询队列状态
```
Usage:
  ykc ent file queue-status [flags]
Example:
  ykc ent file queue-status --queue-id "xxx"
Flags:
  --queue-id string   队列ID (必需)
```

#### 权限相关
```
# 设置文件夹权限继承状态
ykc ent file set-permission-inherit --mount-id 1 --fullpath /文档 --inherit
ykc ent file set-permission-inherit --mount-id 1 --fullpath /文档


# 获取文件夹单独设置的权限
ykc ent file get-all-permission --mount-id 1 --fullpath "/项目"


# 获取文件权限
ykc ent file get-permission  --mount-id 1 --fullpath "/文档/a.pdf"

# 修改文件夹权限
Usage:
  ykc ent file file-permission [flags]

Examples:
  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"user123":["file_preview","file_read"]}'
  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"group456":["file_preview","file_read"]}' --is-group
  ykc ent file file-permission --mount-id 1 --fullpath "/项目" --permissions '{"out_user789":["file_preview"]}' --is-out

Flags:
      --fullpath string      文件夹完整路径 (必需)
  -h, --help                 help for file-permission
      --is-group             设置部门权限 (不添加此参数则设置用户权限)
      --is-out               是否外部成员 (不添加此参数则为否)
      --permissions string   权限设置, 如 '{"用户ID":["file_preview","file_read"]}' (必需)

# 重置或移除文件夹权限
```
Usage:
  ykc ent file reset-permission [flags]

Examples:
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --members 1,2
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --members 1,2 --clear
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --groups 1
  ykc ent file reset-permission --mount-id 1 --fullpath /文档 --groups 1 --clear

Flags:
      --clear             清除通过部门加入的成员权限, 仅在文件夹非继承状态时生效, 默认不清除
      --fullpath string   文件夹完整路径 (必需)
      --groups string     要重置或移除的部门ID, 多个用逗号分隔
  -h, --help              help for reset-permission
      --members string    要重置或移除的成员ID, 多个用逗号分隔
```

---

## 意图判断

### 企业管理
- 说"同步成员/部门"、"从外部系统导入" → `ent sync`
- 说"添加/修改/删除成员" → `ent member add/set/delete`
- 说"成员信息/列表" → `ent member info/list`
- 说"添加/修改/删除部门" → `ent group add/set/delete`
- 说"创建/查询/删除库"（企业视角）→ `ent org create/info/destroy`
- 说"库成员/库部门管理"（企业视角）→ `ent org add-member/del-member/add-group/del-group`
- 说"管理日志" → `ent log`
- 说"角色列表" → `ent roles`
- 说"企业文件操作/库文件管理"（有 org_client_id）→ `ent file`

## 上下文传递表

| 操作 | 提取字段 | 用于 |
|------|---------|------|
| `ent org create` | `org_id` | `ent org set/bind/destroy` 的 `--org-id` |
| `ent org bind` | `org_client_id` `org_client_secret` | `GOKUAI_ORG_CLIENT_ID` `GOKUAI_ORG_CLIENT_SECRET` 环境变量 |
| `ent roles` | `role_id` | `ent org add-member` 的 `--role-id` |
| `ent member list` | `member_id` | `ent org add-member/set-owner` 的 `--member-id` |
