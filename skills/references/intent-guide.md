# 意图路由指南

当用户请求难以判断归属哪个产品时，参考本指南。

本 CLI 有两套并行的 API 体系，这是最主要的混淆来源：

| 体系 | 认证方式 | 命令组 |
|------|----------|--------|
| 用户 API | `ykc auth login` 登录获取 access_token | `contact` / `file` / `library` / `account` / `favorite` |
| 企业管理 API | 环境变量 `GOKUAI_CLIENT_ID`/`GOKUAI_CLIENT_SECRET` 签名 | `ent`（含 `ent file`，需库授权） |

## 易混淆场景快速对照表

| 用户说... | 真实意图 | 应该用 | 不要用 | 理由 |
|-----------|----------|--------|--------|------|
| "查一下技术部有哪些人" | 以登录用户身份查通讯录 | `contact group members` | `ent member` | 通讯录查询走用户 API |
| "给企业添加一个成员" | 企业级成员管理 | `ent member add` | `contact` | 增删成员是企业管理操作，contact 只读 |
| "列出我的文件" | 操作登录用户可见的库文件 | `file ls` | `ent file ls` | 用户已登录时优先用户 API |
| "用企业凭证批量操作某个库的文件" | 无用户登录态的库级操作 | `ent file` | `file` | ent file 走库授权（org_client），适合服务端脚本 |
| "看看我加入了哪些库" | 用户视角的库列表 | `account mount` / `library` | `ent org` | 用户 API 反映登录者可见范围 |
| "创建/修改一个企业库" | 企业级库管理 | `ent org create` / `ent org set` | `library` | 库的增删改属企业管理 API |
| "同步组织架构到云库" | 企业目录同步 | `ent sync` | `contact` | 同步是企业管理专属能力 |

## 判断步骤

1. **看操作对象归属**：操作"我的/我可见的"资源 → 用户 API；操作"企业的/任意库的"资源 → `ent`
2. **看认证条件**：用户未登录但有 `GOKUAI_CLIENT_ID`/`GOKUAI_CLIENT_SECRET` → 只能走 `ent`；反之只能走用户 API
3. **看读写性质**：通讯录(contact)只提供查询；成员/部门的增删改一律在 `ent member` / `ent group`
4. `ent file` 需要库授权凭证（`--org-id`/`--mount-id` 自动换取，或直接配置 `GOKUAI_ORG_CLIENT_ID`/`GOKUAI_ORG_CLIENT_SECRET`），细节见 [ent.md](./products/ent.md)

仍无法判断时，向用户确认"是以你的账号身份操作，还是用企业凭证操作"。
