# Error on `update` with no flags

**Issue:** [#69](https://github.com/dreikanter/notescli/issues/69)
**Date:** 2026-04-04

## Problem

`notes update <id>` with no flags silently reads the file, re-serializes it unchanged, writes it back, and prints the path as if the update succeeded. This is confusing for users and masks bugs in scripts where a flag variable may be empty.

## Solution

Return an error when no update flags are provided. Exit before any file I/O or note resolution.

**Error message:** `at least one update flag is required`

## Implementation

### Detection

Check `cmd.Flags().Changed()` for all update flags immediately after flag parsing. The flags to check:

- `tag`, `no-tags`
- `title`, `description`
- `slug`, `no-slug`
- `type`, `no-type`
- `public`, `private`

If none are changed, return an error.

### Placement in `update.go`

Insert the check after line 28 (flag variable assignments), before the type validation on line 29. This is the earliest exit point — before note resolution and file I/O.

```go
// Check that at least one update flag was provided.
updateFlags := []string{
    "tag", "no-tags", "title", "description",
    "slug", "no-slug", "type", "no-type",
    "public", "private",
}
hasFlag := false
for _, name := range updateFlags {
    if cmd.Flags().Changed(name) {
        hasFlag = true
        break
    }
}
if !hasFlag {
    return fmt.Errorf("at least one update flag is required")
}
```

### Test changes

In `update_test.go`, replace `TestUpdateNoFlagsUnchanged` with `TestUpdateNoFlagsErrors`:

- Call `runUpdate(t, root, "8823")` with no flags
- Assert error is returned (non-nil)
- Assert error message contains "at least one update flag is required"
- Remove the file-content comparison (file should not be touched)

No other tests are affected — all other test cases provide at least one flag.

## Scope

- **Files changed:** `internal/cli/update.go`, `internal/cli/update_test.go`
- **No new dependencies**
- **No changes to other commands**
