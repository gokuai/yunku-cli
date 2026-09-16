# YKC Skill 能力验证测试集

## 目标

验证 `ykc` skill 在各 AI Agent 工具中的**意图理解→CLI 命令生成**能力。
Agent 安装 ykc skill 后，仅依据 skill 提供的参考文档，将自然语言意图翻译为 `ykc` CLI 命令。

## 一句话启动测试

复制以下 prompt 发给 Agent 即可：

```
帮我跑一下 skill_tests.md 里的全部测试用例。
每条用例读 Prompt 后用 ykc 命令实现，和 Expected 对比，输出 PASS/FAIL。
把每条用例的详细测试情况都打印出来。特别是依赖skill的哪个信息去判断的。
最后给个汇总：总数、通过数、按产品分组通过率、所有 FAIL 的详情。
以上所有输出内容都写入 skill_tests_results.md
```

## 校验规则

| 维度 | 规则 |
|------|------|
| **命令路径** | `ykc` 之后、第一个 `--` 之前的 token 序列必须与用例的命令路径一致 |
| **参数** | 每个 `--key value` 对必须与 Flags 中列出的一致（`--format json` 忽略）|
| **占位符** | `<...>` 形式的值表示动态 ID，验证时只需确认 key 存在 |
| **[ASK_USER]** | Agent 应识别出缺少必要信息需要追问用户，只验证命令路径 |
| **参数顺序** | 不要求参数顺序一致，只要 key-value 内容正确即可 |

## 评分标准

| 指标 | 计算方式 |
|------|---------|
| **命令准确率** | 命令路径正确的用例数 / 总用例数 × 100% |
| **参数准确率** | 命令路径和所有参数都正确的用例数 / 总用例数 × 100% |
| **产品识别率** | 正确选择产品命令的用例数 / 总用例数 × 100% |

## 测试覆盖总览

| 产品 | Skill 参考文档 | 命令数 | 用例数 |
|------|---------------|--------|--------|
| `contact` | `references/products/contact.md` | 6 | 12 |
| `ent` | `references/products/ent.md` | 6 | 12 |
| `auth` | `references/products/auth.md` | 4 | 8 |

---

## 测试用例

### contact（14 条）

#### `ykc contact group search`

**contact_group_search_001**
- Prompt: 搜索企业 12345 中名称含"技术部"的部门
- Expected: `ykc contact group search --ent-id 12345 --keyword 技术部 --format json`
- Flags: `--ent-id` = `12345`, `--keyword` = `技术部`

**contact_group_search_002**
- Prompt: 帮我找一下企业 99999 里的产品部
- Expected: `ykc contact group search --ent-id 99999 --keyword "产品部" --format json`
- Flags: `--ent-id` = `99999`, `--keyword` = `产品部`

#### `ykc contact group list`

**contact_group_list_001**
- Prompt: 列出企业 12345 的所有部门
- Expected: `ykc contact group list --ent-id 12345 --format json`
- Flags: `--ent-id` = `12345`

**contact_group_list_002**
- Prompt: 查看企业 88888 下所有部门
- Expected: `ykc contact group list --ent-id 88888 --format json`
- Flags: `--ent-id` = `88888`

#### `ykc contact group members`

**contact_group_members_001**
- Prompt: 查看企业 12345 部门 67890 的成员列表
- Expected: `ykc contact group members --ent-id 12345 --group-id 67890 --format json`
- Flags: `--ent-id` = `12345`, `--group-id` = `67890`

**contact_group_members_002**
- Prompt: 技术部（ID 11111）在企业 22222 里有哪些员工
- Expected: `ykc contact group members --ent-id 22222 --group-id 11111 --format json`
- Flags: `--ent-id` = `22222`, `--group-id` = `11111`

#### `ykc contact member info`

**contact_member_info_001**
- Prompt: 查询企业 12345 中成员 user001 的信息
- Expected: `ykc contact member info --ent-id 12345 --member-id user001 --format json`
- Flags: `--ent-id` = `12345`, `--member-id` = `user001`

**contact_member_info_002**
- Prompt: 企业 99999 里员工 empABC 的详细资料
- Expected: `ykc contact member info --ent-id 99999 --member-id empABC --format json`
- Flags: `--ent-id` = `99999`, `--member-id` = `empABC`

#### `ykc contact member groups`

