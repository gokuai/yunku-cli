# 通讯录 (contact) 命令参考

> 需要先 `ykc auth login` 登录

## 命令总览

### group — 部门管理

#### 获取部门列表
```
Usage:
  ykc contact group list [flags]
Example:
  ykc contact group list --ent-id 100 --group-id 0
Flags:
  --ent-id string     企业ID (必需)
  --group-id string   部门ID，0返回一级部门 (默认"0")
  --filter string     0表示忽略部门禁用和隐藏规则
```

#### 查询部门列表
```
Usage:
  ykc contact group search [flags]
Example:
  ykc contact group search --ent-id 100 --keyword "技术部"
Flags:
  --ent-id string    企业ID (必需)
  --keyword string   搜索关键字 (必需)
```

#### 获取部门成员列表
```
Usage:
  ykc contact group members [flags]
Example:
  ykc contact group members --ent-id 100 --group-id 12345
Flags:
  --ent-id string     企业ID (必需)
  --group-id string   部门ID
  --keyword string    搜索关键字
  --show-child        显示子部门成员
  --filter string     0表示忽略部门禁用和隐藏规则
  --start int         开始位置
  --size int          返回条数
```

#### 添加部门
```
Usage:
  ykc contact group add [flags]
Example:
  ykc contact group add --ent-id 100 --name "研发部" --parent-id 0
Flags:
  --ent-id string      企业ID (必需)
  --name string        部门名称 (必需)
  --parent-id string   父部门ID，0为根部门 (默认"0")
  --state string       部门状态: 1启用, 0禁用
```

#### 更新部门
```
Usage:
  ykc contact group update [flags]
Example:
  ykc contact group update --ent-id 100 --group-id 12345 --name "研发一部"
Flags:
  --ent-id string     企业ID (必需)
  --group-id string   部门ID (必需)
  --name string       部门名称
  --state string      部门状态: 1启用, 0禁用
```

#### 删除部门
```
Usage:
  ykc contact group delete [flags]
Example:
  ykc contact group delete --ent-id 100 --group-id 12345
Flags:
  --ent-id string     企业ID (必需)
  --group-id string   部门ID (必需)
```

#### 添加部门成员
```
Usage:
  ykc contact group add-member [flags]
Example:
  ykc contact group add-member --ent-id 100 --group-id 12345 --member-ids "456,789"
Flags:
  --ent-id string       企业ID (必需)
  --group-id string     部门ID
  --member-ids string   成员ID，逗号分隔 (必需)
```

#### 移除部门成员
```
Usage:
  ykc contact group remove-member [flags]
Example:
  ykc contact group remove-member --ent-id 100 --member-ids "456"
Flags:
  --ent-id string       企业ID (必需)
  --group-id string     部门ID
  --member-ids string   成员ID，逗号分隔 (必需)
```

---

### member — 成员管理

#### 获取成员信息
```
Usage:
  ykc contact member info [flags]
Example:
  ykc contact member info --ent-id 100 --member-id 456
Flags:
  --ent-id string      企业ID (必需)
  --member-id string   成员ID (必需)
  --with-groups        显示所属部门
  --show-space         显示个人空间使用量
```

#### 获取成员所属部门列表
```
Usage:
  ykc contact member groups [flags]
Example:
  ykc contact member groups --ent-id 100 --member-id 456
Flags:
  --ent-id string      企业ID (必需)
  --member-id string   成员ID (必需)
```

#### 添加成员
```
Usage:
  ykc contact member add [flags]
Example:
  ykc contact member add --ent-id 100 --name "张三" --password "Pass123"
Flags:
  --ent-id string      企业ID (必需)
  --name string        成员名称 (必需)
  --account string     帐号
  --email string       邮箱地址
  --password string    登录密码
  --phone string       手机号
  --title string       职位
  --group-id string    部门ID，默认根部门，-1不加入部门
  --group-ids string   多个部门ID，逗号分隔
  --leader-id string   直属上级用户ID
  --expire string      临时成员过期日期，格式: 1970-01-01
```

#### 修改成员
```
Usage:
  ykc contact member update [flags]
Example:
  ykc contact member update --ent-id 100 --member-ids "456" --name "张三新"
Flags:
  --ent-id string       企业ID (必需)
  --member-ids string   成员ID，逗号分隔 (必需)
  --name string         成员名称
  --email string        邮箱地址
  --password string     登录密码
  --phone string        手机号
  --title string        职位
  --state string        成员状态: 1启用, 0禁用
  --group-ids string    所有部门ID，逗号分隔
  --leader-id string    直属上级用户ID
  --expire string       临时成员过期日期
```

#### 移除成员
```
Usage:
  ykc contact member remove [flags]
Example:
  ykc contact member remove --ent-id 100 --member-ids "456" --to-member-id 789
Flags:
  --ent-id string         企业ID (必需)
  --member-ids string     成员ID，逗号分隔 (必需)
  --to-member-id string   转移库给此成员ID (必需)
```

---

### 其他命令

#### 获取根部门属性
```
Usage:
  ykc contact root-group [flags]
Example:
  ykc contact root-group --ent-id 100
Flags:
  --ent-id string   企业ID (必需)
```

---

## 意图判断

- 说"找部门/哪个部门/搜部门" → `contact group search`（需要 keyword）
- 说"部门列表/查看部门" → `contact group list`（需要 ent-id）
- 说"部门有谁/部门成员" → `contact group members`（需要 ent-id + group-id）
- 说"添加/修改/删除部门" → `contact group add/update/delete`
- 说"添加部门成员/移除部门成员" → `contact group add-member/remove-member`
- 说"查成员信息" → `contact member info`（需要 ent-id + member-id）
- 说"成员所在部门" → `contact member groups`
- 说"添加成员" → `contact member add`
- 说"修改成员/禁用成员" → `contact member update`
- 说"删除成员/移除成员" → `contact member remove`
- 说"根部门" → `contact root-group`

## 核心工作流

```bash
# 1. 搜索部门，获取 group-id
ykc contact group search --ent-id <ent-id> --keyword "技术部" -f json

# 2. 查看部门成员
ykc contact group members --ent-id <ent-id> --group-id <group-id> -f table

# 3. 添加成员到指定部门
ykc contact member add --ent-id <ent-id> --name "张三" --password "Pass123" --group-id <group-id>

# 4. 查看成员信息
ykc contact member info --ent-id <ent-id> --member-id <member-id> --with-groups

# 5. 修改成员状态（禁用）
ykc contact member update --ent-id <ent-id> --member-ids "<member-id>" --state 0
```

## 上下文传递表

| 操作 | 提取字段 | 用于 |
|------|---------|------|
| `account ent-info` / `account info` | `ent_id` | 所有 contact 命令的 `--ent-id` |
| `contact group list/search` | `group_id` | `contact group members/update/delete` 的 `--group-id` |
| `contact group members` | `member_id` | `contact member info/update/remove` 的 `--member-id` |
| `contact member add` | `member_id` | 后续成员操作 |

## 注意事项

- `--ent-id` 是所有 contact 命令的必填参数，可通过 `ykc account ent-info` 或 `ykc account info` 获取
- `contact member remove` 必须提供 `--to-member-id`（接收库的成员），被移除成员的文件库会转移给该成员
- `contact group members` 支持 `--show-child` 递归获取子部门成员
