<h1 align="center">Yunku CLI (ykc)</h1>

<p align="center"><code>ykc</code> — yunku工作台命令行工具，为人类和 AI Agent 而生。</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-green?logo=go&logoColor=white" alt="Go 1.25+">
  <a href="https://github.com/gokuai/yunku-cli/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue" alt="License Apache-2.0"></a>
  <a href="https://github.com/gokuai/yunku-cli/releases"><img src="https://img.shields.io/github/v/release/gokuai/yunku-cli?color=red&label=release" alt="Latest Release"></a>

</p>

<p align="center">
  <a href="./README_zh.md">中文版</a> · <a href="./README.md">English</a> · <a href="./docs/reference.md">参考手册</a> · <a href="./CHANGELOG.md">更新日志</a>
</p>


<details>
<summary><strong>目录</strong></summary>

- [为什么选择 ykc？](#why-ykc)
- [安装](#安装)
- [升级](#升级)
- [快速开始](#快速开始)
- [在 Agent 中使用](#在-agent-中使用)
- [功能特性](#功能特性)
- [核心服务](#核心服务)
- [安全设计](#安全设计)
- [参考与文档](#参考与文档)
- [贡献指南](#贡献指南)

</details>


---

<h2 id="why-ykc">为什么选择 ykc？</h2>

- **为人类而设计** — `--help` 查看用法，`-f table/json/raw` 切换格式。
- **为 AI Agent 而设计** — 结构化 JSON 响应 + 内置 Agent Skills，开箱即用。

## 安装

**macOS / Linux：**

```bash
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.sh | sh
```

**Windows（PowerShell）：**

```powershell
irm https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.ps1 | iex
```

<details>
<summary>其他安装方式</summary>

**npm**（需要 Node.js（npm/npx））：

```bash
npm install -g yunku-cli
```

**预编译二进制文件**：从 [GitHub Releases](https://github.com/gokuai/yunku-cli/releases) 下载。

> **macOS 用户注意**：如果提示“无法打开，因为 Apple 无法检查其是否包含恶意软件”，请执行：
> ```bash
> xattr -d com.apple.quarantine /path/to/ykc
> ```

**从源码构建**：

```bash
git clone https://github.com/gokuai/yunku-cli.git
cd yunku-cli
go build -o ykc ./cmd       # 编译到当前目录
cp ykc ~/.local/bin/         # 安装到 PATH
```

> 需要 Go 1.25+。也可以用 `make package` 构建所有平台产物（macOS / Linux / Windows × amd64 / arm64）。

</details>

## 升级

重新执行安装脚本即可升级到最新版本：

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.sh | sh

# Windows（PowerShell）
irm https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.ps1 | iex
```

## 快速开始

```bash
ykc auth login                                             # 登录（无浏览器环境如 Docker、SSH、CI 也支持）
ykc contact group search --ent-id <ent-id> --keyword "技术部"  # 搜索部门
ykc contact member info --ent-id <ent-id> --member-id <id>    # 查询成员
```

## 在 Agent 中使用

`ykc` 是为 AI Agent 设计的 CLI 工具。请先完成[安装](#安装)和[快速开始](#快速开始)，然后配置 Agent 环境：


### Agent Skills

仓库内置完整的 Agent Skill 体系（`skills/`），安装后 Claude Code / Cursor 等 AI 工具可通过自然语言直接操作yunku：

```bash
# 安装 skills 到当前项目
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install-skills.sh | sh
```

> `install.sh` 安装到 `$HOME/.agents/skills/ykc`（全局）；`install-skills.sh` 安装到 `./.agents/skills/ykc`（当前项目）。技能文件夹沿用 `ykc` 名称以保持 AI 工具路由一致性。

**包含内容：**

| 组件 | 路径 | 说明 |
|------|------|------|
| 主 Skill | `SKILL.md` | 意图路由、决策树、安全规则、错误处理 |
| 产品参考 | `references/products/*.md` | 各产品命令详细参考 |
| 意图指南 | `references/intent-guide.md` | 易混淆场景消歧 |
| 全局参考 | `references/global-reference.md` | 认证、输出格式、全局 flag |
| 错误码 | `references/error-codes.md` | 错误码 + 调试流程 |
| Recovery 指南 | `references/recovery-guide.md` | `RECOVERY_EVENT_ID` 处理 |
| 现成脚本 | `scripts/*.py` | 2 个批量操作脚本（见下方） |

<details>
<summary><strong>现成脚本</strong> — 2 个 Python 脚本，覆盖常见多步工作流</summary>

| 脚本 | 说明 |
|------|------|
| `contact_dept_members.py` | 按部门名称搜索并列出所有成员 |
| `report_inbox_today.py` | 获取今日收到的日报周报 |

</details>

**ISV 集成**：编写您自己的 Agent Skill，与内置 Skill 搭配构建跨产品工作流：**ISV Skill → ykc Skill → yunku开放平台 API（强制鉴权 + 全链路审计）**。

## 功能特性

<details>
<summary><strong>智能输入纠错</strong> — 自动修正 AI 模型常见的参数错误</summary>

内置 Pipeline 纠错引擎，支持命名风格转换、粘连参数拆分、拼写模糊匹配：

```bash
# 命名风格自动转换 (camelCase / snake_case / UPPER → kebab-case)
ykc contact group search --ent-id <id> --keyword "技术部" --limit100   # 自动纠正为 --limit 100

# 粘连参数自动拆分
ykc contact group search --ent-id <id> --keyword "技术部" --timeout30  # 自动拆分为 --timeout 30

# 参数值归一化 (布尔 / 数字 / 日期 / 枚举)
# "yes" → true, "1,000" → 1000, "2024/03/29" → "2024-03-29", "ACTIVE" → "active"
```

| Agent 输出 | ykc 自动纠正为 |
|-----------|--------------|
| `--userId` | `--user-id` |
| `--limit100` | `--limit 100` |
| `--USER-ID` | `--user-id` |
| `--user_name` | `--user-name` |

</details>


## 核心服务

| 服务 | 命令 | 子命令 | 描述 |
|------|------|--------|------|
| 企业管理 | `ent` | `member` `group` `org` `file` `oauth` `roles` `log` | 成员/部门管理、企业库管理、文件操作、同步、授权、角色、日志 |
| 通讯录 | `contact` | `group` `member` `root-group` | 部门管理、成员管理、根部门查询 |
| 文件操作 | `file` | `ls` `search` `create-file` `download` `link` `keyword` `add-lifecycle` ... | 库文件管理：列表、搜索、上传下载、外链、标签、生命周期等 |
| 库管理 | `library` | `create` `info` `list` `delete` `groups` `members` `add-member` `add-group` ... | 库创建/信息/列表/删除、成员和部门管理 |
| 账户管理 | `account` | `info` `devices` `change-password` `ent-info` `mount` `toggle-device` ... | 账户信息、设备管理、密码修改、企业信息等 |
| 收藏管理 | `favorite` | `list` `add` `remove` | 收藏夹文件列表、添加/移除收藏 |

> 运行 `ykc --help` 查看完整列表，或 `ykc <command> --help` 查看子命令。

## 安全设计

`ykc` 从架构层面将安全作为一等公民，而非事后补丁。**凭证不落盘、Token 不出域、权限不越界、操作不脱审** — 每一次 API 调用都必须经过yunku开放平台的鉴权和审计链路，无例外。

<details>
<summary><strong>开发者安全机制</strong></summary>

| 机制 | 说明 |
|------|------|
| **Token 加密存储** | **PBKDF2（600,000 次迭代 + SHA-256）+ AES-256-GCM** 加密，密钥绑定设备物理 MAC 地址；macOS 集成系统 Keychain、Windows 集成 DPAPI 提供额外保护，跨设备无法解密 |
| **输入安全防护** | 路径遍历防护（符号链接解析 + 工作目录约束）、CRLF 注入拦截、Unicode 视觉欺骗字符过滤，防止 AI Agent 被恶意指令诱导 |
| **数据完整性** | 所有配置写入采用原子操作（temp + fsync + rename），确保进程中断时数据不损坏 |
| **HTTPS 强制** | 除 loopback 开发调试外，所有请求强制 TLS |
| **Dry-run 预览** | `--dry-run` 展示调用参数但不执行，防止误操作生产数据 |
| **凭证零落盘** | Client ID / Secret 仅在内存中使用，不写入配置文件或日志 |

</details>

<details>
<summary><strong>企业管理员安全机制</strong></summary>

| 机制 | 说明 |
|------|------|
| **权限最小化** | CLI 仅能调用管理员授予该应用的 API 权限范围，无法越权 |
| **白名单准入** | 共创阶段需管理员主动确认开通，后续支持自助审批 |
| **操作全链路审计** | 每一次数据读写都经过yunku开放平台 API，企业管理员可在管理后台实时追溯完整调用日志，任何异常操作无处隐藏 |

</details>

<details>
<summary><strong>ISV / 企业服务商安全机制</strong></summary>

| 机制 | 说明 |
|------|------|
| **租户数据隔离** | 以已授权应用身份调用 API，不同租户数据严格隔离 |
| **Skill 沙箱** | Agent Skills 是 Markdown 文档（`SKILL.md`），仅提供 prompt 描述，不执行任意代码 |
| **集成链路零盲区** | ISV Skill 与 ykc Skill 联调时，每一次 API 调用都强制经过yunku开放平台鉴权，完整调用链路可追溯，不存在绕过审计的旁路 |

</details>

> 发现安全漏洞？请通过 [GitHub Security Advisories](https://github.com/gokuai/yunku-cli/security/advisories/new) 报告，详见 [SECURITY.md](./SECURITY.md)。

## 参考与文档

- [参考手册](./docs/reference.md) — 环境变量、退出码、输出格式、Shell 补全
- [更新日志](./CHANGELOG.md) — 版本历史与迁移说明

## 贡献指南

参见 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解构建、测试和开发工作流。

## 许可证

Apache-2.0
