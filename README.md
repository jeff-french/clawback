# clawback

A CLI tool to manage modular OpenClaw configuration using JSON5 `$include` directives.

## Problem

`openclaw plugins install` (and other commands) resolve and flatten all `$include` directives in `openclaw.json`, converting a modular config into a monolithic file. See [openclaw#41050](https://github.com/openclaw/openclaw/issues/41050).

## Solution

Treat `openclaw.json` as a **build artifact** rendered from a master `config/openclaw.json5` template that `$include`s individual section files. This tool:

1. **Renders** the master template → `openclaw.json`
2. **Detects** when `openclaw.json` has been clobbered (drifted from template)
3. **Backports** new/changed values to the correct JSON5 source files
4. **Preserves** comments and formatting in JSON5 files

## Usage

```bash
# Initialize modular config from an existing openclaw.json
clawback init

# Render openclaw.json from config/openclaw.json5
clawback render

# Show diffs between current openclaw.json and what render would produce
clawback diff

# Output diff as JSON
clawback diff --json

# Backport changes from openclaw.json to JSON5 sources, then re-render
clawback sync

# Preview what sync would change without modifying files
clawback sync --dry-run

# Check exit code only (for CI/heartbeat)
# Exit codes: 0 = clean, 1 = drifted, 2 = error
clawback diff --quiet
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--home` | OpenClaw home directory (default: `~/.openclaw`) |

## Configuration

Place `.clawback.json5` in your `~/.openclaw/` directory:

```json5
{
  configDir: "./config",
  outputFile: "./openclaw.json",
  masterTemplate: "./config/openclaw.json5",
  passthrough: [
    "meta",
    "wizard",
    "plugins.installs",
  ],
}
```

## Directory Structure

```
~/.openclaw/
├── openclaw.json           ← BUILD ARTIFACT (do not edit directly)
├── .clawback.json5         ← clawback config
└── config/
    ├── openclaw.json5      ← master template (edit this + section files)
    ├── env.json5           ← API keys, tokens
    ├── auth.json5          ← auth profiles
    ├── agents.json5        ← agent definitions, models, runtime
    ├── channels.json5      ← Discord, Slack, etc.
    ├── acp.json5           ← ACP runtime config
    ├── plugins.json5       ← plugin entries and config
    ├── hooks.json5         ← internal hooks
    ├── tools.json5         ← tool policies, exec, web
    ├── gateway.json5       ← gateway port, auth, TLS
    ├── skills.json5        ← skill install config
    ├── messages.json5      ← TTS, formatting
    └── commands.json5      ← slash command config
```

> **Note:** `bindings` is an array and must be inlined in `config/openclaw.json5` — `$include` only supports object-valued files.

## Typical Workflow

**Editing config:**
```bash
# Edit the relevant section file
vim ~/.openclaw/config/channels.json5

# Render to openclaw.json
clawback render

# Restart gateway if needed
openclaw gateway restart
```

**After OpenClaw auto-modifies openclaw.json** (e.g. after `openclaw plugins install`):
```bash
# Backport changes to source files, then re-render
clawback sync

# Verify clean
clawback diff
```

**CI / heartbeat drift check:**
```bash
clawback diff --quiet
status=$?
if [ $status -eq 0 ]; then echo "clean"
elif [ $status -eq 1 ]; then echo "drifted — run sync"
else echo "error"; exit $status
fi
```

## Known Limitations

- `$include` only supports object-valued files. Array-typed top-level keys (e.g. `bindings`) must be inlined in the master template.
- Key ordering in rendered `openclaw.json` is alphabetical, which differs from the original. This is cosmetic — OpenClaw doesn't care about key order.

## OpenClaw Skill

An [OpenClaw skill](skills/clawback/SKILL.md) is included in this repo. Install it to give your agent procedural knowledge of the clawback workflow:

```bash
cp -r skills/clawback ~/.openclaw/skills/
```

Once installed, your agent will automatically use clawback when editing config sections instead of modifying `openclaw.json` directly.

## Quick Start — Let Your Agent Set It Up

Give your OpenClaw agent this prompt to install clawback, modularize your config, install the skill, and set up a drift-check heartbeat:

<details>
<summary>Copy this prompt</summary>

```
Install and configure clawback to manage my openclaw.json as modular JSON5 files.

Steps:
1. Install the clawback binary:
   go install github.com/jeff-french/clawback@latest

2. Initialize modular config from the existing openclaw.json:
   clawback init
   This automatically splits each top-level object key into a separate
   config/*.json5 file, creates the master template with $include directives,
   generates .clawback.json5, and verifies the round-trip.

3. Verify everything is clean:
   clawback diff

4. Install the clawback skill so you know the workflow:
   git clone https://github.com/jeff-french/clawback.git /tmp/clawback
   cp -r /tmp/clawback/skills/clawback ~/.openclaw/skills/
   rm -rf /tmp/clawback

5. Add a heartbeat hook that runs "clawback diff --quiet" after gateway start
   to detect config drift.
```

</details>

## Install

### Pre-built binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/jeff-french/clawback/releases).

### From source

Requires Go 1.22+:

```bash
go install github.com/jeff-french/clawback@latest
```

## License

MIT
