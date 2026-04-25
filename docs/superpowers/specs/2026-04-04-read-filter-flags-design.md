# read filter flags design

**Date:** 2026-04-04  
**Issue:** [#62](https://github.com/dreikanter/notesctl/issues/62)

## Problem

`notes read` accepts only a single positional ref (ID, path, basename, slug, or type). There is no way to filter by tag in one command — you must chain `notes latest --tag work` into `notes read`. The `--today` restriction available on `ls`, `append`, and `resolve` is also absent.

## Decision

Add optional filter flags (`--type`, `--slug`, `--tag`, `--today`) to `read`, mutually exclusive with the positional argument — same pattern as `append`. When filter flags are used, read the most recent matching note (notes are sorted newest-first by `note.Scan`).

## Command signature

```
notes read [<id|path|basename|slug|type>] [--type T] [--slug S] [--tag TAG]... [--today] [--no-frontmatter]
```

## Behavior

| Scenario | Result |
|---|---|
| `notes read 101` | Unchanged — resolves by ref via `ResolveRef` |
| `notes read --tag work` | Reads most recent note tagged `work` |
| `notes read --type todo --today` | Reads most recent `todo` note from today |
| `notes read --slug meeting` | Reads most recent note with slug `meeting` |
| `notes read 101 --tag work` | Error: cannot combine positional arg with filter flags |
| `notes read` (no args, no flags) | Error: specify a note by positional argument or filter flags |
| Filters match nothing | Error: no notes found matching the given criteria |

`--no-frontmatter` works in both modes.

`--today` purely filters by date (does not imply create-if-missing — `read` never creates notes).

## Implementation

### `internal/cli/read.go`

- Change `cobra.ExactArgs(1)` → `cobra.MaximumNArgs(1)`
- Add flags: `--type` (string), `--slug` (string), `--tag` (string slice), `--today` (bool)
- Validation guard:
  - `len(args) == 1 && hasFilters` → error "cannot combine positional argument with filter flags"
  - `len(args) == 0 && !hasFilters` → error "specify a note by positional argument or filter flags (--type, --slug, --tag, --today)"
- Positional path: unchanged — `note.ResolveRef(root, args[0])`
- Filter path: `note.Scan` → `FilterByDate` (if `--today`) → `FilterByTypes` (if `--type`) → `FilterBySlugs` (if `--slug`) → `FilterByTags` (if `--tag`) → take `notes[0]`; error if empty

Filter flags match `append.go` types: `--type` and `--slug` are single-value strings; `--tag` is a string slice (repeatable, AND logic).

### `internal/cli/read_test.go` (new file)

New test cases:
- `TestReadByTagFilter` — reads most recent note with matching tag
- `TestReadByTypeFilter` — reads most recent note with matching type
- `TestReadBySlugFilter` — reads most recent note with matching slug
- `TestReadByTodayFilter` — returns error (no today notes in testdata)
- `TestReadPositionalArgWithFilterErrors` — errors when both provided
- `TestReadNoTargetErrors` — errors when nothing provided
- `TestReadNoMatchErrors` — errors when filters match nothing

Test harness mirrors `append_test.go`: `runRead` helper resets flags, sets `--path`, calls `rootCmd.Execute()`, captures stdout.

## Files changed

- `internal/cli/read.go` — rewrite command body and `init()`
- `internal/cli/read_test.go` — new test file
- `CHANGELOG.md` — add entry for next patch version
