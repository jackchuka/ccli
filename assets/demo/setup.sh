#!/usr/bin/env bash
# Builds a fabricated Claude Code installation under $1 for the demo GIF.
set -euo pipefail

D="$1"
H="$D/home"
P="$H/code/acme-api"
C="$H/.claude"

rm -rf "$D"
mkdir -p "$D/bin" "$P/.claude/skills/release-notes" "$C/skills" "$C/rules" "$C/projects"

cat >"$D/bin/claude" <<'SH'
#!/bin/sh
case "$1" in
  --version) echo "2.1.0 (Claude Code)" ;;
  auth) echo '{"loggedIn":true,"authMethod":"claude.ai"}' ;;
esac
SH
chmod +x "$D/bin/claude"

cat >"$H/.claude.json" <<JSON
{
  "numStartups": 128,
  "mcpServers": {
    "github": { "type": "http", "url": "https://api.example.com/mcp/github" },
    "filesystem": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "~/code"] },
    "postgres": {
      "command": "npx",
      "args": ["-y", "mcp-postgres"],
      "env": { "DATABASE_URL": "postgres://localhost:5432/app", "PGPASSWORD": "not-a-real-secret" }
    }
  },
  "projects": {
    "$H/code/acme-api": {
      "hasTrustDialogAccepted": true, "lastCost": 4.82,
      "lastLinesAdded": 412, "lastLinesRemoved": 97,
      "lastTotalInputTokens": 182340, "lastTotalOutputTokens": 24510,
      "lastModelUsage": {
        "claude-sonnet-4-5": { "inputTokens": 150000, "outputTokens": 20000, "costUSD": 3.95 },
        "claude-haiku-4-5": { "inputTokens": 32340, "outputTokens": 4510, "costUSD": 0.87 }
      }
    },
    "$H/code/web-app": {
      "hasTrustDialogAccepted": true, "lastCost": 2.17,
      "lastLinesAdded": 158, "lastLinesRemoved": 41,
      "lastTotalInputTokens": 90211, "lastTotalOutputTokens": 11024
    },
    "$H/code/dotfiles": {
      "hasTrustDialogAccepted": true, "lastCost": 0.36,
      "lastLinesAdded": 12, "lastLinesRemoved": 3,
      "lastTotalInputTokens": 15002, "lastTotalOutputTokens": 1820
    }
  }
}
JSON

cat >"$P/.mcp.json" <<'JSON'
{ "mcpServers": { "sentry": { "type": "http", "url": "https://mcp.example.com/sentry" } } }
JSON

cat >"$C/settings.json" <<'JSON'
{ "model": "sonnet", "enabledPlugins": { "code-review@example-plugins": true } }
JSON

skill() {
  mkdir -p "$1/$2"
  printf -- '---\nname: %s\ndescription: %s\n---\n\n# %s\n' "$2" "$3" "$2" >"$1/$2/SKILL.md"
}
skill "$C/skills" commit-message "Write a Conventional Commits message for the staged diff"
skill "$C/skills" pr-summary "Summarize a pull request for reviewers"
skill "$C/skills" test-plan "Draft a test plan from a feature spec"
skill "$P/.claude/skills" release-notes "Turn merged PRs into release notes"
skill "$C/plugins/cache/example-plugins/code-review/1.0.0/skills" code-review "Review the current diff for bugs"

printf '# Comments\n\nOnly comment on why, not what.\n' >"$C/rules/code-comments.md"
printf '# Tests\n\nEvery bug fix ships with a regression test.\n' >"$C/rules/testing.md"

printf '# Preferences\n\n- Prefer small, focused commits\n- Use the latest stable toolchain\n\n@~/.claude/style.md\n' >"$C/CLAUDE.md"
printf '# Style\n\n- Keep functions short\n- Name things for what they do\n' >"$C/style.md"
printf '# acme-api\n\nGo service. Run `make test` before committing.\n' >"$P/CLAUDE.md"
printf 'Use the local database on port 5432.\n' >"$P/CLAUDE.local.md"

session() {
  local dir="$C/projects/$1" id="$2" stamp="$3"
  mkdir -p "$dir"
  head -c "$4" /dev/zero | tr '\0' 'x' >"$dir/$id.jsonl"
  touch -t "$stamp" "$dir/$id.jsonl"
}
enc() { printf '%s' "$1" | sed 's/[^A-Za-z0-9]/-/g'; }
A=$(enc "$H/code/acme-api"); W=$(enc "$H/code/web-app"); F=$(enc "$H/code/dotfiles")
OLD=$(date -v-60d +%Y%m%d0900 2>/dev/null || date -d '60 days ago' +%Y%m%d0900)
NEW=$(date +%Y%m%d0900)
session "$A" 1b6f2a10-0000-4000-8000-000000000001 "$OLD" 184320
session "$A" 1b6f2a10-0000-4000-8000-000000000002 "$OLD" 96512
session "$A" 1b6f2a10-0000-4000-8000-000000000003 "$NEW" 51200
session "$W" 2c7a3b20-0000-4000-8000-000000000004 "$OLD" 230400
session "$W" 2c7a3b20-0000-4000-8000-000000000005 "$NEW" 40960
session "$F" 3d8b4c30-0000-4000-8000-000000000006 "$NEW" 20480
mkdir -p "$C/projects/$A/memory"
printf '# Memory\n\n- [Build](build.md): make targets and flags\n' >"$C/projects/$A/memory/MEMORY.md"
printf 'Run `make lint` before pushing.\n' >"$C/projects/$A/memory/build.md"
for id in 1b6f2a10-0000-4000-8000-000000000001 2c7a3b20-0000-4000-8000-000000000004; do
  mkdir -p "$C/debug" "$C/todos"
  head -c 40960 /dev/zero >"$C/debug/$id.txt"
  echo '[]' >"$C/todos/$id-agent-$id.json"
done
printf '{"display":"fix flaky test"}\n{"display":"add healthcheck"}\n' >"$C/history.jsonl"
