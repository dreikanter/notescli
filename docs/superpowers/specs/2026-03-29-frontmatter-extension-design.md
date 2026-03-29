# notescli Frontmatter Extension Design

Date: 2026-03-29

## Context

`notescli/note` is the canonical source of note-format knowledge, shared with `notespub` as a Go module dependency. Currently `ParseFrontmatterFields` returns a limited `FrontmatterFields` struct covering only `title`, `tags`, and `description`. The `notespub` site builder needs access to additional fields (`public`, `slug`) and potentially arbitrary frontmatter keys — without duplicating parsing logic.

## Goal

Extend the `note` package to expose full YAML frontmatter as a generic map, covering all fields any consumer might need, while keeping the existing typed API available for callers that prefer it.

## Approach

Add a new function alongside the existing one:

```go
// ParseFrontmatter returns all frontmatter fields as a map.
// Returns nil if no valid frontmatter is present.
func ParseFrontmatter(data []byte) map[string]any
```

- Uses `gopkg.in/yaml.v3` to unmarshal the frontmatter block into `map[string]any`
- `StripFrontmatter` and the frontmatter delimiter logic are reused as-is
- Existing `ParseFrontmatterFields` remains unchanged for backward compatibility
- `FrontmatterFields` convenience struct stays, backed by `ParseFrontmatter` internally to avoid duplication

## Fields notespub requires

| Field | Type | Purpose |
|---|---|---|
| `public` | bool | Whether note is included in the build |
| `title` | string | Page title |
| `slug` | string | URL slug override (falls back to slugified title) |
| `tags` | []string | Tag pages + related notes |
| `description` | string | Meta description (future use) |

## Implementation Notes

- Add `gopkg.in/yaml.v3` as a direct dependency (currently only in indirect deps via golangci-lint)
- No changes to `Note` struct or store scanning — frontmatter parsing is a separate concern
- New function covered by unit tests in `frontmatter_test.go`

## Out of Scope

- Validating frontmatter schema
- Supporting TOML or JSON frontmatter
