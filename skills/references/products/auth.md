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
  ykc auth login --username user@example.com --password yourpassword
  ykc auth login -u user@example.com -p yourpassword
Flags:
  -u, --username string          用户名 (必需)
  -p, --password string          密码 (必需)
      --client-id string         设备应用 Client ID (可使用环境变量 GOKUAI_CLIENT_ID)
      --client-secret string     设备应用 Client Secret (可使用环境变量 GOKUAI_CLIENT_SECRET)
```

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
- 说"登出"/"退出登录"/"logout" → `auth logout`
- 说"刷新token"/"refresh" → `auth refresh`
- 说"查看登录状态"/"认证状态" → `auth status`
