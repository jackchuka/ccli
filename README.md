# ccli

[![Test](https://github.com/jackchuka/ccli/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/ccli/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/jackchuka/ccli?sort=semver)](https://github.com/jackchuka/ccli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A unified CLI for inspecting your Claude Code installation — MCP servers, skills, rules, projects, and metadata.

## Why ccli?

Claude Code stores its configuration across many files and directories — global settings, project configs, MCP server definitions, skills, rules, session history, and more. There's no built-in way to get a unified view of what's configured, where it lives, or how it all fits together.

**ccli** gives you that visibility in a single command-line tool:

- **See everything at a glance** — `ccli info` shows your full setup: version, auth, model, paths, session counts, and storage usage.
- **Audit MCP servers** — List servers across all scopes, inspect their config, and verify environment variables without digging through JSON files.
- **Discover skills and rules** — Find what's available across personal, project, and plugin sources in one place.
- **Audit memory** — `ccli memory` shows which `CLAUDE.md`/`AGENTS.md` files and auto memory load into a session, in order, and what they cost in context.
- **Track project usage** — View per-project costs, token usage, and model breakdowns from session history.
- **Clean up old sessions** — Delete old session data and associated artifacts (debug logs, telemetry, todos, tasks) across all or specific projects.
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

### Memory

```bash
# Audit every memory file that loads, in load order
ccli memory list

# Include the files Claude Code loads on demand rather than at launch
ccli memory list --on-demand

# Details for one file, by scope name or path
ccli memory get user
ccli memory get CLAUDE.local.md
```

Shows managed policy, user, project, and local `CLAUDE.md` files, any
`AGENTS.md` in play, their `@path` imports nested underneath, and the auto
memory index, in the order Claude Code concatenates them — with line and byte
counts against the limits it documents. Warns about a `MEMORY.md` past its
200-line or 25KB load cutoff (either one drops the rest), a `CLAUDE.md` over
200 lines (which reduces adherence) or over 4 MiB (which Claude Code
**skips entirely**), broken or too-deep imports, imports that need external
approval, files shadowed by `claudeMdExcludes`, and auto memory disabled in
settings, which means `MEMORY.md` does not load at all. Each warning is
printed under the file it belongs to, and repeated in a summary at the end.

The launch total counts only what actually loads, so a file Claude Code
refuses — excluded, absent, past the four-hop import limit, a cycle, or auto
memory switched off — contributes nothing, while a file imported by two
parents counts twice because it is expanded twice. The on-demand figure is a
discovery count: those files never load at launch, so it reports how many are
there, and any that would not load when read say so.

The audit reflects Claude Code's documented resolution rules rather than
instrumenting a running session; `/context` remains the ground truth for what
one session actually loaded.

Claude Code reads `AGENTS.md` as project instructions when your repository has
no `CLAUDE.md`, `.claude/CLAUDE.md`, or `CLAUDE.local.md` in the working
directory or above it. Your personal `~/.claude/CLAUDE.md` and your
organization's managed `CLAUDE.md` don't count toward that check — only a
project-style `CLAUDE.md` in the working directory or an ancestor does,
which includes a bare `$HOME/CLAUDE.md` if your repository lives under your
home directory. `@path` imports and `claudeMdExcludes` apply to `AGENTS.md`
exactly as they do to `CLAUDE.md`.

In the launch tier, `ccli memory` follows the same rule and reports an
`AGENTS.md` a `CLAUDE.md` shadows so you can see it is doing nothing. A
subdirectory's `AGENTS.md`, which only appears under `--on-demand`, is
handled differently: when its own directory also has a `CLAUDE.md`, it is
left out of the listing entirely, with no row and no warning. The audit
names the active mode under the totals:

```
  loaded at launch:  2 files · 113L · 4.1 KB
  on demand:         0 files
  instruction files: claude-md-or-agents-md
```

The mode comes from `instructionFiles` under the built-in `agents-md` plugin
in your user or managed settings — Claude Code ignores it in project and local
settings files. `claude-md-and-agents-md` loads both, `claude-md` ignores
`AGENTS.md`, and `managed-only` loads just your organization's managed
`CLAUDE.md` and auto memory — and also excludes `.claude/rules/`, which this
audit warns about rather than enumerates; run `ccli rules` to see them. An
unrecognized `instructionFiles` value falls back to the default silently: the
totals show `claude-md-or-agents-md` with no indication that your configured
value didn't match one of the four modes. Reading `AGENTS.md` directly
requires Claude Code v2.1.277 or later, and is not available on Bedrock,
Vertex or Foundry.

### Projects

```bash
# List all known projects with usage stats
ccli projects list

# Show detailed stats for a project
ccli projects get my-project
```

Displays per-project cost, token usage (input/output), line changes, session count, and per-model cost breakdown.

### Cleaning up sessions

```bash
# Delete sessions older than 30 days across all projects
ccli projects clean --older-than 30d

# Preview what would be deleted
ccli projects clean --older-than 30d --dry-run

# Clean a specific project
ccli projects clean my-project --older-than 7d

# Retire a project entirely: every session, its memory, and its config entry
ccli projects clean my-project
```

Removes old session logs and associated artifacts (debug logs, telemetry, todos, tasks, file history, session environment) matched by session UUID.

Omitting `--older-than` for a named project retires that project: on top of every session it also deletes the project's `memory/` directory and removes the project from `~/.claude.json`. A `--older-than` sweep never touches memory, since memory is project-scoped rather than session-scoped.

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

| Path                                   | Content                              |
| -------------------------------------- | ------------------------------------ |
| `~/.claude.json`                       | Global MCP servers, project metadata |
| `~/.claude/settings.json`              | Model, plugins, permissions          |
| `~/.claude/history.jsonl`              | Session history metadata             |
| `.mcp.json`                            | Project-specific MCP servers         |
| `~/.claude/skills/`                    | Personal skills                      |
| `.claude/skills/`                      | Project-scoped skills                |
| `~/.claude/plugins/cache/`             | Plugin-provided skills               |
| `~/.claude/rules/`                     | Global rules                         |
| `.claude/rules/`                       | Project-scoped rules                 |
| `~/.claude/CLAUDE.md`                  | User memory instructions             |
| `./CLAUDE.md`, `./.claude/CLAUDE.md`   | Project memory instructions          |
| `./CLAUDE.local.md`                    | Local (gitignored) memory            |
| `./AGENTS.md`, `./.claude/AGENTS.md`   | Project instructions if no CLAUDE.md |
| Managed policy `CLAUDE.md`             | Organization-wide instructions       |
| `~/.claude/projects/<project>/memory/` | Auto memory index and topic files    |
| `~/.claude/projects/`                  | Project session data                 |
| `~/.claude/debug/`                     | Debug logs (per session)             |
| `~/.claude/telemetry/`                 | Telemetry events (per session)       |
| `~/.claude/todos/`                     | Agent todo tracking (per session)    |
| `~/.claude/tasks/`                     | Task records (per session)           |
| `~/.claude/file-history/`              | File edit history (per session)      |
| `~/.claude/session-env/`               | Session environment (per session)    |

All resources are categorized by scope — global, project, personal, or plugin — shown with colored bullets in text output.

No network calls. No dependency on the `claude` binary (except for version and auth status).

## License

MIT
