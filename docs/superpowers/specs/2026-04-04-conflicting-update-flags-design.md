# Reject conflicting update flags (#57)

## Problem

`notes update` accepts mutually exclusive flag pairs without error, silently
picking one winner. This makes intent ambiguous and can cause unexpected results
in scripts where flags are composed from variables.

Conflicting pairs:

| Flag A    | Flag B     | Current winner |
|-----------|------------|----------------|
| `--slug`  | `--no-slug`| `--no-slug`    |
| `--type`  | `--no-type`| `--no-type`    |
| `--tag`   | `--no-tags`| `--no-tags`    |
| `--public`| `--private`| `--private`    |

## Approach

Use Cobra's built-in `MarkFlagsMutuallyExclusive` API. This validates flag
groups before `RunE` executes using the same `pflag.Changed` check the manual
approach would use.

Benefits over hand-rolled validation:
- 4 declarative lines vs ~12 lines of if-statements
- Free shell completion behavior (conflicting flags auto-hidden)
- Consistent Cobra error formatting

## Changes

### `internal/cli/update.go`

**In `init()` (after flag registration, before `rootCmd.AddCommand`):**

Add four mutual exclusion declarations:

```go
updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
updateCmd.MarkFlagsMutuallyExclusive("public", "private")
```

**Flag description update (line 125):**

Change `"mark note as private in frontmatter (overrides --public)"` to
`"mark note as private in frontmatter"` since the flags are now mutually
exclusive rather than one overriding.

**`RunE` body:** No changes. The existing precedence logic becomes unreachable
for conflicting inputs but is harmless to keep.

### `internal/cli/update_test.go`

**Update three existing tests** that validate precedence behavior to instead
assert an error is returned:

- `TestUpdateNoSlugTakesPrecedenceOverSlug` (line 234) — rename to
  `TestUpdateSlugAndNoSlugConflict`, assert `err != nil`
- `TestUpdateNoTypeTakesPrecedenceOverType` (line 249) — rename to
  `TestUpdateTypeAndNoTypeConflict`, assert `err != nil`
- `TestUpdatePrivateTakesPrecedenceOverPublic` (line 382) — rename to
  `TestUpdatePublicAndPrivateConflict`, assert `err != nil`

**Add one new test:**

- `TestUpdateTagAndNoTagsConflict` — run with `--tag foo --no-tags`, assert
  `err != nil`

All four tests should verify the error message contains the conflicting flag
names (e.g., `strings.Contains(err.Error(), "slug")` and
`strings.Contains(err.Error(), "no-slug")`).

### `CHANGELOG.md`

Add entry for next patch version referencing PR number.

## Files affected

| File | Type of change |
|------|----------------|
| `internal/cli/update.go` | Add 4 `MarkFlagsMutuallyExclusive` calls, update flag description |
| `internal/cli/update_test.go` | Update 3 tests, add 1 new test |
| `CHANGELOG.md` | Add version entry |
