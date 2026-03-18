---
name: clawback
description: "Manage OpenClaw config using clawback — a modular JSON5 config tool. Use when rendering openclaw.json from JSON5 source files, checking if openclaw.json has drifted from sources, syncing openclaw changes back to source files, editing specific config sections (channels, plugins, agents, bindings, etc.), or setting up clawback on a new gateway. Binary at ~/bin/clawback, config at ~/.openclaw/.clawback.json5, sources at ~/.openclaw/config/*.json5."
---

# clawback

Treats `openclaw.json` as a build artifact rendered from modular JSON5 source files in `~/.openclaw/config/`.

## Setup (already done on this gateway)

- Binary: `~/bin/clawback`
- Config: `~/.openclaw/.clawback.json5`
- Sources: `~/.openclaw/config/*.json5`
- Master template: `~/.openclaw/config/openclaw.json5`

## Commands

```bash
# Check if openclaw.json has drifted from sources
~/bin/clawback diff

# Exit code only (0=clean, 1=drifted) — good for CI/heartbeat checks
~/bin/clawback diff --quiet

# Render sources → openclaw.json
~/bin/clawback render

# Backport openclaw.json changes → source files, then re-render
~/bin/clawback sync
```

## Config Structure

| File | Owns |
|------|------|
| `config/env.json5` | API keys, tokens |
| `config/auth.json5` | Auth profiles |
| `config/agents.json5` | Agent definitions, models, runtime |
| `config/channels.json5` | Discord, Slack, etc. |
| `config/acp.json5` | ACP runtime config |
| `config/plugins.json5` | Plugin entries and config |
| `config/hooks.json5` | Internal hooks |
| `config/tools.json5` | Tool policies, exec, web |
| `config/gateway.json5` | Gateway port, auth, TLS |
| `config/skills.json5` | Skill install config |
| `config/messages.json5` | TTS, formatting |
| `config/commands.json5` | Slash command config |
| `config/openclaw.json5` | Master template (inline `bindings` array here) |

## Workflow

**To edit config:**
1. Edit the relevant section file in `~/.openclaw/config/`
2. Run `~/bin/clawback render`
3. Restart gateway if needed: `openclaw gateway restart`

**After OpenClaw auto-modifies openclaw.json** (e.g. plugin install):
1. Run `~/bin/clawback sync` to backport changes to source files
2. Verify with `~/bin/clawback diff`

**To check config health:**
```bash
~/bin/clawback diff --quiet && echo "clean" || echo "drifted — run sync"
```

## Known Limitations

- `$include` only supports object-valued files — arrays must be inline in the master template
- `bindings` is an array, so it lives inline in `config/openclaw.json5` (not a separate file)
- Key ordering in rendered `openclaw.json` is alphabetical (differs from original order, but functionally identical)

## Rebuilding the Binary

If the binary is missing or needs updating:
```bash
cd /tmp && rm -rf clawback
GH_TOKEN=$(cat ~/.openclaw/credentials/github) gh repo clone jeff-french/clawback
cd clawback && /usr/local/go/bin/go build -o ~/bin/clawback .
```
