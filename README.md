# ccli

A unified CLI for inspecting your Claude Code installation — MCP servers, skills, rules, projects, and metadata.

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
