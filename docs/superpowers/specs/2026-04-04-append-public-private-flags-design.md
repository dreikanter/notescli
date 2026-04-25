# Design: Add --public/--private Flags to `append`

**Date:** 2026-04-04
**Issue:** #68 — append --create missing --public/--private flags available on new

## Problem

The `new` command exposes `--public` and `--private` flags that write a `public:` field into note frontmatter. The `append` command can create notesctl (via `--create` or `--today`) but lacks these flags, making it impossible to set visibility when creating through `append`.

## Design

### New flags

Add to `registerAppendFlags()` in `internal/cli/append.go`:

```
--public   mark note as public in frontmatter (requires --create or --today)
--private  mark note as private in frontmatter (requires --create or --today; overrides --public)
```

### Validation — mirrors --title / --description

- `--public` or `--private` used without `--create`/`--today` → return error
- Append resolves to an existing note and `--public`/`--private` was set → print warning to stderr (flags are ignored), same pattern as the existing title/description warning

### Propagation

In `RunE`, read flags and compute:

```go
publicFlag, _ := cmd.Flags().GetBool("public")
privateFlag, _ := cmd.Flags().GetBool("private")
```

Pass `Public: publicFlag && !privateFlag` to `createNoteParams` — identical to `new.go`.

Extend the `!needsCreate` warning to include public/private:

```go
if !needsCreate && (title != "" || description != "" || publicFlag || privateFlag) {
    fmt.Fprintln(cmd.ErrOrStderr(), "warning: --title, --description, --public, and --private are ignored when appending to an existing note")
}
```

### No changes needed elsewhere

`createNote` and `createNoteParams` already have the `Public bool` field. The frontmatter builder already handles it. No other files need to change.

## Tests

Four new tests in `internal/cli/append_test.go`:

| Test | Assertion |
|------|-----------|
| `TestAppendCreateWithPublic` | `--create --public` → `public: true` in frontmatter |
| `TestAppendCreatePrivateOverridesPublic` | `--create --public --private` → no `public: true` (private wins) |
| `TestAppendPublicWithoutCreateErrors` | `--public` without `--create`/`--today` → error |
| `TestAppendPrivateWithoutCreateErrors` | `--private` without `--create`/`--today` → error |
