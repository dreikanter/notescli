# Design: Make `ls` output absolute paths

**Issue:** [#55](https://github.com/dreikanter/notesctl/issues/55)
**Date:** 2026-04-04

## Problem

`ls` outputs store-relative paths (e.g. `2026/04/20260404_103.todo.md`), but every
ref-consuming command (`read`, `resolve`, `append`, `update`) uses `ResolveRef` which
resolves paths containing `/` via `filepath.Abs` relative to CWD — not the store root.

This breaks the natural Unix pipeline: `notes ls --type todo | xargs notes read`.

Every other path-emitting command (`new`, `append`, `latest`, `resolve`, `update`) already
outputs absolute paths. `ls` is the only outlier.

## Solution

Change `ls` to output absolute paths by prepending the store root to each `RelPath`.

### Code change

**`internal/cli/ls.go:57`** — replace:

```go
fmt.Fprintln(cmd.OutOrStdout(), n.RelPath)
```

with:

```go
fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
```

Add `"path/filepath"` to the import block.

`root` is already resolved at line 23 via `mustNotesPath()`.

This matches the pattern in `latest.go:22` and `resolve.go:31`.

### Test changes

**`internal/cli/ls_test.go`** — tests that inspect output lines need to expect absolute
paths. The test helper `runLs` already passes `--path root` where `root` is the testdata
directory, so assertions using `strings.Contains` on filenames (e.g. `"todo"`, `"meeting"`,
`"8814"`) will continue to pass without changes. Tests that check line counts are unaffected.

Verify all existing tests pass after the one-line production change. If any assertion
compares exact path strings, update to expect the absolute form.

### Changelog

Next version will be `v0.1.41`. Add entry to `CHANGELOG.md` referencing PR number.

## What is NOT changing

- No new flags (no `--relative`, no `--absolute`)
- No changes to `ResolveRef`
- No changes to any other command
- No changes to `note.Note` struct or `Scan`
