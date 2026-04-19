# Note frontmatter schema

Reserved keys are the fields declared on `Frontmatter` in
`github.com/dreikanter/notes-cli/note`. Any key not listed below is
preserved verbatim on read/write (stored in `Frontmatter.Extra`)
and ignored by notes-cli itself.

Downstream projects (notes-pub, notes-view) and users are free to
introduce new bare keys. Collision risk with future reserved names
is called out in `CHANGELOG.md` when a new reserved key is added.

## Reserved keys

### title
- **Type:** string
- **Semantics:** human-readable title; optional.
- **Consumers:** notes-pub (HTML `<h1>`, `<title>`), notes-view (sidebar).

### slug
- **Type:** string
- **Semantics:** URL-safe identifier, canonical in frontmatter. The
  filename may carry a cached copy; on mismatch, frontmatter wins.
- **Consumers:** notes-cli (`new`, `update --sync-filename`),
  notes-pub (URL path segment).

### type
- **Type:** string
- **Semantics:** note category. Any value is valid. A small set of
  values (`todo`, `backlog`, `weekly`) trigger special notes-cli
  behavior; see `note.TypesWithSpecialBehavior`. The filename may
  carry a cached copy as a `.type` dot-suffix; on mismatch,
  frontmatter wins.
- **Consumers:** notes-cli (filters, rollover), notes-pub / notes-view
  (optional rendering).

### tags
- **Type:** list of strings
- **Semantics:** free-form tags, matched case-sensitively. In-body
  `#hashtag` usage is a separate feature not governed by this field.
- **Consumers:** notes-cli (`tags`, filters), notes-pub (tag pages,
  feed), notes-view.

### description
- **Type:** string
- **Semantics:** short summary; optional.
- **Consumers:** notes-pub (meta description), notes-view.

### public
- **Type:** bool
- **Semantics:** mark for inclusion in the published site. Absent or
  non-true = private.
- **Consumers:** notes-pub (inclusion filter).

## Unreserved keys

Any other top-level key is preserved untouched by notes-cli. Nested
structures (mappings, sequences) are preserved intact.

Duplicate top-level keys are rejected at the document level.
Non-scalar keys are rejected. Anchors and aliases in the YAML
tree are preserved inside `Extra` values as-is but are not
specifically tested in notes-cli; use at your own risk.

## Process

Adding a key to `Frontmatter` requires updating this file in
the same PR. `CHANGELOG.md` entries reference both the PR and the
new schema entry.
