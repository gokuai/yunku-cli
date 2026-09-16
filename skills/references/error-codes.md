# 错误码说明
全产品错误参考 + 调试流程。Agent 遇到错误时查阅此文档。

## 错误返回格式

使用 `--format json` 时，错误统一输出为：

```json
{"error": {"category": "validation", "code": 3, "message": "ent-id 不能为空"}}
{"error": {"category": "auth", "code": 2, "message": "请先登录，运行 'ykc auth login'"}}
{"error": {"category": "api", "code": 1, "message": "error_code: 40401, error_msg: 库不存在或已删除", "reason": "error_code=40401", "retryable": true}}
```

- `code` 同时是进程退出码：1=API、2=Auth、3=Validation、5=Internal
- 服务端业务错误会在 `reason` 字段携带原始 `error_code=NNNNN`
- 可能附带 `hint`（修复提示）、`actions`（建议命令）字段

## 服务端错误码分类

够快 API 的 `error_code` 前三位对应 HTTP 状态类，CLI 按此自动分类：

| error_code 前缀 | 含义 | CLI 分类/退出码 | Agent 行为 |
|-----------------|------|----------------|-----------|
| 400xx | 请求参数不合法 | validation / 3 | 检查参数后修正重试 |
| 401xx | 未认证/token 无效 | auth / 2 | 执行 `ykc auth login` 或 `ykc auth refresh` |
| 403xx | 无权限/被禁止 | auth / 2 | 报告用户，不要自行重试 |
| 404xx / 405xx | 资源不存在/方法不支持 | api / 1 | 确认 ID 与路径后重试一次 |
| 5xxxx | 服务端错误 | api / 1 | 稍后重试；持续失败报告用户 |

常见错误码：`40003` 请求参数错误、`40101` token 无效、`40102` token 已过期、`40310` 没有操作权限、`40311` 没有权限访问该库、`40401` 库不存在或已删除、`40402` 文件(夹)不存在或已删除。完整表见 [docs/reference.md](../../docs/reference.md)。

## 通用错误
- 请求超时 — 网络慢或服务端响应慢 → `--timeout 60` 重试
- 网络连接失败 — 无法连接服务 → 用最简命令验证: `ykc auth status --format json`，再用 `ykc doctor` 检查网络
- stderr 出现 `RECOVERY_EVENT_ID=<event_id>` — runtime 失败已被 CLI 捕获 → 优先执行 `ykc recovery execute --event-id <event_id> --format json`

## Recovery 闭环

- `ykc recovery plan --last|--event-id <event_id> --format json`：读取失败快照并生成恢复计划
- `ykc recovery execute --last|--event-id <event_id> --format json`：生成带 `probe_results`、`agent_task` 的分析包（`doc_search` 字段当前恒为 `skipped`，未接入检索后端）
- `ykc recovery finalize --event-id <event_id> --outcome recovered|failed|handoff --execution-file execution.json --format json`：回写恢复结果

执行 `finalize` 时：

- `execution-file` 至少应包含 `actions`、`attempts`、`result`、`error_summary`
- 兼容旧格式：`action` + 数值型 `attempts`
- 若 recovery bundle 已进入 unknown/agent 路线，不要把 `human_actions` 视为最终结论；必须结合完整 bundle 判断

更多字段解释见 [recovery-guide.md](./recovery-guide.md)。

---

## contact 高频错误

- `contact group members` 报错 — `--group-id` 传了非整数值 → group-id 必须为整数，从 `contact group search` 获取

---

## 通用排查三步法

1. **确认 ID** — 从最顶层资源逐级获取，不猜 ID、不跳步
2. **确认参数** — flag 用 kebab-case；特殊字段查产品参考文档确认格式
3. **确认限制** — 检查批量上限和已知约束（各产品注意事项见对应产品参考文档）
