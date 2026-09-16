# account — 账户管理命令参考（用户 API）

## 认证说明

| 命令组 | 认证方式 | 环境变量 |
|--------|----------|----------|
| `ykc account` | 用户 access_token | `ykc auth login` 登录后自动管理 |

> 需要先 `ykc auth login` 登录

---

#### 获取账户信息
```
Usage:
  ykc account info [flags]
Example:
  ykc account info
```

#### 获取企业信息
```
Usage:
  ykc account ent-info [flags]
Example:
  ykc account ent-info --ent-id 100
Flags:
  --ent-id string   企业ID (必需)
```

#### 更新企业信息
```
Usage:
  ykc account update-ent [flags]
Example:
  ykc account update-ent --ent-id 100 --name "新企业名称"
Flags:
  --ent-id string   企业ID (必需)
  --name string     企业名称
  --logo string     企业Logo
```

#### 获取文件库列表
```
Usage:
  ykc account mount [flags]
Example:
  ykc account mount
```

#### 获取设备列表
```
Usage:
  ykc account devices [flags]
Example:
  ykc account devices
```

#### 删除设备
```
Usage:
  ykc account delete-device [flags]
Example:
  ykc account delete-device --device-id "xxx"
Flags:
  --device-id string   设备ID (必需)
```

#### 启用/禁用设备
```
Usage:
  ykc account toggle-device [flags]

Examples:
  ykc account toggle-device --device-id xxx --state 1
  ykc account toggle-device --device-id xxx --state 0

Flags:
      --device-id string   设备ID (必需)
  -h, --help               help for toggle-device
      --state string       状态: 1启用/0禁用 (必需)
```

#### 启用/禁用"禁用新设备登录"功能
```
Usage:
  ykc account disable-new-device [flags]

Examples:
  # 启用"禁用新设备登录"功能
  ykc account disable-new-device --state 1

  # 禁用"禁用新设备登录"功能
  ykc account disable-new-device --state 0

Flags:
  --state string   状态: 1(启用"禁用新设备"功能) / 0(禁用该功能) (必需)
```

#### 修改密码
```
Usage:
  ykc account change-password [flags]

Examples:
  ykc account change-password --old-password "OldPass" --new-password "NewPass"

  # 手动指定 refresh_token（通常登录后自动读取，无需手动传入）
  ykc account change-password --old-password "OldPass" --new-password "NewPass" --refresh-token "xxx"

Flags:
  --old-password string    旧密码 (必需，自动 MD5 加密后发送)
  --new-password string    新密码 (必需，明文发送)
  --refresh-token string   Refresh Token (可选，优先从 flag > 本地登录存储 自动获取)
```

#### 找回密码
```
Usage:
  ykc account find-password [flags]
Example:
  ykc account find-password --email zhangsan@company.com
Flags:
  --email string   邮箱地址 (必需)
```

#### 绑定邮箱
```
Usage:
  ykc account bind-mail [flags]
Example:
  ykc account bind-mail --email zhangsan@company.com
Flags:
  --email string   邮箱地址 (必需)
```

#### 重新发送验证邮件
```
Usage:
  ykc account resend-mail [flags]
Example:
  ykc account resend-mail --email zhangsan@company.com
Flags:
  --email string   邮箱地址 (必需)
```

#### 上传头像
```
Usage:
  ykc account upload-avatar [flags]

Examples:
  # 上传个人头像
  ykc account upload-avatar --file /path/to/avatar.jpg --type avatar

  # 上传企业标识
  ykc account upload-avatar --file /path/to/logo.png --type ent

  # 上传库头像
  ykc account upload-avatar --file /path/to/avatar.jpg --type org

  # 上传发布页logo
  ykc account upload-avatar --file /path/to/logo.png --type link

Flags:
  --file string   头像文件路径 (必需)
  --type string   上传头像类型: ent(企业标识)/org(库头像)/avatar(个人头像)/link(发布页logo) (必需)
```

#### 获取设置信息
```
Usage:
  ykc account settings [flags]
Example:
  ykc account settings
```

#### 获取软件信息
```
Usage:
  ykc account software [flags]
Example:
  ykc account software
```

#### 获取服务器列表
```
Usage:
  ykc account servers [flags]

Examples:
  ykc account servers --server-type upload

Flags:
  -h, --help                   help for servers
      --server-type string     服务器类型: upload/download
      --storage-point string   存储点
```

---

## 意图判断

- 说"账户信息/设备" → `account info/devices`

## 上下文传递表

| 操作 | 提取字段 | 用于 |
|------|---------|------|
| `account mount` | `mount_id` | `file` 命令的 `--mount-id` |
