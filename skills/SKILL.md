---
name: ykc
description: 管理够快云库(GoKuai)API：企业成员/部门管理与组织同步、企业库管理、库文件操作(上传下载/外链/权限)、通讯录查询、用户文件与收藏管理、账户与设备管理。当用户需要操作够快云库的文件、库、部门、成员、收藏或账户信息时使用。
cli_version: ">=1.0.0"
---

# 够快云库 Skill

通过 `ykc` 命令管理够快云库(GoKuai)企业与用户资源。

## 严格禁止 (NEVER DO)
- 不要使用 ykc 命令以外的方式操作（禁止 curl、HTTP API、浏览器）
- 不要编造 UUID、ID 等标识符，必须从命令返回中提取
- 不要猜测字段名/参数值，操作前必须先查询确认

## 严格要求 (MUST DO)
- 所有命令必须加 `--format json` 以获取可解析输出
- 危险操作必须先向用户确认，用户同意后才加 `--yes` 执行
- 单次批量操作不超过 30 条记录
- 所有命令必须**严格遵循**对应产品参考文档里面规定的参数格式（如：如果有参数值，则参数和参数值之间至少用一个空格隔开）


## 产品总览

| 产品           | 用途                                             | 参考文件                                                 |
|--------------|------------------------------------------------|----------------------------------------------------|
| `auth`       | 认证管理：登录/登出/刷新/状态查询（用户 API 命令的前置步骤）             | [auth.md](./references/products/auth.md)           |
| `contact`    | 通讯录：部门查询/成员管理/常用联系人                            | [contact.md](./references/products/contact.md)     |
| `ent`        | 企业管理：成员/部门/同步/库管理/文件操作（需企业 client_id）          | [ent.md](./references/products/ent.md)             |
| `file`       | 用户文件操作：列表/搜索/下载/外链/标签等（需 access_token）         | [file.md](./references/products/file.md)           |
| `library`    | 库管理：库信息/成员/部门（需 access_token）                   | [library.md](./references/products/library.md)     |
| `account`    | 账户管理：账户信息/设备/企业信息（需 access_token）               | [account.md](./references/products/account.md)     |
| `favorite`   | 收藏管理：收藏夹列表/添加/移除（需 access_token）                | [favorite.md](./references/products/favorite.md)   |

## 意图判断决策树

用户提到"通讯录/同事/部门/组织架构" → `contact`
用户提到"企业成员/同步/企业库管理"（企业管理 API）→ `ent`
用户提到"够快/云库/文件库/文件操作"（用户 API）→ `file` / `library` / `account`

> 更多易混淆场景见 [intent-guide.md](./references/intent-guide.md)

## 危险操作确认

以下操作为不可逆或高影响操作，执行前**必须先向用户展示操作摘要并获得明确同意**，同意后才加 `--yes` 执行。

（暂无高危命令）

### 确认流程
```
Step 1 → 展示操作摘要（操作类型 + 目标对象 + 影响范围）
Step 2 → 用户明确回复确认（如 "确认" / "好的"）
Step 3 → 加 --yes 执行命令
```

## 核心流程
作为一个智能助手，你的首要任务是**理解用户的真实、完整的意图**，而不是简单地执行命令。在选择 `ykc` 的产品命令前，必须严格遵循以下四步流程：

1. 意图分类：首先，判断用户指令的核心 动词/动作 属于哪一类。这比关注名词更重要。
2. 歧义处理与信息追问：如果用户指令模糊或包含多个产品的关键字，严禁猜测。必须主动向用户追问以澄清意图。这是你作为智能助手而非命令执行器的核心价值。
3. 精准产品映射：在完成前两步，意图已经清晰后，参考产品总览和意图判断决策树 来选择产品。
4. 充分阅读产品参考文件，通过编写代码或直接调用指令实现用户意图。

## 错误处理
1. 遇到错误，加 `--verbose` 重试一次
2. 若 stderr 出现 `RECOVERY_EVENT_ID=<event_id>`，优先按 [recovery-guide.md](./references/recovery-guide.md) 执行 recovery 闭环
3. 仍然失败，报告完整错误信息给用户，禁止自行尝试替代方案
4. 认证失败时，参考 [global-reference.md](./references/global-reference.md) 中的认证章节处理
5. 各产品高频错误及排查流程见 [error-codes.md](./references/error-codes.md)


## 详细参考 (按需读取)

- [references/products/](./references/products/) — 各产品命令详细参考
- [references/intent-guide.md](./references/intent-guide.md) — 意图路由指南（易混淆场景对照）
- [references/global-reference.md](./references/global-reference.md) — 全局标志、认证、输出格式
- [references/error-codes.md](./references/error-codes.md) — 错误码 + 调试流程
- [references/recovery-guide.md](./references/recovery-guide.md) — recovery 闭环、`RECOVERY_EVENT_ID`、`execute/finalize` 规范
- [scripts/](./scripts/) — 批量操作脚本（通讯录部门成员查询等）
