# CLAUDE.md — oc-include-guard

## What This Is

A Go CLI tool that manages modular OpenClaw configuration. It treats `openclaw.json` as a build artifact rendered from JSON5 source files via `$include` directives.

## Architecture

```
~/.openclaw/
├── openclaw.json           ← BUILD ARTIFACT (rendered output)
├── .oc-include-guard.json5 ← tool config
└── config/
    ├── openclaw.json5      ← MASTER TEMPLATE (source of truth)
    ├── env.json5
    ├── auth.json5
    ├── agents.json5
    ├── tools.json5
    ├── messages.json5
    ├── hooks.json5
    ├── channels.json5
    ├── gateway.json5
    ├── skills.json5
    └── plugins.json5
```

## Commands

### `render`
- Parse `config/openclaw.json5` master template
- For each `{ $include: "./path.json5" }` directive, parse the referenced JSON5 file
- For **passthrough** sections: if `openclaw.json` already exists, copy those sections from the existing file (they're machine-managed and flow FROM openclaw.json TO the template direction)
- Render the combined result as standard JSON → `openclaw.json`
- Exit 0 on success

### `diff`
- Render to a temp buffer (don't write)
- Compare rendered output against current `openclaw.json`
- Show structural diff (added/removed/changed keys with JSON paths)
- `--quiet` flag: exit code only (0 = clean, 1 = drifted)
- `--json` flag: output diff as JSON (for programmatic consumption)
- Default: human-readable colored diff output

### `sync`
- Compare current `openclaw.json` against what render would produce
- For each difference:
  - Identify which JSON5 source file owns that section
  - **New keys in openclaw.json**: backport to the JSON5 source file
  - **Changed values in openclaw.json**: backport to the JSON5 source file
  - **Keys in JSON5 but not openclaw.json**: KEEP in JSON5 (don't delete from source of truth)
- Re-render after backporting
- `--dry-run` flag: show what would change without modifying files

## JSON5 Comment Preservation Strategy

This is the critical requirement. When backporting changes to JSON5 files:

1. **Parse the JSON5 file** to understand its structure and get key-value mappings
2. **Also read the raw text** of the JSON5 file
3. For **new keys**: append before the final closing `}` of the relevant object, formatted to match the file's existing style (indentation, trailing commas, unquoted keys)
4. For **changed values**: locate the key in the raw text using the parsed structure as a guide, then replace only the value portion. Use a robust key-finding approach:
   - Find the key name (quoted or unquoted) followed by `:` 
   - Replace the value token(s) after the colon up to the next comma/closing brace/newline
   - Handle nested objects/arrays by tracking brace/bracket depth
5. **Never rewrite the entire file from parsed data** — always do surgical text edits
6. For **removed keys** (in openclaw.json but not JSON5): during sync, this means openclaw.json lost a key that JSON5 has — keep the JSON5 version (JSON5 is source of truth for non-passthrough sections)

## Config File (`.oc-include-guard.json5`)

```json5
{
  // Path to config directory (relative to openclaw home dir)
  configDir: "./config",
  // Path to output file (relative to openclaw home dir)  
  outputFile: "./openclaw.json",
  // Path to master template (relative to openclaw home dir)
  masterTemplate: "./config/openclaw.json5",
  // JSON paths that are "passthrough" — machine-managed sections
  // These flow FROM openclaw.json TO config files (reverse direction)
  // All other sections flow FROM config files TO openclaw.json
  passthrough: [
    "meta",
    "wizard",
    "plugins.installs",
  ],
}
```

Defaults if no config file exists:
- configDir: `./config`
- outputFile: `./openclaw.json`  
- masterTemplate: `./config/openclaw.json5`
- passthrough: `["meta", "wizard", "plugins.installs"]`

The tool resolves all paths relative to `~/.openclaw/` (the OpenClaw home directory), or accepts `--home /path/to/openclaw/dir` to override.

## Go Project Structure

```
oc-include-guard/
├── main.go              # CLI entry point (cobra or just flag)
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go          # Root command, config loading
│   ├── render.go        # render subcommand
│   ├── diff.go          # diff subcommand
│   └── sync.go          # sync subcommand
├── internal/
│   ├── config/
│   │   └── config.go    # Tool config loading (.oc-include-guard.json5)
│   ├── json5/
│   │   ├── parse.go     # JSON5 parsing (to map[string]any)
│   │   ├── edit.go      # Surgical text editing (comment-preserving writes)
│   │   └── include.go   # $include directive resolution
│   ├── jsonutil/
│   │   └── compare.go   # Deep structural comparison, diff generation
│   └── render/
│       └── render.go    # Full render pipeline
└── testdata/            # Test fixtures
    ├── simple/
    │   ├── config/
    │   │   ├── openclaw.json5
    │   │   └── env.json5
    │   └── expected.json
    └── with-comments/
        ├── config/
        │   ├── openclaw.json5
        │   └── plugins.json5
        └── expected.json
```

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/titanous/json5` — JSON5 parsing (for reading config files)
- `github.com/fatih/color` — colored terminal output
- `github.com/sergi/go-diff` — text diffing for human-readable output
- Standard library `encoding/json` — for JSON output rendering

## Key Implementation Details

### $include Resolution
An `$include` directive looks like: `{ "$include": "./relative/path.json5" }` or `{ $include: "./path.json5" }` (unquoted key in JSON5).

When a top-level value is an object with exactly one key `$include` whose value is a string path, replace that entire object with the parsed contents of the referenced file.

### Passthrough Section Handling
During `render`:
1. Parse the master template
2. Resolve all `$include` directives  
3. If `openclaw.json` already exists, for each passthrough path:
   - Read the value at that path from the existing `openclaw.json`
   - Override the rendered value with the existing one
4. Write the final result

During `sync`:
1. For passthrough sections: copy from `openclaw.json` → JSON5 source file
2. For all other sections: copy from JSON5 source file → render → `openclaw.json`

### JSON Path Notation
Use dot-separated paths: `plugins.installs`, `agents.defaults.model`
Array access not needed for current use cases.

### Exit Codes
- 0: success / no diff
- 1: diff found (for `diff` command) or sync needed
- 2: error (parse failure, file not found, etc.)

## Testing

Write table-driven tests. Key scenarios:
1. Clean render from template with includes
2. Render with passthrough sections
3. Diff detection after openclaw.json is clobbered
4. Sync backporting new keys to JSON5
5. Sync backporting changed values to JSON5
6. Comment preservation during sync
7. Nested $include values
8. Missing config file (use defaults)
9. Passthrough sections flow in correct direction

## Build

```bash
go build -o oc-include-guard .
```

## Style

- Go 1.22+
- Use `fmt.Errorf` with `%w` for error wrapping
- No global state
- Context-aware where appropriate
- Comprehensive error messages (include file paths, line numbers when possible)
