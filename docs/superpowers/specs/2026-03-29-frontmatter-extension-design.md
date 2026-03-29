# notescli Frontmatter Refactor Design

Date: 2026-03-29

## Context

`notescli/note` is the canonical source of note-format knowledge, shared with `notespub` as a Go module dependency. Currently all YAML frontmatter handling is hand-rolled:

- `ParseFrontmatterFields` — line-by-line string matching, only reads `title`, `tags`, `description`
- `BuildFrontmatter` — manual string concatenation to produce YAML output

Both are fragile: special YAML characters (colons, quotes) in titles or tags would produce invalid output or silently misparse. `notespub` also needs fields not covered (`public`), and any future consumer would need to duplicate knowledge of the format.

Additionally, `BuildFrontmatter` drops all fields it doesn't know about — round-tripping a note with unknown fields (e.g. `public: true`) through notescli would silently lose them.

## Goal

Replace all manual YAML handling with `gopkg.in/yaml.v3`. No mix of manual and library approaches. Preserve unknown frontmatter fields on update.

## API Changes

### Parsing

Replace hand-rolled parser with a generic function:

```go
// ParseFrontmatter returns all frontmatter fields as a map.
// Returns nil if no valid frontmatter is present.
func ParseFrontmatter(data []byte) map[string]any
```

- Extracts the YAML block using existing delimiter logic
- Unmarshals into `map[string]any` via `yaml.v3` — supports any field, any value type
- `ParseFrontmatterFields` is reimplemented on top of `ParseFrontmatter` — same signature, backward compatible
- `StripFrontmatter` is unchanged

### Writing — new notes

```go
// BuildFrontmatter generates YAML frontmatter from the given fields.
// Use for new notes. Same signature, now backed by yaml.v3 marshal.
func BuildFrontmatter(f FrontmatterFields) string
```

- Correctly handles special characters in all field values
- Used only when creating a note from scratch

### Writing — existing notes

```go
// UpdateFrontmatter merges fields into the note's existing frontmatter.
// Unknown frontmatter fields are preserved.
// Returns the full updated note content (frontmatter + body).
func UpdateFrontmatter(data []byte, f FrontmatterFields) ([]byte, error)
```

- `data` is the full note file content
- Extracts existing YAML block → unmarshals to `map[string]any`
- Merges known fields from `f` onto the map (only non-zero values)
- Marshals full map back to YAML — unknown fields survive
- Reconstructs note: new frontmatter + original body via `StripFrontmatter`
- No file I/O — caller reads and writes the file

### `FrontmatterFields` struct

Add `Slug` — it is a note management concern (can be set in frontmatter to override the filename slug):

```go
type FrontmatterFields struct {
    Title       string
    Slug        string   // ← added
    Tags        []string
    Description string
}
```

`public` is a publishing concern — not added to `FrontmatterFields`. notespub reads it directly from `ParseFrontmatter` map:

```go
fm := note.ParseFrontmatter(data)
public, _ := fm["public"].(bool)
```

## Implementation Notes

- Promote `gopkg.in/yaml.v3` from indirect to direct dependency (already in module graph via golangci-lint)
- No changes to `Note` struct, store scanning, or `StripFrontmatter`
- All changes in `frontmatter.go` and `frontmatter_test.go`
- Existing tests must continue to pass; add cases for:
  - Special characters in field values
  - Unknown fields preserved through `UpdateFrontmatter`
  - Block-style tag syntax (`- tag`) now supported via yaml.v3

## Out of Scope

- Validating frontmatter schema
- Supporting TOML or JSON frontmatter
