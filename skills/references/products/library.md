# library — 库管理命令参考（用户 API）

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc library` | 用户 access_token | `ykc auth login` 登录后自动管理 |

> 需要先 `ykc auth login` 登录

---

#### 库列表
```
Usage:
  ykc library list [flags]
Example:
  ykc library list
```

#### 获取库信息
```
Usage:
  ykc library info [flags]
Example:
  ykc library info --org-id 123
Flags:
  --org-id int   库ID (必需)
```

#### 创建库
```
Usage:
  ykc library create [flags]

Examples:
  ykc library create --name "新库" --ent-id 0
  ykc library create --name "新库" --ent-id 0 --capacity 1073741824

Flags:
      --capacity int           空间上限(字节)，-1表示不限制 (default -1)
      --ent-id int             企业ID (必需，0表示个人)
  -h, --help                   help for create
      --name string            库名称 (必需)
      --storage-point string   存储点名称
```

#### 更新库信息
```
Usage:
  ykc library update [flags]
Example:
  ykc library update --org-id 123 --name "新库名称"
Flags:
  --org-id string   库ID
  --name string     库名称
```

#### 删除库
```
Usage:
  ykc library delete [flags]
Example:
  ykc library delete --org-id 123
Flags:
  --org-id string   库ID (必需)
```

#### 获取库成员列表
```
Usage:
  ykc library members [flags]

Examples:
  ykc library members --org-id 919427
  ykc library members --org-id 919427 --start 0 --size 20
  ykc library members --org-id 919427 --state 1
  ykc library members --org-id 919427 --keyword test
  ykc library members --org-id 919427 --with-info --with-group
  ykc library members --org-id 919427 --group-fullpath
  ykc library members --org-id 919427 --role-ids 17042
  ykc library members --org-id 919427 --is-out

Flags:
      --group-fullpath   返回成员部门路径
  -h, --help                 help for members
      --is-out           是否外部成员
      --keyword string       搜索关键字
      --org-id int           库ID (必需)
      --role-ids string      角色ID
      --size int             数量，不传拿全部
      --start int            开始位置
      --state int            成员状态: 1启用, 0禁用
      --with-group       返回成员分组信息
      --with-info        返回成员详细信息
```

#### 添加库成员
```
Usage:
  ykc library add-member [flags]

Examples:
  ykc library add-member --org-id 1 --member-ids "123,456" --role-id 2

Flags:
  -h, --help                help for add-member
      --member-ids string   成员ID,以逗号分隔 (必需)
      --org-id string       库ID (必需)
      --role-id string      角色ID (必需)
```

#### 更新库成员
```
Usage:
  ykc library update-member [flags]

Examples:
  ykc library update-member --org-id 1 --member-ids "123,456" --role-id 2
  ykc library update-member --org-id 1 --member-ids "123" --state 1

Flags:
  -h, --help                help for update-member
      --member-ids string   成员ID,以逗号分隔 (必需)
      --org-id string       库ID (必需)
      --role-id string      角色ID
      --state string        成员状态: 0禁用 1正常
```

#### 移除库成员
```
Usage:
  ykc library remove-member [flags]
Example:
  ykc library remove-member --org-id 123 --member-ids "456,789"
Flags:
  --org-id string       库ID (必需)
  --member-ids string   成员ID列表, 逗号分隔 (必需)
```

#### 获取库部门列表
```
Usage:
  ykc library groups [flags]
Example:
  ykc library groups --org-id 123
Flags:
  --org-id string   库ID (必需)
  --with-info       返回详细信息
```

#### 添加库部门
```
Usage:
  ykc library add-group [flags]

Examples:
  ykc library add-group --ent-id 1 --org-id 2 --group-id 3 --role-id 4

Flags:
      --ent-id string     企业ID (必需)
      --group-id string   部门ID (必需)
  -h, --help              help for add-group
      --org-id string     库ID (必需)
      --role-id string    角色ID (必需)
```

#### 更新库部门
```
Usage:
  ykc library update-group [flags]

Examples:
  ykc library update-group --ent-id 1 --org-id 2 --group-id 3 --role-id 4

Flags:
      --ent-id string     企业ID (必需)
      --group-id string   部门ID (必需)
  -h, --help              help for update-group
      --org-id string     库ID (必需)
      --role-id string    角色ID (必需)
```

#### 移除库部门
```
Usage:
  ykc library remove-group [flags]

Examples:
  ykc library remove-group --ent-id 1 --org-id 2 --group-id 3

Flags:
      --ent-id string     企业ID (必需)
      --group-id string   部门ID (必需)
  -h, --help              help for remove-group
      --org-id string     库ID (必需)
```

#### 搜索库部门
```
Usage:
  ykc library search-group [flags]
Example:
  ykc library search-group --org-id 123 --keyword "研发"
Flags:
  --org-id string    库ID (必需)
  --keyword string   搜索关键字 (必需)
```

#### 获取文件标签
```
Usage:
  ykc library tags [flags]
Example:
  ykc library tags --org-id 123
Flags:
  --org-id string     库ID
  --ent-id string     企业ID
  --mount-id string   挂载点ID
  --recent            只获取最近使用的标签
```

---

## 意图判断

- 说"库列表/库信息"（用户视角）→ `library list/info`

## 上下文传递表

| 操作 | 提取字段 | 用于 |
|------|---------|------|
| `library list/info` | `org_id` | `library` 子命令的 `--org-id` |
