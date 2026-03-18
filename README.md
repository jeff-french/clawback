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
# Render openclaw.json from config/openclaw.json5
clawback render

# Show diffs between current openclaw.json and what render would produce
clawback diff

# Backport changes from openclaw.json to JSON5 sources, then re-render
clawback sync

# Check exit code only (for CI/heartbeat)
clawback diff --quiet
```

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

## Install

```bash
go install github.com/jeff-french/clawback@latest
```

## License

MIT
