<h1 align="center">Yunku CLI (ykc)</h1>

<p align="center"><code>ykc</code> — Yunku CLI — built for humans and AI agents.</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-green?logo=go&logoColor=white" alt="Go 1.25+">
  <a href="https://github.com/gokuai/yunku-cli/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue" alt="License Apache-2.0"></a>
  <a href="https://github.com/gokuai/yunku-cli/releases"><img src="https://img.shields.io/github/v/release/gokuai/yunku-cli?color=red&label=release" alt="Latest Release"></a>

</p>

<p align="center">
  <a href="./README_zh.md">中文版</a> · <a href="./README.md">English</a> · <a href="./docs/reference.md">Reference</a> · <a href="./CHANGELOG.md">Changelog</a>
</p>


<details>
<summary><strong>Table of Contents</strong></summary>

- [Why ykc?](#why-ykc)
- [Installation](#installation)
- [Upgrade](#upgrade)
- [Quick Start](#quick-start)
- [Using with Agents](#using-with-agents)
- [Features](#features)
- [Key Services](#key-services)
- [Security by Design](#security-by-design)
- [Reference & Docs](#reference--docs)
- [Contributing](#contributing)

</details>


---

<h2 id="why-ykc">Why ykc?</h2>

- **For humans** — `--help` for usage, `-f table/json/raw` for output formats.
- **For AI agents** — structured JSON responses + built-in Agent Skills, ready out of the box.

## Installation

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.ps1 | iex
```

<details>
<summary>Other install methods</summary>

**npm** (requires Node.js (npm/npx)):

```bash
npm install -g yunku-cli
```

**Pre-built binary**: download from [GitHub Releases](https://github.com/gokuai/yunku-cli/releases).

> **macOS users**: If you see "cannot be opened because Apple cannot check it for malicious software", run:
> ```bash
> xattr -d com.apple.quarantine /path/to/ykc
> ```

**Build from source**:

```bash
git clone https://github.com/gokuai/yunku-cli.git
cd yunku-cli
go build -o ykc ./cmd       # build to current directory
cp ykc ~/.local/bin/         # install to PATH
```

> Requires Go 1.25+. Use `make package` to cross-compile for all platforms (macOS / Linux / Windows x amd64 / arm64).

</details>

## Upgrade

Re-run the install script to upgrade to the latest version:

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install.ps1 | iex
```

## Quick Start

```bash
ykc auth login                                                      # log in
ykc contact group search --ent-id <ent-id> --keyword "engineering"  # search departments
ykc contact member info --ent-id <ent-id> --member-id <id>          # get member info
```

## Using with Agents

`ykc` is designed as an AI-native CLI. Complete [Installation](#installation) and [Quick Start](#quick-start) first, then configure your agent:


### Agent Skills

The repo ships a complete Agent Skill system (`skills/`). After installing, AI tools like Claude Code / Cursor can operate yunku directly through natural language:

```bash
# Install skills into current project
curl -fsSL https://raw.githubusercontent.com/gokuai/yunku-cli/main/scripts/install-skills.sh | sh
```

> `install.sh` installs to `$HOME/.agents/skills/ykc` (global); `install-skills.sh` installs to `./.agents/skills/ykc` (current project). The skill folder uses the `ykc` name for consistent AI tool routing.

**What's included:**

| Component | Path | Description |
|-----------|------|-------------|
| Master Skill | `SKILL.md` | Intent routing, decision tree, safety rules, error handling |
| Product references | `references/products/*.md` | Per-product command reference |
| Intent guide | `references/intent-guide.md` | Disambiguation for confusing scenarios |
| Global reference | `references/global-reference.md` | Auth, output formats, global flags |
| Error codes | `references/error-codes.md` | Error codes + debugging workflows |
| Recovery guide | `references/recovery-guide.md` | `RECOVERY_EVENT_ID` handling |
| Ready-made scripts | `scripts/*.py` | 2 batch operation scripts (see below) |

<details>
<summary><strong>Ready-made scripts</strong> — 2 Python scripts for common multi-step workflows</summary>

| Script | Description |
|--------|-------------|
| `contact_dept_members.py` | Search department by name and list all members |
| `report_inbox_today.py` | Get today's received daily/weekly reports |

</details>

**ISV Integration**: Author your own Agent Skills and orchestrate them with ykc skills for cross-product workflows: **ISV Skill -> ykc Skill -> yunku Open Platform API (enforced auth + full audit)**.

## Features

<details>
<summary><strong>Smart Input Correction</strong> — auto-corrects common AI model parameter mistakes</summary>

Built-in pipeline engine that normalizes flag names, splits sticky arguments, and fuzzy-matches typos:

```bash
# Naming convention auto-conversion (camelCase / snake_case / UPPER -> kebab-case)
ykc contact group search --ent-id <id> --keyword "engineering" --limit100   # auto-corrected to --limit 100

# Sticky argument splitting
ykc contact group search --ent-id <id> --keyword "engineering" --timeout30  # auto-split to --timeout 30

# Value normalization (boolean / number / date / enum)
# "yes" -> true, "1,000" -> 1000, "2024/03/29" -> "2024-03-29", "ACTIVE" -> "active"
```

| Agent Output | ykc Auto-Corrects To |
|-----------|--------------|
| `--userId` | `--user-id` |
| `--limit100` | `--limit 100` |
| `--USER-ID` | `--user-id` |
| `--user_name` | `--user-name` |

</details>


## Key Services

| Service | Command | Subcommands | Description |
|---------|---------|-------------|-------------|
| Enterprise Management | `ent` | `member` `group` `org` `file` `oauth` `roles` `log` | Member/department management, enterprise library management, file operations, sync, authorization, roles, logs |
| Contacts | `contact` | `group` `member` `root-group` | Department management, member management, root group query |
| File Operations | `file` | `ls` `search` `create-file` `download` `link` `keyword` `add-lifecycle` ... | Library file management: list, search, upload/download, links, tags, lifecycle, etc. |
| Library Management | `library` | `create` `info` `list` `delete` `groups` `members` `add-member` `add-group` ... | Library create/info/list/delete, member and department management |
| Account | `account` | `info` `devices` `change-password` `ent-info` `mount` `toggle-device` ... | Account info, device management, password, enterprise info, etc. |
| Favorites | `favorite` | `list` `add` `remove` | Favorites list, add/remove files |

> Run `ykc --help` for the full list, or `ykc <command> --help` for subcommands.

<h2 id="security-by-design">Security by Design</h2>

`ykc` treats security as a first-class architectural concern, not an afterthought. **Credentials never touch disk, tokens never leave trusted domains, permissions never exceed grants, operations never escape audit** — every API call must pass through yunku Open Platform's authentication and audit chain, no exceptions.

<details>
<summary><strong>For Developers</strong></summary>

| Mechanism | Details |
|-----------|----------|
| **Encrypted token storage** | **PBKDF2 (600,000 iterations + SHA-256) + AES-256-GCM** encryption, keyed by device physical MAC address; macOS integrates system Keychain, Windows integrates DPAPI for additional protection — tokens cannot be decrypted on another machine |
| **Input security** | Path traversal protection (symlink resolution + working directory containment), CRLF injection blocking, Unicode visual spoofing filter — prevents AI Agents from being tricked by malicious instructions |
| **Data integrity** | All config writes use atomic operations (temp + fsync + rename), ensuring no data corruption on process interruption |
| **HTTPS enforced** | All requests require TLS; HTTP only permitted for loopback during development |
| **Dry-run preview** | `--dry-run` shows call parameters without executing, preventing accidental mutations |
| **Zero credential persistence** | Client ID / Secret used in memory only — never written to config files or logs |

</details>

<details>
<summary><strong>For Enterprise Admins</strong></summary>

| Mechanism | Details |
|-----------|---------|
| **Least-privilege scoping** | CLI can only invoke APIs granted to the application — no privilege escalation |
| **Allowlist gating** | Admin confirmation required during co-creation phase; self-service approval planned |
| **Full-chain audit** | Every data read/write passes through the yunku Open Platform API — enterprise admins can trace complete call logs in real time; no anomalous operation can hide |

</details>

<details>
<summary><strong>For ISVs</strong></summary>

| Mechanism | Details |
|-----------|---------|
| **Tenant data isolation** | Operates under authorized app identity; cross-tenant data is strictly isolated |
| **Skill sandbox** | Agent Skills are Markdown documents (`SKILL.md`) — prompt descriptions only, no arbitrary code execution |
| **Zero blind spots** | Every API call during ISV-ykc skill orchestration is forced through yunku Open Platform authentication — full call chain is traceable with no bypass path |

</details>

> Found a vulnerability? Report via [GitHub Security Advisories](https://github.com/gokuai/yunku-cli/security/advisories/new). See [SECURITY.md](./SECURITY.md).

## Reference & Docs

- [Reference](./docs/reference.md) — environment variables, exit codes, output formats, shell completion
- [Changelog](./CHANGELOG.md) — release history and migration notes

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for build instructions, testing, and development workflow.

## License

Apache-2.0
