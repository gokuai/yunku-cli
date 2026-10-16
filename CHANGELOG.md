# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by [Keep a Changelog](https://keepachangelog.com/) and this project follows [Semantic Versioning](https://semver.org/).

## [1.1.0] - 2026-10-16

### Added

- Browser OAuth2 login: `ykc auth login --browser` (`-b`) opens the passport `/oauth/login` authorization page and completes CLI login via a one-shot loopback (`127.0.0.1`) callback; the received tokens are stored in the same format as password login. Only needs a dedicated OAuth client id (`--oauth-client-id` or `GOKUAI_OAUTH_CLIENT_ID`, no secret — passport holds the secret server-side); post-login request signing continues to use `GOKUAI_USER_CLIENT_SECRET`. Honors the `BROWSER` environment variable for headless hosts
- New `ykc auth web` command: opens the web console via SSO using the current CLI login state, no separate web login required; target address configurable with `--url` (defaults to `GOKUAI_API_HOST`)
- New `GOKUAI_PASSPORT_DOMAIN` configuration item (default `https://passport.gokuai.com`) for the passport web-authorization domain, used by browser login and overridable for private deployments and test environments

## [1.0.1] - 2026-09-22

### Added

- `ykc auth login` now sends device information (OS name/version, etc.) with the login request

### Fixed

- `ykc account servers` failed to parse when no arguments were passed
- `ykc account upload-avatar` returned empty parameter values after a successful upload
- Favorite list pagination: `--size` minimum is 50, and paging should use `next_start` from the response; `ykc file queue` now requires `--queue-id`

## [1.0.0] - 2026-06-24

First public release of Yunku CLI (`ykc`).

### Core

- Direct Cobra command tree calling the GoKuai Cloud Library API (`pkg/gokuai`) — no runtime service discovery
- Username/password login (`ykc auth login`) against the GoKuai enterprise device application; tokens encrypted at rest (PBKDF2 + AES-256-GCM, with macOS Keychain / Windows DPAPI integration where available)
- Enterprise (`ent`) API signing via `client_id`+`client_secret` (HMAC-SHA1); library file operations use a separate library-scoped `org_client_id`/`org_client_secret`, obtainable either via automatic enterprise authorization or direct configuration
- Structured error model with documented exit codes (1=API, 2=Auth, 3=Validation, 4=Discovery, 5=Internal) and JSON error payloads (category, reason, hint, actions)
- Pipeline engine for pre-parse and post-parse input correction
  - `AliasHandler`: normalises flag casing (e.g. `--userId` → `--user-id`)
  - `StickyHandler`: splits glued flag values (e.g. `--limit100` → `--limit 100`)
  - `ParamNameHandler`: fixes near-miss flag typos (e.g. `--limt` → `--limit`)
  - `ParamValueHandler`: normalises structured parameter values
- Output filtering via `--fields` and `--jq` global flags
- `@file` / `@-` syntax for reading flag values from files or stdin
- Structured output formats: JSON, table, raw
- Global flags: `--format`, `--verbose`, `--debug`, `--dry-run`, `--yes`, `--timeout`
- Atomic file write helpers, path traversal protection, CRLF injection prevention
- Recovery closed-loop for capturing and replaying failed command executions

### Supported Services

- **ent** — 企业开放接口：成员/部门管理、企业库管理、文件操作、同步、授权、角色、日志
- **contact** — 部门管理、成员管理、根部门查询
- **file** — 库文件管理：列表、搜索、上传下载（分块上传）、复制移动、外链、标签、生命周期、权限等
- **library** — 库创建/信息/列表/删除、成员和部门管理
- **account** — 账户信息、设备管理、密码修改、企业信息等
- **favorite** — 收藏夹文件列表、添加/移除收藏

### Agent Skills

- Bundled `SKILL.md` with product reference docs, intent routing guide, error codes, and recovery guide
- One-line installer for macOS / Linux / Windows
- Skills installed to `$HOME/.agents/skills/ykc` (global) or `./.agents/skills/ykc` (project)

### Packaging

- Pre-built binaries for macOS (arm64/amd64), Linux (arm64/amd64), Windows (amd64)
- One-line install scripts (`install.sh`, `install.ps1`)
- Project-level skill installer (`install-skills.sh`)
- npm distribution (`npm install -g yunku-cli`)
- Shell completion: Bash, Zsh, Fish
