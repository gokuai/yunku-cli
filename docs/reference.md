# Reference / 参考手册

## Environment Variables / 环境变量

| Variable | Purpose / 用途 |
|---------|---------|
| `YKC_CONFIG_DIR` | Override default config directory / 覆盖默认配置目录 |
| `GOKUAI_CLIENT_ID` | Enterprise (`ent`) API client_id, used with `GOKUAI_CLIENT_SECRET` for HMAC-SHA1 signing / 企业管理 API client_id，与 `GOKUAI_CLIENT_SECRET` 配合做签名认证 |
| `GOKUAI_CLIENT_SECRET` | Enterprise (`ent`) API client_secret / 企业管理 API client_secret |
| `GOKUAI_ORG_CLIENT_ID` | Library authorization client_id for `ent file` commands / `ent file` 库文件操作命令所需的库授权 client_id |
| `GOKUAI_ORG_CLIENT_SECRET` | Library authorization client_secret for `ent file` commands / `ent file` 库文件操作命令所需的库授权 client_secret |
| `GOKUAI_BEARER_TOKEN` | Bearer token auth, same as the global `--bearer-token` flag (flag wins) / Bearer Token 认证，等同全局参数 `--bearer-token`（flag 优先） |

## Exit Codes / 退出码

ykc 将够快开放平台 API 返回的 `error_code`（完整表见 [错误码文档](https://developer.goukuai.cn/overview/errorcode.html)）按其 HTTP 状态码前缀归类到以下退出码；CLI 自身的参数/flag 校验错误遵循同一约定。

ykc classifies the GoKuai Open Platform API `error_code` (full table at the [official error code reference](https://developer.goukuai.cn/overview/errorcode.html)) into the exit codes below, based on its HTTP-status-class prefix. The CLI's own flag/argument validation errors follow the same convention.

| Code | Category | 对应 HTTP 状态码<br>HTTP Status | Description / 描述 |
|------|----------|:---:|-------------|
| 0 | Success | - | Command completed successfully / 命令执行成功 |
| 1 | API | 404 / 405 / 5xx | Upstream API failure: resource not found, method not allowed, server error / 上游 API 失败：资源不存在、方法不支持、服务端错误等 |
| 2 | Auth | 401 / 403 | Authentication or authorization failure: invalid/expired token, no permission / 身份认证失败或无权限访问 |
| 3 | Validation | 400 | Invalid or missing input/flags/parameters / 请求参数或 CLI flag 校验失败 |
| 4 | Discovery | - | Reserved, currently unused / 保留字段，当前未使用 |
| 5 | Internal | - | Unexpected internal error (local I/O, encoding, etc.) / 未预期的内部错误（本地 I/O、编码等） |

With `-f json`, error responses include structured payloads: `category`, `reason` (carries the original `error_code`, e.g. `error_code=40101`), `hint`, `actions`.

使用 `-f json` 时，错误响应包含结构化字段：`category`、`reason`（携带原始 `error_code`，如 `error_code=40101`）、`hint`、`actions`。

### GoKuai Open Platform Error Codes / 够快开放平台错误码

服务端出错时 HTTP 状态码 ≥ 400，响应体为 `{"error_code": <code>, "error_msg": "<message>"}`。完整错误码表（更新于 2025-10-30）见：https://developer.goukuai.cn/overview/errorcode.html

When the API call fails, the HTTP status is ≥ 400 and the body is `{"error_code": <code>, "error_msg": "<message>"}`. See the full table (last updated 2025-10-30) at the link above.

以下按 HTTP 状态码分类节选常见错误码，完整列表以官方文档为准：

**HTTP 400 — 请求数据不合法 / Bad Request → exit 3 (Validation)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 400 | 请求数据不合法 |
| 40001 | 请求地址错误 |
| 400012 | 签名验证错误 |
| 40002 | 请求方法错误 |
| 40003 | 请求参数错误 |
| 400031 | 邮箱不能为空 |
| 400032 | 邮箱格式不正确 |
| 400036 | 手机号不能为空 |
| 4000325 | 输入的密码错误 |
| 4000329 | 验证码错误 |
| 40013 | 无法将文件夹移动到它的子文件夹中 |
| 40017 | 该文件不支持在线预览 |
| 40051 | 该存储点不存在 |
| 40093 | 已存在相同邮箱的成员 |

**HTTP 401 — 未通过身份验证 / Unauthorized → exit 2 (Auth)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 401 | 没有进行身份验证 |
| 40101 | token 无效 |
| 401010 | refresh_token 无效 |
| 40102 | token 已经过期 |
| 40103 | client 不存在 |
| 40104 | 签名错误 |
| 40106 | 登录超时 |

**HTTP 403 — 无权限访问 / Forbidden → exit 2 (Auth)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 403 | 没有权限访问对应的资源 |
| 40302 | 该帐号已被禁止登录 |
| 40303 | 帐号或密码不正确 |
| 40306 | 该功能未开启 |
| 40310 | 没有操作权限 |
| 40311 | 没有权限访问该库 |
| 40321 | 库空间不足，请清理回收站或删除不需要的文件 |
| 40332 | 企业的成员个数已超出限制 |

**HTTP 404 — 资源不存在 / Not Found → exit 1 (API)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 404 | 请求的资源不存在 |
| 40401 | 库不存在或已删除 |
| 40402 | 文件(夹)不存在或已删除 |
| 40404 | 库不存在 |
| 40406 | 用户不存在 |
| 40407 | 部门不存在 |
| 40410 | 企业不存在 |
| 40420 | 帐号不存在 |

**HTTP 405 — 方法不支持 / Method Not Allowed → exit 1 (API)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 405 | Method Not Allowed |
| 40501 | 请求的方法未授权 |

**HTTP ≥ 500 — 服务端错误 / Server Error → exit 1 (API)**

| error_code | 说明 / Description |
|-----------:|---------------------|
| 500 | 服务器内部错误 |
| 50001 | 数据库出错 |
| 502 | 接口 API 关闭或正在升级 |
| 503 | 服务端资源不可用 |
| 50301 | 无法获取到上传服务器 |

## Output Formats / 输出格式

```bash
ykc contact group search --ent-id <ent-id> --keyword "Alice" -f table   # Table (default, human-friendly / 表格，默认)
ykc contact group search --ent-id <ent-id> --keyword "Alice" -f json    # JSON (for agents and piping / 适合 agent)
ykc contact group search --ent-id <ent-id> --keyword "Alice" -f raw     # Raw API response / 原始响应
```

## Output to File / 输出到文件

```bash
ykc contact group search --ent-id <ent-id> --keyword "Alice" -o result.json
```

## Shell Completion / 自动补全

```bash
# Bash
ykc completion bash > /etc/bash_completion.d/ykc

# Zsh
ykc completion zsh > "${fpath[1]}/_ykc"

# Fish
ykc completion fish > ~/.config/fish/completions/ykc.fish
```
