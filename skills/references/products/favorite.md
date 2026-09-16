# favorite — 收藏管理命令参考（用户 API）

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc favorite` | 用户 access_token | `ykc auth login` 登录后自动管理 |

> 需要先 `ykc auth login` 登录

---

#### 获取收藏夹文件列表
```
Usage:
  ykc favorite list [flags]
Example:
  ykc favorite list
Flags:
  --fav-id int   收藏夹ID，-1表示默认收藏夹 (默认-1)
  --start int    开始位置
  --size int     返回条数 (默认20)
```

#### 添加文件到收藏夹
```
Usage:
  ykc favorite add [flags]
Example:
  ykc favorite add --mount-id 1 --fullpath "/文档/a.pdf"
Flags:
  --mount-id int      文件库ID (必需)
  --fullpath string   文件路径 (必需)
  --fav-id int        收藏夹ID，-1表示默认收藏夹 (默认-1)
```

#### 从收藏夹移除文件
```
Usage:
  ykc favorite remove [flags]

Examples:
  ykc favorite remove --mount-id 1 --fullpath /foo/bar.txt --fav-id 2

Flags:
      --fav-id int        收藏夹ID (必需) (default -1)
      --fullpath string   文件路径 (必需)
  -h, --help              help for remove
      --mount-id int      文件库ID (必需)
```

---

## 意图判断

- 说"收藏" → `favorite add/list/remove`
