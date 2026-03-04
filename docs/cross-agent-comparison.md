# Cross-Agent Comparison

Research on what configuration files and metadata AI coding agents store locally, and what's useful to expose via `ccli`.

## Config File Locations

| Agent | Global Config Directory | Primary Config File | Format | Project-Level Config |
|---|---|---|---|---|
| **Claude Code** | `~/.claude/` | `~/.claude.json` | JSON | `.claude/`, `.mcp.json` |
| **Codex CLI** | `~/.codex/` | `~/.codex/config.toml` | TOML | `.codex/config.toml` |
| **Gemini CLI** | `~/.gemini/` | `~/.gemini/settings.json` | JSON | `.gemini/settings.json` |
| **Cursor** | `~/Library/Application Support/Cursor/User/` | `globalStorage/state.vscdb` (SQLite) | SQLite + JSON | `.cursor/mcp.json`, `.cursor/rules/*.mdc` |
| **GitHub Copilot CLI** | `~/.copilot/` | `~/.copilot/config.json` | JSON | `.copilot/settings.json`, `.github/agents/` |
| **GitHub Copilot VS Code** | `~/.config/github-copilot/` | `apps.json`, `versions.json` | JSON + SQLite | `.github/copilot-instructions.md`, `.vscode/mcp.json` |
| **Windsurf** | `~/.codeium/windsurf/` | `mcp_config.json` | JSON | `.windsurf/rules/` |
| **Aider** | `~/.aider.conf.yml` (file) | `~/.aider.conf.yml` | YAML | `.aider.conf.yml` in repo root |
| **Amazon Q** | `~/.aws/amazonq/` | `mcp.json` + `q settings` | JSON | `.amazonq/mcp.json`, `.amazonq/rules/` |

## MCP Servers

| Agent | Where Stored | Format | Notes |
|---|---|---|---|
| **Claude Code** | `~/.claude.json` (root `mcpServers`), `.mcp.json` | JSON | Project also in per-project config |
| **Codex CLI** | `config.toml` `[mcp_servers.*]` | TOML | Includes timeout, enabled flag, bearer token env var |
| **Gemini CLI** | `settings.json` `mcpServers` | JSON | Separate `mcp-server-enablement.json` for on/off state |
| **Cursor** | `~/.cursor/mcp.json`, `.cursor/mcp.json` | JSON | Also configurable via Settings UI |
| **Copilot CLI** | `~/.copilot/mcp-config.json` | JSON | Accepts both Claude-style and `mcpServers`-wrapped formats |
| **Copilot VS Code** | `~/.config/github-copilot/intellij/mcp.json`, `.vscode/mcp.json` | JSON | Per-IDE subdirectories |
| **Windsurf** | `~/.codeium/windsurf/mcp_config.json` | JSON | 100-tool limit across all servers |
| **Aider** | N/A | -- | No native MCP support |
| **Amazon Q** | `~/.aws/amazonq/mcp.json` | JSON | Global and workspace configs merged at runtime |

## Models / Plugins / Extensions

| Agent | Model Config | Plugin/Extension Info |
|---|---|---|
| **Claude Code** | `settings.json` `model` | `~/.claude/plugins/installed_plugins.json`; skills in `~/.claude/skills/` |
| **Codex CLI** | `config.toml` `model`; `models_cache.json` | Skills in `~/.codex/skills/`; no plugin system |
| **Gemini CLI** | `settings.json` `model.name` | Extensions via `tools.discoveryCommand`; skills in `antigravity/skills/` |
| **Cursor** | `state.vscdb` (SQLite) | VS Code extensions in app support dir |
| **Copilot CLI** | Default: Claude Sonnet 4.5; `/model` to switch | `~/.copilot/plugins/`, `~/.copilot/agents/`, `~/.copilot/skills/` |
| **Copilot VS Code** | Extension settings | `versions.json` lists installed integrations |
| **Windsurf** | Cascade settings | VS Code-compatible extension marketplace |
| **Aider** | `.aider.conf.yml` `model`; `--list-models` flag | `.aider.model.settings.yml` for custom model definitions |
| **Amazon Q** | `chat.defaultModel` setting | Built-in tools with per-tool config; no extension marketplace |

## Session History

| Agent | Storage Location | Format | Resumable |
|---|---|---|---|
| **Claude Code** | `~/.claude/history.jsonl`, `~/.claude/projects/{id}/*.jsonl` | JSONL | Yes: `claude --resume` |
| **Codex CLI** | `~/.codex/state_5.sqlite`, `sessions/YYYY/` | SQLite + dirs | Yes: `codex resume` |
| **Gemini CLI** | Internal checkpointing | Opaque | Yes: checkpoint system |
| **Cursor** | `workspaceStorage/*/state.vscdb` | SQLite | Within editor only |
| **Copilot CLI** | `~/.copilot/session-state/<uuid>/events.jsonl` | JSONL + YAML + checkpoints | Yes: `--resume` |
| **Copilot VS Code** | `~/.config/github-copilot/go/chat-sessions/` | Per-session dirs | No |
| **Windsurf** | Cascade memories (internal) | Opaque | Via auto-generated memories |
| **Aider** | `.aider.chat.history.md` in working dir | Markdown | Yes: `--restore-chat-history` |
| **Amazon Q** | `~/.aws/amazonq/history/` | Directory | Yes: `--resume` |

