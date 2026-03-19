# CLAUDE.md — clawback

## What This Is

A Go CLI tool that manages modular OpenClaw configuration. It treats `openclaw.json` as a build artifact rendered from JSON5 source files via `$include` directives.

## Commands

- `init` — Bootstrap modular config from an existing monolithic `openclaw.json`
- `render` — Parse JSON5 sources, resolve `$include` directives, write `openclaw.json`
- `diff` — Compare rendered output against current `openclaw.json` (exit 0 = clean, 1 = drifted, 2 = error)
- `sync` — Backport changes from `openclaw.json` to JSON5 source files, then re-render

## Project Structure

```
cmd/           CLI commands (cobra)
internal/
  config/      Tool config loading (.clawback.json5)
  json5/       JSON5 parsing, $include resolution, surgical text editing
  jsonutil/    Deep comparison, diff generation, path operations
  render/      Full render pipeline
testdata/      Test fixtures
```

## Key Concepts

- **$include directives**: `{ "$include": "./path.json5" }` — replaced with the parsed contents of the referenced file
- **Passthrough sections**: Machine-managed paths (e.g. `meta`, `wizard`) that flow FROM `openclaw.json` TO config files (reverse direction). Configured in `.clawback.json5`
- **Comment preservation**: Sync edits JSON5 files surgically (find-and-replace) rather than rewriting from parsed data

## Build & Test

```bash
go build -o clawback .
go test ./...
go vet ./...
```

## Style

- Go 1.22+
- `fmt.Errorf` with `%w` for error wrapping
- No global mutable state
- Table-driven tests
