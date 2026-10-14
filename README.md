<div align="center">

# ccli

**See everything your Claude Code installation is configured with.**

MCP servers, skills, rules, memory files, and per-project usage,<br>
read straight from disk and shown in one place.

[![Test](https://github.com/jackchuka/ccli/actions/workflows/test.yml/badge.svg)](https://github.com/jackchuka/ccli/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/jackchuka/ccli?sort=semver)](https://github.com/jackchuka/ccli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<img src="assets/demo.gif" alt="ccli showing an installation summary, MCP servers, skills, memory files, and project usage" width="800">

[Installation](#installation) · [Quick Start](#quick-start) · [Commands](#commands) · [Memory Audit](#memory-audit) · [How It Works](#how-it-works)

</div>

---

Claude Code spreads its configuration across `~/.claude.json`, `~/.claude/`, `.mcp.json`, project `.claude/` directories, and a managed policy location. Answering "which MCP servers are active here?" or "which `CLAUDE.md` files actually load in this repository?" means opening several JSON files and knowing the resolution rules.

**ccli** reads all of it and answers in one command. It works offline and never modifies anything, except for `projects clean`, which deletes old session data when you ask it to.

## Features

- **Installation summary**: `ccli info` shows version, auth, model, paths, session and project counts, storage use, and resource counts
- **MCP servers**: list servers across global and project scopes and inspect their command, URL, and environment, with secrets masked in every output format
- **Skills and rules**: discover personal, project, and plugin skills, and global and project rules
- **Memory audit**: see which `CLAUDE.md`, `AGENTS.md`, `@path` imports, and auto memory files load into a session, in order, with their line and byte cost and warnings for files that won't load
- **Project usage**: last-session cost, tokens, line changes, and per-model cost for every known project
- **Session cleanup**: delete old session logs and related artifacts, or retire a project entirely, with a `--dry-run` preview
- **Scriptable**: every command supports `--format json` and `--format yaml`

## Installation

### Homebrew

```bash
brew install jackchuka/tap/ccli
```

### Go

```bash
go install github.com/jackchuka/ccli@latest
```

### Binary

Download a prebuilt binary for Linux, macOS, or Windows from the [releases page](https://github.com/jackchuka/ccli/releases).

## Quick Start

Run these from inside a project so that project-scoped config is picked up:

```bash
ccli info                                     # installation summary
ccli mcp list                                 # MCP servers across scopes
ccli memory list                              # what loads into a session, in order
ccli projects list                            # per-project cost and sessions
ccli projects clean --older-than 30d --dry-run # preview a cleanup
```

## Commands

| Command                            | Description                                                                 |
| ---------------------------------- | --------------------------------------------------------------------------- |
| `ccli info`                        | Installation summary                                                        |
| `ccli mcp list`                    | List MCP servers from `~/.claude.json` (global) and `.mcp.json` (project)   |
| `ccli mcp get <name>`              | Server type, command, URL, and environment (secrets masked)                 |
| `ccli skills list`                 | List personal, project, and plugin skills                                   |
| `ccli skills get <name>`           | Skill path and description                                                  |
| `ccli rules list`                  | List global and project rule files                                          |
| `ccli rules get <name>`            | Rule details; the name includes the extension, e.g. `testing.md`            |
| `ccli memory list [--on-demand]`   | Memory files in load order with their context cost                          |
| `ccli memory get <name>`           | One memory file, by scope name (`user`, `project`, ...) or path             |
| `ccli projects list`               | Known projects with cost, sessions, and line changes                        |
| `ccli projects get <name-or-path>` | Project details and per-model cost                                          |
| `ccli projects clean [project]`    | Delete old session data (see [Cleaning up sessions](#cleaning-up-sessions)) |
| `ccli version`                     | Print the version                                                           |

### Global flags

| Flag           | Description                                             |
| -------------- | ------------------------------------------------------- |
| `-f, --format` | Output format: `text`, `json`, `yaml` (default: `text`) |
| `--no-color`   | Disable colored output                                  |

### Cleaning up sessions

```bash
# Delete sessions older than 30 days across all projects
ccli projects clean --older-than 30d

# Preview first
ccli projects clean --older-than 30d --dry-run

# Only one project
ccli projects clean my-project --older-than 7d

# Retire a project: every session, its memory, and its config entry
ccli projects clean my-project
```

`--older-than` takes a whole number of days (`7d`, `30d`, `90d`) and is required unless you name a project. Cleaning removes session logs and the artifacts matched by session UUID: debug logs, telemetry, todos, tasks, file history, and session environment.

Naming a project without `--older-than` retires it: in addition to every session, it deletes the project's `memory/` directory and removes the project from `~/.claude.json`. An `--older-than` sweep never touches memory, because memory belongs to the project rather than to a session.

## Memory Audit

```bash
ccli memory list               # files loaded at launch
ccli memory list --on-demand   # also files loaded later, when Claude opens a file there
ccli memory get user           # one file, by scope name
ccli memory get CLAUDE.local.md # or by path
```

`ccli memory list` shows the managed policy, user, project, and local `CLAUDE.md` files, every `AGENTS.md` it finds, their `@path` imports nested underneath, and the auto memory index, in the order Claude Code concatenates them, with line and byte counts. Under the list it prints totals and the active instruction mode:

```
  loaded at launch:  5 files · 17L · 315 B
  on demand:         1 file
  instruction files: claude-md-or-agents-md
```

It warns about:

- a `MEMORY.md` past its 200-line or 25 KB load cutoff (everything after it is dropped)
- a `CLAUDE.md` over 200 lines (reduces adherence) or over 4 MiB (skipped entirely)
- broken imports, imports deeper than four hops, cycles, and imports that need external approval
- files excluded by `claudeMdExcludes`
- auto memory disabled in settings, which means `MEMORY.md` does not load at all

Each warning appears under its file and again in a summary at the end.

The launch total counts only what actually loads. Excluded, missing, too-deep, cyclic, or mode-skipped files add nothing; a file imported by two parents counts twice because it is expanded twice. The on-demand figure is a count of files found, not a load cost.

The audit follows Claude Code's documented resolution rules rather than instrumenting a running session. `/context` in a live session remains the ground truth.

### AGENTS.md and instruction modes

By default, Claude Code reads `AGENTS.md` only when there is no project-style `CLAUDE.md` (`CLAUDE.md`, `.claude/CLAUDE.md`, or `CLAUDE.local.md`) in the working directory or any ancestor, including a bare `$HOME/CLAUDE.md`. Your `~/.claude/CLAUDE.md` and the managed `CLAUDE.md` don't count. If such a file exists, no `AGENTS.md` is read, not even in subdirectories. Otherwise a subdirectory's `AGENTS.md` loads on demand when that directory has no `CLAUDE.md` of its own. `@path` imports and `claudeMdExcludes` apply to `AGENTS.md` as they do to `CLAUDE.md`.

ccli lists every `AGENTS.md` it finds and marks the ones that won't be read, with the reason (for example, the `CLAUDE.md` that shadows it). Marked files and their imports are left out of the totals.

The mode comes from `pluginConfigs["agents-md@builtin"].options.instructionFiles` in your user or managed settings; Claude Code ignores it in project and local settings.

| Mode                               | Behavior                                                                                                                                                                                            |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `claude-md-or-agents-md` (default) | `AGENTS.md` only when no project `CLAUDE.md` exists, as described above                                                                                                                             |
| `claude-md-and-agents-md`          | Both; an `AGENTS.md` already pulled in through an `@path` import is not read twice                                                                                                                  |
| `claude-md`                        | Never reads `AGENTS.md`                                                                                                                                                                             |
| `managed-only`                     | Only the managed `CLAUDE.md` and auto memory at launch; subdirectory `CLAUDE.md` still loads on demand. Also excludes `.claude/rules/` (ccli warns rather than listing them; run `ccli rules list`) |

An unrecognized value falls back to the default silently, so the totals show `claude-md-or-agents-md` even if you configured something else. Reading `AGENTS.md` directly requires Claude Code v2.1.277 or later and is not available on Bedrock, Vertex, or Foundry.

## How It Works

ccli reads these files directly. It makes no network calls and only invokes the `claude` binary for `ccli info`, to get the version and auth status.

| Path                                                                                  | Content                                                                                                                                         |
| ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `~/.claude.json`                                                                      | Global MCP servers, project metadata                                                                                                            |
| `~/.claude/settings.json`                                                             | Model, plugins, memory settings                                                                                                                 |
| `.claude/settings.json`, `.claude/settings.local.json`                                | Project and local memory settings                                                                                                               |
| `~/.claude/history.jsonl`                                                             | Session history                                                                                                                                 |
| `.mcp.json`                                                                           | Project MCP servers                                                                                                                             |
| `~/.claude/skills/`, `.claude/skills/`                                                | Personal and project skills                                                                                                                     |
| `~/.claude/plugins/cache/`                                                            | Plugin skills                                                                                                                                   |
| `~/.claude/rules/`, `.claude/rules/`                                                  | Global and project rules                                                                                                                        |
| `~/.claude/CLAUDE.md`                                                                 | User memory                                                                                                                                     |
| `CLAUDE.md`, `.claude/CLAUDE.md`                                                      | Project memory (working directory and ancestors)                                                                                                |
| `CLAUDE.local.md`                                                                     | Local memory                                                                                                                                    |
| `AGENTS.md`, `.claude/AGENTS.md`                                                      | Project instructions when no `CLAUDE.md` applies                                                                                                |
| Managed `CLAUDE.md`, `managed-settings.json`                                          | Organization policy (`/Library/Application Support/ClaudeCode` on macOS, `/etc/claude-code` on Linux, `C:\Program Files\ClaudeCode` on Windows) |
| `~/.claude/projects/<project>/memory/`                                                | Auto memory index and topic files                                                                                                               |
| `~/.claude/projects/`                                                                 | Session logs                                                                                                                                    |
| `~/.claude/debug/`, `telemetry/`, `todos/`, `tasks/`, `file-history/`, `session-env/` | Per-session artifacts removed by `projects clean`                                                                                               |

MCP server environment values are masked in text, JSON, and YAML output when the variable name contains `KEY`, `SECRET`, `TOKEN`, `PASSWORD`, `PASSWD`, `CREDENTIAL`, or `PRIVATE`, or when the value is a URL with a password in it (`postgres://user:pass@host/db`). Plain URLs such as `https://api.example.com` are shown as is.

A session counts once per session UUID, whether it exists as a `<uuid>.jsonl` log, a `<uuid>/` directory, or both. Other entries in a project directory, such as `memory/`, are not counted.

Text output marks each resource's scope (global, project, personal, local, plugin, auto) with a colored bullet.

ccli is built around an agent interface, so support for other coding agents' configuration can be added alongside Claude Code.

## License

[MIT](LICENSE)
