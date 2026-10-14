# auth — 认证管理命令参考

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc auth` | 无需认证（本身用于管理认证） | — |

> `ykc auth` 是其他所有命令的前置步骤，用于登录并获取 access_token。

---

#### 用户名密码登录
```
Usage:
  ykc auth login [flags]
Examples:
  ykc auth login --browser
  ykc auth login -b
  ykc auth login --username user@example.com --password yourpassword
  ykc auth login -u user@example.com -p yourpassword
Flags:
  -b, --browser                    打开浏览器访问 passport OAuth2 授权页登录
      --oauth-client-id string     浏览器 OAuth2 授权专用 Client ID (仅 --browser，可用 GOKUAI_OAUTH_CLIENT_ID)
  -u, --username string            用户名 (密码登录时必需)
  -p, --password string            密码 (密码登录时必需)
      --client-id string           设备应用 Client ID (可使用环境变量 GOKUAI_CLIENT_ID)
      --client-secret string       设备应用 Client Secret (可使用环境变量 GOKUAI_CLIENT_SECRET)
```

#### 浏览器 OAuth2 授权登录（--browser）
- 执行 `ykc auth login --browser`（或 `-b`）后，CLI 会在本机 `127.0.0.1` 随机端口启动一次性回调服务，并自动打开浏览器跳转到 `GOKUAI_PASSPORT_DOMAIN/oauth/login`（默认 `https://passport.gokuai.com/oauth/login`）。
- **client_id 独立配置**：通过 `--oauth-client-id` 或环境变量 `GOKUAI_OAUTH_CLIENT_ID` 提供，与企业设备凭证（`GOKUAI_CLIENT_ID`）和个人 SDK 凭证（`GOKUAI_USER_CLIENT_ID`）相互独立；浏览器模式不调用 `get_client_oauth`，也不需要企业设备凭证。
- **两步换取正式 token**：浏览器回跳携带的是 passport 短时效网页 token（约 300 秒，非最终凭证）；CLI 立即调用 `/m-api/oauth2/token2`（`grant_type=exchange_token`、`auth=passport`）将其兑换成与密码登录等价的正式 token。兑换请求需用 `GOKUAI_USER_CLIENT_SECRET` 签名（需与所用 client_id 配对；未设置时回退到本地已保存的同 client 凭证）。
- 兑换成功后 token 与签名凭证一并落盘（与密码登录格式一致），后续命令无需再依赖环境变量。
- 无图形界面时可设置 `BROWSER` 环境变量自定义打开命令（如 `BROWSER=/bin/true`），CLI 会把完整授权地址打印到 stderr 供手动打开。
- 私有化/测试环境通过 `GOKUAI_PASSPORT_DOMAIN` 覆盖 passport 域名（与 `GOKUAI_API_HOST` 配合使用）。
- 服务端前提：`GOKUAI_OAUTH_CLIENT_ID` 对应的 oauth client 需开通 `authorization_code` 与 `exchange_token` 授权，并在 passport 的 `oauth_client_endpoints` 中为用户所属站点登记 host 为 `127.0.0.1` 的 redirect_uri（仅校验 host，不校验端口和路径）。
- 私有化/测试环境通过 `GOKUAI_PASSPORT_DOMAIN` 覆盖 passport 域名（与 `GOKUAI_API_HOST` 配合使用）。
- 服务端前提：`GOKUAI_OAUTH_CLIENT_ID` 对应的 oauth client 需开通 `authorization_code` 授权，并在 passport 的 `oauth_client_endpoints` 中为用户所属站点登记 host 为 `127.0.0.1` 的 redirect_uri（仅校验 host，不校验端口和路径）。

#### 清除认证信息
```
Usage:
  ykc auth logout [flags]
Example:
  ykc auth logout
Flags:
  (无额外参数)
```

#### 刷新 Access Token
```
Usage:
  ykc auth refresh [flags]
Example:
  ykc auth refresh
Flags:
      --refresh-token string   Refresh Token (可选，默认从本地登录存储读取)
```

#### 查看认证状态
```
Usage:
  ykc auth status [flags]
Example:
  ykc auth status
Flags:
      --token string   Access Token (默认读取本地登录凭证)
```

---

## 意图判断

- 说"登录"/"认证"/"login" → `auth login`
- 说"浏览器登录"/"网页登录"/"扫码或浏览器授权登录" → `auth login --browser`
- 说"登出"/"退出登录"/"logout" → `auth logout`
- 说"刷新token"/"refresh" → `auth refresh`
- 说"查看登录状态"/"认证状态" → `auth status`
