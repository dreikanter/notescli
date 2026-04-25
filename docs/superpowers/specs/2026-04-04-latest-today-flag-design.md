# Design: Add `--today` flag to `latest` command

**Issue:** #61 — `--today` flag is inconsistently available across commands

## Problem

The `--today` flag exists on `ls`, `append`, and `resolve` but is absent from `latest`. This makes the CLI inconsistent: users who rely on `--today` for daily-note workflows cannot narrow `latest` to today's notes.

## Solution

Add `--today` to `latestCmd` so it filters candidates to notes created today before applying the existing type/slug/tag filters.

## Changes

### `internal/cli/latest.go`

1. Import `"time"` (not currently imported).
2. Register flag in `init()`:
   ```go
   latestCmd.Flags().Bool("today", false, "filter to notes created today")
   ```
3. In `scanAndFilter()`, read the flag and apply `note.FilterByDate` immediately after `note.Scan`, before type/slug/tag filters:
   ```go
   today, _ := cmd.Flags().GetBool("today")
   if today {
       notes = note.FilterByDate(notes, time.Now().Format("20060102"))
   }
   ```
4. Expand the no-match error condition to include `today`:
   ```go
   if len(types) > 0 || len(slugs) > 0 || len(tags) > 0 || today {
       return nil, fmt.Errorf("no notes found matching the given criteria")
   }
   ```

### `internal/cli/latest_test.go`

1. In `runLatest()`, add the `--today` flag to the reset block so it is re-registered alongside the existing flags.
2. Add `TestLatestWithTodayNoMatch`: passes `--today`, expects an error. Testdata notes are all from 2026-01, so none match today (2026-04-04); this confirms the filter is applied.

### `CHANGELOG.md`

Add one entry for the next patch version referencing the PR.

## Error Behavior

When `--today` is set and no notes match:
- Returns: `"no notes found matching the given criteria"`
- Consistent with how other filters behave on `latest`.

## Out of Scope

- No changes to `ls`, `append`, or `resolve` — they already have `--today`.
- No new testdata files needed; the no-match case is sufficient to verify the filter is wired up.