**contact_member_groups_001**
- Prompt: 查看企业 12345 成员 user001 属于哪些部门
- Expected: `ykc contact member groups --ent-id 12345 --member-id user001 --format json`
- Flags: `--ent-id` = `12345`, `--member-id` = `user001`

**contact_member_groups_002**
- Prompt: 员工 empXYZ 在企业 77777 的部门归属
- Expected: `ykc contact member groups --ent-id 77777 --member-id empXYZ --format json`
- Flags: `--ent-id` = `77777`, `--member-id` = `empXYZ`

#### `ykc contact root-group`

**contact_root_group_001**
- Prompt: 获取企业 12345 的根部门信息
- Expected: `ykc contact root-group --ent-id 12345 --format json`
- Flags: `--ent-id` = `12345`

**contact_root_group_002**
- Prompt: 企业 33333 的顶级部门是什么
- Expected: `ykc contact root-group --ent-id 33333 --format json`
- Flags: `--ent-id` = `33333`

---

### ent（12 条）

> `ent` 命令操作的企业由环境变量 `GOKUAI_CLIENT_ID`/`GOKUAI_CLIENT_SECRET` 决定，无需也不存在 `--ent-id` 参数。

#### `ykc ent member list`

**ent_member_list_001**
- Prompt: 列出当前企业的所有员工
- Expected: `ykc ent member list --format json`

**ent_member_list_002**
- Prompt: 查看企业成员列表，从第 0 条开始取 50 条
- Expected: `ykc ent member list --start 0 --size 50 --format json`
- Flags: `--start` = `0`, `--size` = `50`

#### `ykc ent group list`

**ent_group_list_001**
- Prompt: 列出当前企业的部门树
- Expected: `ykc ent group list --format json`

**ent_group_list_002**
- Prompt: 企业里有哪些部门
- Expected: `ykc ent group list --format json`

#### `ykc ent sync get-member`

**ent_sync_get_member_001**
- Prompt: 通过外部帐号 emp001 查询同步成员信息
- Expected: `ykc ent sync get-member --out-ids emp001 --format json`
- Flags: `--out-ids` = `emp001`

**ent_sync_get_member_002**
- Prompt: 查一下外部 ID 为 emp001 和 emp002 的两个同步成员
- Expected: `ykc ent sync get-member --out-ids "emp001|emp002" --format json`
- Flags: `--out-ids` = `emp001|emp002`

#### `ykc ent org list`

**ent_org_list_001**
- Prompt: 列出企业的库列表
- Expected: `ykc ent org list --format json`

**ent_org_list_002**
- Prompt: 查看所有非个人文件库
- Expected: `ykc ent org list --type 1 --format json`
- Flags: `--type` = `1`

#### `ykc ent log`

**ent_log_001**
- Prompt: 查看企业的管理日志
- Expected: `ykc ent log --format json`

**ent_log_002**
- Prompt: 查最近 50 条管理日志
- Expected: `ykc ent log --size 50 --format json`
- Flags: `--size` = `50`

#### `ykc ent roles`

**ent_roles_001**
- Prompt: 查看企业的角色列表
- Expected: `ykc ent roles --format json`

**ent_roles_002**
- Prompt: 企业有哪些角色
- Expected: `ykc ent roles --format json`

---

### auth（8 条）

#### `ykc auth login`

**auth_login_001**
- Prompt: 用账号 user@example.com 密码 secret123 登录
- Expected: `ykc auth login --username user@example.com --password secret123`
- Flags: `--username` = `user@example.com`, `--password` = `secret123`

**auth_login_002** `[ASK_USER]`
- Prompt: 帮我登录
- Expected: `ykc auth login`

#### `ykc auth status`

**auth_status_001**
- Prompt: 查看当前认证状态
- Expected: `ykc auth status --format json`

**auth_status_002**
- Prompt: 我登录了吗？检查一下 token 状态
- Expected: `ykc auth status --format json`

#### `ykc auth refresh`

**auth_refresh_001**
- Prompt: 刷新一下 Access Token
- Expected: `ykc auth refresh --format json`

**auth_refresh_002**
- Prompt: token 快过期了，帮我续期
- Expected: `ykc auth refresh --format json`

#### `ykc auth logout`

**auth_logout_001**
- Prompt: 退出登录
- Expected: `ykc auth logout`

**auth_logout_002**
- Prompt: 清除认证信息，登出
- Expected: `ykc auth logout`
