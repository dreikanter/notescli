# Design: Fix --help passthrough in grep/rg commands

**Date:** 2026-04-04
**Issue:** #63 — `grep`/`rg` `--help` shows underlying tool help, not notes-specific help

## Problem

Both `grep` and `rg` commands use `DisableFlagParsing: true` so that users can pass arbitrary flags to the underlying subprocess. A side-effect is that Cobra never sees `--help` — it falls through to the subprocess, which shows `grep --help` or `rg --help` output instead of notes-specific help.

`-h` is intentionally excluded from interception because it is a valid `grep` flag (suppress filename prefix).

## Changes

### 1. Intercept `--help` in `RunE`

At the start of `RunE` in both `internal/cli/grep.go` and `internal/cli/rg.go`, scan `args` for the string `"--help"`. If found, call `cmd.Help()` and return nil without invoking the subprocess.

```go
for _, arg := range args {
    if arg == "--help" {
        return cmd.Help()
    }
}
```

### 2. Improve `Long` descriptions

Add a sentence documenting the injected defaults so users running `notesctl grep --help` see the full picture.

**grep:**
```
Search note contents using grep. Only .md files are searched; .git directories are excluded.
The following flags are injected automatically: -r (recursive), -i (case-insensitive),
--include=*.md, --exclude-dir=.git. The notesctl path is appended as the last argument.
```

**rg:**
```
Search note contents using ripgrep (rg). Only .md files are searched.
The following flags are injected automatically: --glob *.md, --sortr path,
--heading, --no-line-number, --ignore-case. The notesctl path is appended as the last argument.
```

### 3. Tests

Add one test per command:

- `TestGrepHelp` — runs `notesctl grep --help`, asserts output contains `"grep"` and no error is returned.
- `TestRgHelp` — runs `notesctl rg --help`, asserts output contains `"rg"` and no error is returned.

Tests capture stdout via `os.Pipe()` consistent with the existing test helpers.

## Files changed

- `internal/cli/grep.go` — intercept `--help`, update `Long`
- `internal/cli/rg.go` — intercept `--help`, update `Long`
- `internal/cli/grep_test.go` — add `TestGrepHelp`
- `internal/cli/rg_test.go` — add `TestRgHelp`
- `CHANGELOG.md` — add entry for v0.1.41
