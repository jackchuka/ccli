# ccli

A unified CLI for inspecting your Claude Code installation — MCP servers, skills, rules, projects, and metadata.

## Why ccli?

Claude Code stores its configuration across many files and directories — global settings, project configs, MCP server definitions, skills, rules, session history, and more. There's no built-in way to get a unified view of what's configured, where it lives, or how it all fits together.

**ccli** gives you that visibility in a single command-line tool:

- **See everything at a glance** — `ccli info` shows your full setup: version, auth, model, paths, session counts, and storage usage.
- **Audit MCP servers** — List servers across all scopes, inspect their config, and verify environment variables without digging through JSON files.
- **Discover skills and rules** — Find what's available across personal, project, and plugin sources in one place.
- **Track project usage** — View per-project costs, token usage, and model breakdowns from session history.
- **Scriptable output** — Every command supports `--format json` and `--format yaml` for automation and piping.

All of this works offline by reading config files directly — no network calls, no dependency on the `claude` binary (except for version/auth detection).

Looking ahead, ccli is designed to grow beyond Claude Code. As the ecosystem of AI coding agents expands — Cursor, Windsurf, Codex, and others — each brings its own configuration formats, MCP setups, and project conventions. ccli aims to become a single pane of glass for inspecting and managing configuration across multiple agents, so you can understand your full AI-assisted development setup regardless of which tools you use.

## Installation

### Homebrew

```bash
brew install jackchuka/tap/ccli
```

### Go

```bash
go install github.com/jackchuka/ccli@latest
```

## Usage

### Info dashboard

```bash
ccli info
```

Shows version, auth status, model, paths, session/project counts, storage size, and resource counts.

### MCP servers

```bash
# List all MCP servers across all scopes
ccli mcp list

# Show details for a specific server
ccli mcp get datadog
```

Displays server type, command, URL, and environment variables (sensitive values are masked).

### Skills

```bash
# List all skills grouped by scope
ccli skills list

# Show details for a specific skill
ccli skills get brainstorming
```

Discovers skills from personal (`~/.claude/skills/`), project (`.claude/skills/`), and plugin (`~/.claude/plugins/cache/`) sources.

### Rules

```bash
# List all rules from global and project scopes
ccli rules list

# Show details for a specific rule
ccli rules get code-comments
```

### Projects

```bash
# List all known projects with usage stats
ccli projects list

# Show detailed stats for a project
ccli projects get my-project
```

Displays per-project cost, token usage (input/output), line changes, session count, and per-model cost breakdown.

### Output formats

All commands support `--format` for machine-readable output:

```bash
ccli mcp list --format json
ccli skills list --format yaml
ccli info --format json
```

### Flags

| Flag           | Description                                             |
| -------------- | ------------------------------------------------------- |
| `-f, --format` | Output format: `text`, `json`, `yaml` (default: `text`) |
| `--no-color`   | Disable colored output                                  |

## How it works

ccli reads Claude Code configuration files directly from disk:

| Path                       | Content                              |
| -------------------------- | ------------------------------------ |
| `~/.claude.json`           | Global MCP servers, project metadata |
| `~/.claude/settings.json`  | Model, plugins, permissions          |
| `~/.claude/history.jsonl`  | Session history metadata             |
| `.mcp.json`                | Project-specific MCP servers         |
| `~/.claude/skills/`        | Personal skills                      |
| `.claude/skills/`          | Project-scoped skills                |
| `~/.claude/plugins/cache/` | Plugin-provided skills               |
| `~/.claude/rules/`         | Global rules                         |
| `.claude/rules/`           | Project-scoped rules                 |
| `~/.claude/projects/`      | Project session data                 |

All resources are categorized by scope — global, project, personal, or plugin — shown with colored bullets in text output.

No network calls. No dependency on the `claude` binary (except for version and auth status).

## License

MIT