## Rules / Custom Instructions

| Agent | Global | Project | Format | Target Globs |
|---|---|---|---|---|
| **Claude Code** | `~/.claude/rules/` | `.claude/rules/` | Markdown with YAML frontmatter | `paths:` in frontmatter |
| **Codex CLI** | `~/.codex/rules/default.rules` | `.codex/` project overrides | Plain text | No |
| **Gemini CLI** | `~/.gemini/GEMINI.md` | `.gemini/GEMINI.md` (walks up to .git) | Markdown with `@import` | No |
| **Cursor** | Cursor Settings > Rules (in `state.vscdb`) | `.cursor/rules/*.mdc` | MDC (YAML header + markdown) | `globs:` in header |
| **Copilot CLI** | N/A | `.github/copilot-instructions.md` | Markdown | No |
| **Copilot VS Code** | `~/.config/github-copilot/intellij/global-copilot-instructions.md` | `.github/copilot-instructions.md` | Markdown | No |
| **Windsurf** | `~/.codeium/windsurf/memories/global_rules.md` | `.windsurf/rules/` | Markdown | Glob-attached rules |
| **Aider** | Config keys (conventions, commit-prompt) | `.aider.conf.yml` | YAML config keys | No |
| **Amazon Q** | `~/.aws/amazonq/profiles/` | `.amazonq/rules/` | Markdown / JSON | No |

## Permissions / Trust

| Agent | Where Stored | What It Tracks |
|---|---|---|
| **Claude Code** | `settings.json` `permissions.allow` | Auto-approved tools/commands |
| **Codex CLI** | `config.toml` `[projects.*]` `trust_level` | Per-project trust, sandbox mode, approval policy |
| **Gemini CLI** | `settings.json` `security.folderTrust.enabled` | Folder trust, auth type enforcement |
| **Cursor** | `~/.cursor/cli-config.json` `permissions.allow/deny` | Operation allowlists/denylists |
| **Copilot CLI** | Cached path approvals | Per-path filesystem access, `--allow-all` flag |
| **Windsurf** | Cascade settings | Per-tool toggles |
| **Aider** | No explicit permission model | `.aiderignore` for file exclusion |
| **Amazon Q** | `toolsSettings`; `--trust-all-tools` | Per-tool trust, path allowlists/denylists |

## Existing CLI Inspection Capabilities

| Agent | Command | What It Shows |
|---|---|---|
| **Claude Code** | `claude mcp list` | MCP servers (no skills, no unified view) |
| **Codex CLI** | `codex mcp list --json` | MCP servers |
| | `codex features list` | Feature flags with maturity stage |
| | `codex login status` | Auth mode, logged-in status |
| | `codex execpolicy check` | Command allow/prompt/block status |
| | `codex resume` / `codex cloud list` | Session and task listing |
| **Gemini CLI** | `/memory show` | Loaded context/instruction files |
| | `/memory refresh` | Reload context |
| **Cursor** | (none) | Must query `state.vscdb` with sqlite3 |
| **Copilot CLI** | `/model` | List/switch models |
| | `--resume` (with picker) | Browse previous sessions |
| **Windsurf** | (none) | Must use MCP panel or Settings UI |
| **Aider** | `/settings` | Current config values in-chat |
| | `--list-models` | Print known models |
| | `--show-prompts` | Print system prompts sent to LLM |
| | `--show-repo-map` | Print repo map |
| **Amazon Q** | `q settings list --all` | All settings with descriptions |
| | `q settings list --format json-pretty` | Export settings as JSON |

## What's Readable by a Third-Party CLI

**Plain text (easy):**
- MCP server configs (all except Aider)
- Rules/instructions files
- Model selection defaults
- Project trust levels (Codex, Gemini)
- Config file settings

**Requires SQLite:**
- Cursor settings and chat history (`state.vscdb`)
- Copilot IntelliJ data (`copilot-intellij.db`)
- Codex session transcripts (`state_5.sqlite`)

**Opaque / internal only:**
- Windsurf auto-generated memories
- Gemini checkpointing state
- Cursor global user rules (locked in `state.vscdb`)

## Universal Concepts for ccli

These concepts exist across all agents and map to `ccli` commands:

| Concept | ccli Command | Agents with Data |
|---|---|---|
| MCP servers | `ccli mcp list/get` | All except Aider |
| Skills/plugins | `ccli skills list/get` | Claude, Codex, Copilot |
| Projects | `ccli projects list/get` | Claude, Codex, Gemini |
| Rules | `ccli rules list/get` | Claude, Codex, Gemini, Cursor, Windsurf, Amazon Q |
| Info/version | `ccli info` | All |
| Model | Part of `ccli info` | All |
| Permissions | Future | Claude, Codex, Cursor, Copilot, Amazon Q |
| Session history | Future (v2) | Claude, Codex, Copilot, Aider, Amazon Q |
