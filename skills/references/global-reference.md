# 全局参考

## 认证

```bash
# 用户名密码登录 (需要先设置 GOKUAI_CLIENT_ID / GOKUAI_CLIENT_SECRET 环境变量)
ykc auth login --username <email> --password <password>

# 查看状态
ykc auth status

# 刷新 Access Token
ykc auth refresh

# 退出 (清除本地 token 与用户凭证)
ykc auth logout
```

Token 加密存储在本地 (Keychain / PBKDF2+AES-256-GCM)，登录一次后 `ent`/`file` 等命令自动复用。

### Headless 环境 (CI/CD)

```bash
# 通过环境变量配置企业凭证
export GOKUAI_CLIENT_ID=<your-client-id>
export GOKUAI_CLIENT_SECRET=<your-client-secret>
ykc auth login --username <email> --password <password>

# 也可直接指定已获取的用户授权凭证，跳过设备应用换取步骤
export GOKUAI_USER_CLIENT_ID=<your-user-client-id>
export GOKUAI_USER_CLIENT_SECRET=<your-user-client-secret>
ykc auth login --username <email> --password <password>
```

## Recovery

当命令失败且 stderr 额外输出 `RECOVERY_EVENT_ID=<event_id>` 时，说明 CLI 已经持久化了失败快照，可进入 recovery 闭环：

```bash
ykc recovery plan --event-id <event_id> --format json
ykc recovery execute --event-id <event_id> --format json
ykc recovery finalize --event-id <event_id> --outcome recovered|failed|handoff --execution-file execution.json --format json
```

- `plan` / `execute` 也支持 `--last`，但 `--last` 与 `--event-id` 互斥
- recovery 文件保存在 `YKC_CONFIG_DIR/recovery/`
- CLI 会自动清理 30 天前的 recovery 文件和事件记录
- recovery 自己发起的文档检索与只读 probe 不会再创建新的 recovery 事件

更多闭环要求见 [recovery-guide.md](./recovery-guide.md)。


## 全局标志

| 标志 | 短名 | 说明 | 默认 |
|------|:---:|------|------|
| `--format` | `-f` | 输出格式: json / table / raw | json |
| `--jq` | | jq 表达式过滤输出 (如: `.items[] \| .name`) | 无 |
| `--fields` | | 筛选输出字段 (逗号分隔, 如: name,id,status) | 无 |
| `--verbose` | `-v` | 详细日志 | false |
| `--debug` | | 调试日志 | false |
| `--yes` | `-y` | 跳过确认提示 | false |
| `--dry-run` | | 预览操作不执行 | false |
| `--timeout` | | HTTP 超时 (秒) | 30 |
| `--bearer-token` | | Bearer Token 认证，用户 API 跳过签名 (可用环境变量 `GOKUAI_BEARER_TOKEN`) | 无 |
| `--mock` | | Mock 数据 (开发用) | false |

## 输出格式

### --format json (机器可读, 默认)

```json
{"success": true, "body": {...}}
```

### --format table (人类可读)

```
已查询通讯录用户 "张三" (userId: 123456)

下一步:
  ykc contact member info --ent-id <ent-id> --member-id 123456
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `YKC_CONFIG_DIR` | 覆盖默认配置目录 |
| `GOKUAI_CLIENT_ID` | 企业管理 API client_id (ent 命令、ykc auth login 需要) |
| `GOKUAI_CLIENT_SECRET` | 企业管理 API client_secret |
| `GOKUAI_ORG_CLIENT_ID` | 库授权 client_id (ent file 命令需要) |
| `GOKUAI_ORG_CLIENT_SECRET` | 库授权 client_secret |
| `GOKUAI_USER_CLIENT_ID` | 用户授权 client_id (跳过 ent oauth get，直接登录) |
| `GOKUAI_USER_CLIENT_SECRET` | 用户授权 client_secret |
| `GOKUAI_BEARER_TOKEN` | Bearer Token 认证，等同 `--bearer-token` (flag 优先) |
