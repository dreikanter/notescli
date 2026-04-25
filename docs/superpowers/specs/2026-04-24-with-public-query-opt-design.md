# WithPublic query option

## Problem

Downstream consumers of the `note` package — primarily `notes-pub` —
currently filter public notes by reading every entry and skipping the
ones with `!fm.Public`:

```go
for entry := range store.All() {
    if !entry.Meta.Public { continue }
    // …
}
```

The `note` package already exposes composable filters (`WithType`,
`WithSlug`, `WithTag`, `WithExactDate`, `WithBeforeDate`) but no way to
filter on `Meta.Public`. Consumers re-implement the same predicate.

## Goal

Add a single `QueryOpt` so consumers can write:

```go
store.All(note.WithPublic(true))
```

and compose it with any existing filter.

## Non-goals

- No CLI surface (`notes ls --public` / `--private`) — deferred to a
  later PR once the primitive exists.
- No change to `Frontmatter` serialisation or the filename layout.
- No I/O optimisation (e.g., partial frontmatter reads). Frontmatter is
  parsed from the same single-file read that produces the body, and
  `OSStore` already reads entries concurrently. Revisit only if
  `notes-pub` profiling shows it matters.

## API

In `note/query.go`:

```go
// WithPublic matches entries whose Meta.Public equals v. A note with no
// frontmatter "public" key reads as Public: false, so WithPublic(false)
// matches both explicit "public: false" and missing-key notes.
func WithPublic(v bool) QueryOpt {
    return func(q *query) {
        q.publicSet = true
        q.public = v
    }
}
```

Two new fields on the unexported `query` struct:

```go
publicSet bool
public    bool
```

Shape matches the existing `typeSet`/`noteType`, `slugSet`/`slug`
pairs.

## Evaluation

`note/query.go`'s `matches()` gains one clause:

```go
if q.publicSet && entry.Meta.Public != q.public {
    return false
}
```

### MemStore

`MemStore.matchLocked` already calls `matches(e, q)` for every entry.
No changes required beyond the new clause in `matches()`.

### OSStore

`OSStore.collect` currently splits filtering into two helpers:

- `refMatchesFilename(r, q)` — pre-read, evaluated on `fileRef`
  (filename-derived: type, slug, date).
- `entryMatchesTags(entry, q)` — post-read, evaluated on `Meta`
  (body-hashtag-merged tags only).

`Public` lives in frontmatter, so it's a post-read filter. Rather than
add a third helper, replace the post-read call to `entryMatchesTags`
with `matches(entry, q)`. Filename-derivable clauses re-run on `Meta`
but that is cheap, and we collapse two places that each track "which
filters are post-read" into one.

`entryMatchesTags` is removed. `refMatchesFilename` stays — it still
provides the pre-read skip that avoids reading files that can't match.

## Tests

1. **`note/query_test.go`** — extend table-driven `matches()` tests with
   `WithPublic(true)` and `WithPublic(false)` cases against entries
   that have `Meta.Public: true` and `Meta.Public: false`. Also verify
   composition: `WithPublic(true)` combined with `WithTag("foo")`.

2. **`note/mem_store_test.go`** — seed a `MemStore` with a mix of
   public and private entries, assert:
   - `All(WithPublic(true))` returns only the public entries.
   - `All(WithPublic(false))` returns only the private entries.
   - `All(WithPublic(true), WithTag("x"))` intersects both filters.

3. **`note/os_store_test.go`** — write the same mix to disk (a note
   with `public: true` in frontmatter, a note without the key, a note
   with `public: false`), assert the same three cases. This covers the
   frontmatter → `Meta.Public` round-trip plus the filter.

No changes to existing tests are required; the new clause only runs
when `q.publicSet` is true.

## Documentation

- `CHANGELOG.md`: add an entry under the next patch version noting the
  new `WithPublic` opt and that it is additive (no breaking change).
- No `SCHEMA.md` change — the `public` field semantics are already
  documented there.

## Risk / compatibility

Purely additive. Existing callers (none of which set `q.publicSet`)
see identical behaviour. The `OSStore.collect` refactor is
behaviour-preserving: the `matches()` function is already the
canonical filter predicate and `MemStore` exercises it; the OSStore
test suite covers the same filter matrix post-refactor.

## Downstream (`notes-pub`)

Replace the hand-rolled loop:

```go
entries, err := store.All()
for _, e := range entries {
    if !e.Meta.Public { continue }
    // build page for e
}
```

with:

```go
entries, err := store.All(note.WithPublic(true))
for _, e := range entries {
    // build page for e
}
```

No change to `notesctl` is required to unblock that migration beyond
releasing this PR with a new patch version and a downstream
`go get notesctl@vX.Y.Z` bump.
