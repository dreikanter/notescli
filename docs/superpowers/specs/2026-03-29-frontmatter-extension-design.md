# notescli Frontmatter Extension Design

Date: 2026-03-29

## Context

`notescli/note` is the canonical source of note-format knowledge, shared with `notespub` as a Go module dependency. `notespub` needs fields not currently in `FrontmatterFields` (`public`, `slug`), and any future consumer would need to duplicate format knowledge or parse filenames itself.

## Goal

Add `Slug` and `Public` to `FrontmatterFields` so external tools (e.g. `notespub`) can read all relevant fields without parsing filenames or duplicating format knowledge. Keep the existing hand-rolled parser and builder — no yaml.v3 refactor.

## API Changes

### `FrontmatterFields` struct

```go
type FrontmatterFields struct {
    Title       string
    Slug        string // overrides filename slug for external consumers
    Tags        []string
    Description string
    Public      bool
}
```

Empty/zero values are omitted from frontmatter output (existing behavior, unchanged).

### Parsing

Extend the existing hand-rolled parser in `ParseFrontmatterFields` to recognize:
- `slug: <value>`
- `public: true` (absent or any non-`true` value → `false`)

### Writing

Extend `BuildFrontmatter` with the same omit-if-zero pattern:

```go
if f.Slug != "" {
    lines = append(lines, "slug: "+f.Slug)
}
if f.Public {
    lines = append(lines, "public: true")
}
```

Field order: `title`, `slug`, `tags`, `description`, `public`.

### `update.go` changes

When `--slug foo` is passed: updates both the filename and the `slug` frontmatter field (kept in sync).
When `--no-slug` is passed: removes slug from both filename and frontmatter.

## Implementation Notes

- All changes in `frontmatter.go`, `frontmatter_test.go`, and `internal/cli/update.go`
- Existing tests must continue to pass; add cases for `Slug` and `Public` round-trip
- No new dependencies

## Out of Scope

- Special character handling in field values
- Block-style tag syntax
- Preserving unknown frontmatter fields on update
- Validating frontmatter schema
