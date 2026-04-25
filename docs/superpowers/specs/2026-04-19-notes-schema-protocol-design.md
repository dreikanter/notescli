# Notes schema protocol design

Date: 2026-04-19

Issue: [#104](https://github.com/dreikanter/notesctl/issues/104)

## Context

After [PR #111](https://github.com/dreikanter/notesctl/pull/111) and [PR #113](https://github.com/dreikanter/notesctl/pull/113) the frontmatter API is:

- Struct: `note.Frontmatter`.
- Parse: `note.ParseNote(data []byte) (Frontmatter, []byte, error)` — real errors, strict document-level validation.
- Emit: `note.FormatNote(f Frontmatter, body []byte) []byte`.

Adding a known field to `Frontmatter` is a one-line struct addition. Several problems remain:

1. **Unknown-field loss.** `ParseNote` calls `yaml.Unmarshal` into the `Frontmatter` struct, which silently drops any YAML key that is not a declared field. A user (or downstream tool) hand-adding `featured: true` loses it on the next `notesctl update`.
2. **Cross-project drift.** `notes-pub` imports `notesctl/note` and depends on its typed struct. `notes-view` has its own parser. A new field (`featured`, `aliases`, etc.) that only matters to a downstream project today requires a notesctl release before any of them can rely on it surviving edits.
3. **Filename/frontmatter split.** Metadata lives in two places. Slug is duplicated between filename and frontmatter, with no documented rule about which wins. Type (`todo`, `backlog`, `weekly`) is filename-only (encoded as a `.type` dot-suffix) and gated by `KnownTypes`, making new types a code change.
4. **No published contract.** Downstream projects and future contributors have no single place to look up which frontmatter keys notesctl reserves, and with what semantics.

Backward compatibility with existing stores is **not a concern** for this design — the format is treated as greenfield.

## Goals

- Adding a new frontmatter field is trivial for notesctl and free of charge for downstream consumers that don't care about it.
- Unknown fields round-trip safely through `notesctl update` and any other notesctl command that rewrites a note.
- `notes-pub` and `notes-view` can consume fields that notesctl doesn't recognize without waiting for a notesctl release.
- Filename identity is stable and minimal; everything else is frontmatter.
- There is one documented source of truth for which frontmatter keys are reserved and what they mean.

## Non-goals

- No schema version marker, no migration protocol between schema versions. Semantic changes (e.g., renaming a reserved field, changing a type) happen ad-hoc and are announced in `CHANGELOG.md` under the relevant notesctl version.
- No `notesctl lint` command. Can be added later if format drift becomes a real problem.
- No namespacing convention for custom frontmatter keys. Bare keys, Obsidian-style.
- No one-shot migration of existing `~/Notes` archives. Existing notesctl parse as before for the identity fields (date, ID); previously filename-only `type` is read as empty until the user edits the frontmatter.
- No `aliases`, `featured`, or other specific new reserved fields in this design. It enables them; the actual fields land when there is a concrete use case.

## Mental model

1. **Filename = identity + optional cache.** `YYYYMMDD_ID` is the stable identity and the primary sort/grouping key. Anything else in the filename (slug, type dot-suffix) is a denormalized view of frontmatter, maintained *one-way* (fm → filename) on explicit user command. Mismatches between filename cache and frontmatter are tolerated and ignored at read time.
2. **Frontmatter = canonical metadata.** Known fields are typed in Go and have documented semantics. Unknown fields are preserved verbatim through edits. Adding a field is a non-event for consumers that don't care.
3. **Cross-project contract = one markdown file.** `SCHEMA.md` lists reserved frontmatter keys and their semantics. Everything not listed is fair game for users and downstream projects; future reserved names may collide, and such collisions are called out explicitly in `CHANGELOG.md`.

## Data model

### `note.Frontmatter`

```go
type Frontmatter struct {
    Title       string                `yaml:"title,omitempty"`
    Slug        string                `yaml:"slug,omitempty"`
    Type        string                `yaml:"type,omitempty"`
    Tags        []string              `yaml:"tags,omitempty"`
    Description string                `yaml:"description,omitempty"`
    Public      bool                  `yaml:"public,omitempty"`
    Extra       map[string]yaml.Node  `yaml:"-"`
}
```

- `Type` is new (moved in from the filename).
- `Extra` is new. It holds every mapping pair whose key is not a reserved name. Values are kept as `yaml.Node` so structure (scalar, list, map, nested) survives untouched.
- `Extra` is intentionally not tagged for yaml. It is populated and emitted by custom code, not by struct (un)marshaling.

### Parser behavior

`ParseNote` continues to call `yaml.Unmarshal` into a `Frontmatter` value, so document-level strictness (from PR #113 — duplicate keys rejected, non-mapping top-level documents rejected, control characters rejected) is preserved as-is. To capture unknown keys into `Extra`, `Frontmatter` implements `UnmarshalYAML(node *yaml.Node) error`:

- If the node is not a mapping, return an error (strictness preserved).
- Walk the mapping's `Content` in pairs.
- For each pair, if the key matches a reserved name (`title`, `slug`, `type`, `tags`, `description`, `public`), `node.Decode` the value into the corresponding typed field. On decode error, return the error (keeps document-level strictness — no per-field tolerance).
- Otherwise, copy the value `yaml.Node` into `Extra[key]`.
- An empty or missing frontmatter block yields a zero `Frontmatter{}` with `Extra == nil` (handled by `ParseNote` before `UnmarshalYAML` is called).

### Writer behavior

`FormatNote` continues to call `yaml.Marshal(f)`. `Frontmatter` implements `MarshalYAML() (interface{}, error)` that returns a `*yaml.Node` composed manually:

- Reserved fields first in fixed order: `title, slug, type, tags, description, public`.
- Reserved fields with a zero value are omitted (matches the `omitempty` discipline of the struct-tag form).
- `Extra` keys after reserved fields, alphabetically sorted by key.
- `IsZero()` is extended so it returns false as soon as `Extra` is non-empty; otherwise the reflection-based check still covers all reserved fields.
- Deterministic output: same input always produces the same bytes.

### Type registry

`note.KnownTypes` becomes a soft registry of types that trigger special notesctl behavior. Proposed name: `note.TypesWithSpecialBehavior` (final name subject to review; any well-named symbol is fine).

- Populated with `"todo"`, `"backlog"`, `"weekly"` — unchanged set.
- `IsKnownType` is renamed to `HasSpecialBehavior` (or removed; most call sites are doing `KnownTypes`-style filtering and can inline).
- `notesctl new --type meeting` succeeds. Nothing special happens; `type: meeting` is written to frontmatter and (by default) cached in the filename as `.meeting`.

## Filename model

### `ParseFilename`

- Identity = `YYYYMMDD_ID`. Unchanged.
- A single dot-suffix on the base name (e.g., `.todo` in `20260106_8823.todo.md`) is accepted as a **filename-reported type** and populates `Note.Type`. This is what `Scan`-driven filters (`FilterByTypes`, `notesctl ls --type X`, `notesctl resolve todo`) operate on, for performance — they never read file contents. Any dot-suffix string is accepted; there is no `TypesWithSpecialBehavior` gate at the filename layer.
- When a specific note's contents are parsed (e.g., in `notesctl update`, `notesctl annotate`), the frontmatter `type` is **canonical**: if present, it overrides the filename-reported value; if absent, the filename-reported value stands. Commands that write frontmatter should prefer `fm.Type` as input; commands that operate on scans of filenames are free to use `Note.Type` as-is.
- Mismatches between filename-reported type and frontmatter type are tolerated at read time. `notesctl update --sync-filename` is the mechanism for reconciling them on explicit command.
- Arbitrary dot-suffix strings are accepted; no `KnownTypes` check at the filename layer.
- Slug continues to be read from the filename for parsing purposes, but frontmatter wins if both are present and differ.

### Command-surface changes

**`notesctl new`**
- Writes typed fields (including `type`) into frontmatter.
- Always caches `slug` and `type` into the filename at creation. There is no opt-out flag: the file is being created, there is no existing name to preserve, so cache-at-creation is the single default.

**`notesctl update`**
- Preserves `Extra` through the rewrite (because `ParseNote` now populates `Extra` via the custom `UnmarshalYAML`, and `FormatNote` emits it via the custom `MarshalYAML`).
- **No longer auto-renames the file on `--slug` or `--type` changes.** Updates frontmatter only; filename is untouched. This is a deliberate change from current behavior: the filename cache is not authoritative, so the update command should not pretend it is.
- New flag: `--sync-filename`. Renames the file so slug/type cache matches frontmatter. Strips suffixes that have no frontmatter counterpart (e.g., if fm.Type is now empty, any `.type` dot-suffix is stripped). Prints the resulting path to stdout.
- Calling with *only* `--sync-filename` (no content flags) performs the rename only and is idempotent (a no-op if the filename already matches fm).
- Calling with both content flags and `--sync-filename` (e.g., `notesctl update --slug bar --sync-filename`) updates fm then renames in the same invocation.
- Calling with no flags at all continues to error (existing behavior).

**`notesctl new_todo` and other type-specific commands**
- Continue to set `type: todo` in frontmatter. Existing rollover / daily-task behavior is gated on `TypesWithSpecialBehavior`, not on `KnownTypes` as a validation set.

**Removed:**
- `--no-cache-filename` (not introduced).
- `notesctl migrate` (not introduced).

## Cross-project contract: `SCHEMA.md`

A single markdown file at the notesctl repo root, with one section per reserved key:

```markdown
# Note frontmatter schema

Reserved keys live in the typed `Frontmatter` struct in
`github.com/dreikanter/notesctl/note`. Any key not listed here is
preserved verbatim on read/write and ignored by notesctl itself.

Downstream projects (notes-pub, notes-view) and users are free to
introduce new bare keys. Collision risk with future reserved names
is called out in CHANGELOG when a new reserved key is added.

## Reserved keys

### title
- Type: string
- Semantics: human-readable title; optional.
- Consumers: notes-pub (HTML `<h1>`, `<title>`), notes-view (sidebar).

### slug
- Type: string
- Semantics: URL-safe identifier, canonical in frontmatter. The
  filename may carry a cached copy; on mismatch, frontmatter wins.
- Consumers: notesctl (`new`, `update --sync-filename`),
  notes-pub (URL path segment).

### type
- Type: string
- Semantics: note category. Any value is valid. A small set of
  values (`todo`, `backlog`, `weekly`) trigger special notesctl
  behavior; see `TypesWithSpecialBehavior`.
- Consumers: notesctl (filters, rollover), notes-pub / notes-view
  (optional rendering).

### tags
- Type: list of strings
- Semantics: free-form tags. Matched case-sensitively. In-body
  `#hashtag` usage is a separate feature not governed by this field.
- Consumers: notesctl (`tags`, filters), notes-pub (tag pages,
  feed), notes-view.

### description
- Type: string
- Semantics: short summary; optional.
- Consumers: notes-pub (meta description), notes-view.

### public
- Type: bool
- Semantics: mark for inclusion in the published site. Absent or
  non-true = private.
- Consumers: notes-pub (inclusion filter).

## Unreserved keys

Any other top-level key is preserved untouched by notesctl.
Nested structures (mappings, sequences) are preserved intact.

Duplicate top-level keys are rejected at the document level (per
PR #113). Non-string keys and anchors/aliases in the YAML tree
are preserved inside `Extra` values as-is but are not specifically
tested; use at your own risk.
```

### Discipline

Adding a key to `Frontmatter` requires updating `SCHEMA.md` in the same PR. Reviewing the PR includes checking the entry. A `CHANGELOG.md` entry references both the PR and the new schema entry.

## Rollout

Order of operations within notesctl:

1. Land `note` package changes: add `Type` and `Extra` to `Frontmatter`; extend parser; extend writer; rename `KnownTypes` → `TypesWithSpecialBehavior`; relax `ParseFilename` dot-suffix handling.
2. Update `notesctl new` to set `Type` in fm and cache in filename at creation.
3. Update `notesctl update`: preserve `Extra`; remove auto-rename; add `--sync-filename` flag.
4. Add `SCHEMA.md` at repo root.
5. `CHANGELOG.md` entry under the next patch version.

Each of these is a separate commit, per the repo's atomic-commit convention.

Downstream:

- **notes-pub** — separate issue tracks dep bump to the new notesctl, exposure of `Type` (already implicit via struct), and reading `featured` / other future fields via `fm.Extra`.
- **notes-view** — separate issue tracks whether to switch its index to `notesctl/note` (deduplicates parsing) or mirror the `Extra` pattern locally.

## Open points (review-time)

- Final names for `TypesWithSpecialBehavior` and `--sync-filename`. The semantics are what matters; names are bikesheddable in the implementation PR.
- Whether `IsKnownType` stays, is renamed, or is removed in favor of inline checks.
- Whether the fm-over-filename override should live in a `note`-package helper (e.g., `note.LoadNote(path) (Note, Frontmatter, error)`) or inline in each command that needs it. The plan defaults to inline override at call sites; a helper can be introduced later if the pattern repeats.

## Implementation notes

- All changes are in `note/frontmatter.go`, `note/frontmatter_test.go`, `note/note.go`, `internal/cli/new.go`, `internal/cli/update.go`, and the new `SCHEMA.md`.
- `gopkg.in/yaml.v3` is already a dependency; no new deps.
- Tests must cover:
  - `Extra` round-trip (parse, re-emit, content-identical for unknown fields).
  - Reserved-field preservation when `Extra` is non-empty.
  - `Type` round-trip via frontmatter.
  - `notesctl update` no longer renames on `--slug` or `--type`.
  - `notesctl update --sync-filename` renames correctly, strips absent-in-fm suffixes, is idempotent.
  - `notesctl update --sync-filename` with only that flag is a no-op when already synced.
  - Pre-existing `.todo.md` files parse with empty `Type` when fm has none.
