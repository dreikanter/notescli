# WithPublic Query Opt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a composable `WithPublic(bool)` `QueryOpt` to the `note` package so downstream consumers (notes-pub) can filter by `Meta.Public` without hand-rolling a post-`All()` loop.

**Architecture:** Extend the unexported `query` struct with a `publicSet`/`public` pair, add a single clause in `matches()`, and collapse `OSStore`'s post-read tag-only helper into a single `matches(entry, q)` call. `MemStore` needs no change beyond the new `matches()` clause — it already routes every filter through `matches()`.

**Tech Stack:** Go 1.x, stdlib `testing`. Build/test via `make test` and `make lint`.

Spec: [`docs/superpowers/specs/2026-04-24-with-public-query-opt-design.md`](../specs/2026-04-24-with-public-query-opt-design.md).

---

## File Structure

- **Modify** `note/query.go` — add `publicSet`/`public` fields to `query`, add `WithPublic` constructor, extend `matches()`.
- **Create** `note/query_test.go` — unit tests for `matches()` covering `WithPublic(true)`, `WithPublic(false)`, and composition with another filter. (No `query_test.go` exists today; the `matches()` function is currently exercised only indirectly through `mem_store_test.go`.)
- **Modify** `note/os_store.go` — inside `collect()`, replace the `entryMatchesTags` post-read call with `matches(entry, q)`; delete `entryMatchesTags`.
- **Modify** `note/mem_store_test.go` — add `TestMemStore_AllFilterByPublic` and a combined `WithPublic + WithTag` case.
- **Modify** `note/os_store_test.go` — add `TestOSStore_AllFilterByPublic` that writes three files (public: true, public: false, key absent) and asserts filtering in both directions.
- **Modify** `CHANGELOG.md` — add an entry for the new opt under the next patch version. Per `CLAUDE.md`, this lands in a follow-up commit on the same branch after the PR number is assigned.

---

## Task 1: Add `WithPublic` QueryOpt + `matches()` clause

**Files:**
- Create: `note/query_test.go`
- Modify: `note/query.go`

- [ ] **Step 1: Write the failing test**

Create `note/query_test.go` with this content:

```go
package note

import "testing"

func TestMatches_WithPublic(t *testing.T) {
	pub := Entry{ID: 1, Meta: Meta{Public: true}}
	priv := Entry{ID: 2, Meta: Meta{Public: false}}

	tests := []struct {
		name  string
		opt   QueryOpt
		entry Entry
		want  bool
	}{
		{"public=true matches public entry", WithPublic(true), pub, true},
		{"public=true rejects private entry", WithPublic(true), priv, false},
		{"public=false rejects public entry", WithPublic(false), pub, false},
		{"public=false matches private entry", WithPublic(false), priv, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := buildQuery([]QueryOpt{tc.opt})
			if got := matches(tc.entry, q); got != tc.want {
				t.Fatalf("matches = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMatches_WithPublicNotSetMatchesAny(t *testing.T) {
	q := buildQuery(nil)
	for _, e := range []Entry{
		{Meta: Meta{Public: true}},
		{Meta: Meta{Public: false}},
	} {
		if !matches(e, q) {
			t.Fatalf("matches with no opts should accept %+v", e)
		}
	}
}

func TestMatches_WithPublicAndTagAreAND(t *testing.T) {
	q := buildQuery([]QueryOpt{WithPublic(true), WithTag("x")})

	cases := []struct {
		entry Entry
		want  bool
	}{
		{Entry{Meta: Meta{Public: true, Tags: []string{"x"}}}, true},
		{Entry{Meta: Meta{Public: true, Tags: []string{"y"}}}, false},
		{Entry{Meta: Meta{Public: false, Tags: []string{"x"}}}, false},
	}
	for i, c := range cases {
		if got := matches(c.entry, q); got != c.want {
			t.Fatalf("case %d: matches = %v, want %v", i, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./note/ -run TestMatches_WithPublic -v`

Expected: compile error — `undefined: WithPublic`.

- [ ] **Step 3: Add the `publicSet`/`public` fields, `WithPublic` constructor, and `matches()` clause**

In `note/query.go`, extend the `query` struct (add the two new fields at the end, before the closing brace):

```go
type query struct {
	typeSet    bool
	noteType   string
	slugSet    bool
	slug       string
	tags       []string
	dateSet    bool
	date       time.Time
	beforeSet  bool
	beforeDate time.Time
	publicSet  bool
	public     bool
}
```

Add the `WithPublic` constructor after `WithBeforeDate` (before `buildQuery`):

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

Extend `matches()` with one additional clause (insert before `return true`):

```go
	if q.publicSet && entry.Meta.Public != q.public {
		return false
	}
	return true
}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `go test ./note/ -run TestMatches_WithPublic -v`

Expected: `PASS` for all three tests.

- [ ] **Step 5: Run the full note-package test suite to confirm no regressions**

Run: `go test ./note/...`

Expected: `ok  github.com/dreikanter/notesctl/note`.

- [ ] **Step 6: Commit**

```bash
git add note/query.go note/query_test.go
git commit -m "note: add WithPublic query opt"
```

---

## Task 2: MemStore filter test

**Files:**
- Modify: `note/mem_store_test.go`

MemStore's `matchLocked` already calls `matches(e, q)` per entry, so Task 1 made this work. This task pins the behaviour with an end-to-end test through the `Store.All` API.

- [ ] **Step 1: Add the test**

Append to `note/mem_store_test.go` (after the existing `TestMemStore_All…` tests):

```go
func TestMemStore_AllFilterByPublic(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Public: true, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Public: false, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Public: true, CreatedAt: day(2026, 1, 3)}})

	pub, err := s.All(WithPublic(true))
	if err != nil {
		t.Fatalf("All WithPublic(true): %v", err)
	}
	if len(pub) != 2 || pub[0].ID != 3 || pub[1].ID != 1 {
		t.Fatalf("WithPublic(true) = %v, want [3 1]", entryIDs(pub))
	}

	priv, err := s.All(WithPublic(false))
	if err != nil {
		t.Fatalf("All WithPublic(false): %v", err)
	}
	if len(priv) != 1 || priv[0].ID != 2 {
		t.Fatalf("WithPublic(false) = %v, want [2]", entryIDs(priv))
	}
}

func TestMemStore_AllPublicComposesWithTag(t *testing.T) {
	s := NewMemStore()
	mustPut(t, s, Entry{ID: 1, Meta: Meta{Public: true, Tags: []string{"x"}, CreatedAt: day(2026, 1, 1)}})
	mustPut(t, s, Entry{ID: 2, Meta: Meta{Public: true, Tags: []string{"y"}, CreatedAt: day(2026, 1, 2)}})
	mustPut(t, s, Entry{ID: 3, Meta: Meta{Public: false, Tags: []string{"x"}, CreatedAt: day(2026, 1, 3)}})

	got, err := s.All(WithPublic(true), WithTag("x"))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("WithPublic(true)+WithTag(x) = %v, want [1]", entryIDs(got))
	}
}
```

- [ ] **Step 2: Run the tests and confirm they pass**

Run: `go test ./note/ -run TestMemStore_AllFilterByPublic -v && go test ./note/ -run TestMemStore_AllPublicComposesWithTag -v`

Expected: both `PASS`.

- [ ] **Step 3: Commit**

```bash
git add note/mem_store_test.go
git commit -m "note: test MemStore WithPublic filter"
```

---

## Task 3: OSStore post-read filter — collapse `entryMatchesTags` into `matches()`

**Files:**
- Modify: `note/os_store_test.go`
- Modify: `note/os_store.go`

Before changing `OSStore.collect`, write an OSStore-level test that exercises the full path (frontmatter → `Meta.Public` → filter). It should already pass after Task 1 (because `Meta.Public` reaches the current post-read filter only if we check it — so this test **will fail until we change `collect()`**). TDD.

- [ ] **Step 1: Write the failing test**

Append to `note/os_store_test.go`:

```go
func TestOSStore_AllFilterByPublic(t *testing.T) {
	s := newOSTestStore(t)

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := s.Put(Entry{Meta: Meta{Title: "pub", Public: true, CreatedAt: base}, Body: "p\n"})
	if err != nil {
		t.Fatalf("Put pub: %v", err)
	}
	_, err = s.Put(Entry{Meta: Meta{Title: "priv-explicit", Public: false, CreatedAt: base.Add(24 * time.Hour)}, Body: "x\n"})
	if err != nil {
		t.Fatalf("Put priv-explicit: %v", err)
	}
	// A third note with no "public" key is implicit Public: false. Put
	// writes Public: false the same as explicit, so we seed one more
	// public entry to round-trip the true path instead.
	_, err = s.Put(Entry{Meta: Meta{Title: "pub2", Public: true, CreatedAt: base.Add(48 * time.Hour)}, Body: "y\n"})
	if err != nil {
		t.Fatalf("Put pub2: %v", err)
	}

	pub, err := s.All(WithPublic(true))
	if err != nil {
		t.Fatalf("All WithPublic(true): %v", err)
	}
	if len(pub) != 2 {
		t.Fatalf("WithPublic(true) len = %d, want 2 (got IDs %v)", len(pub), entryIDs(pub))
	}
	for _, e := range pub {
		if !e.Meta.Public {
			t.Fatalf("WithPublic(true) returned Public=false entry %d", e.ID)
		}
	}

	priv, err := s.All(WithPublic(false))
	if err != nil {
		t.Fatalf("All WithPublic(false): %v", err)
	}
	if len(priv) != 1 {
		t.Fatalf("WithPublic(false) len = %d, want 1 (got IDs %v)", len(priv), entryIDs(priv))
	}
	if priv[0].Meta.Public {
		t.Fatalf("WithPublic(false) returned Public=true entry %d", priv[0].ID)
	}
}
```

This test relies on the existing `entryIDs` helper used by other tests in the package.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./note/ -run TestOSStore_AllFilterByPublic -v`

Expected: FAIL. The assertion `len(pub) != 2` will report `len = 3` (or similar) because `OSStore.collect` currently evaluates only `entryMatchesTags` post-read and ignores `WithPublic`.

- [ ] **Step 3: Replace `entryMatchesTags` with `matches(entry, q)` in `collect`**

In `note/os_store.go`, change the two sites in `collect` that currently call `entryMatchesTags`:

Find:

```go
	if firstOnly {
		for _, r := range filtered {
			entry, err := s.readEntry(r)
			if err != nil {
				return nil, err
			}
			if entryMatchesTags(entry, q) {
				return []Entry{entry}, nil
			}
		}
		return nil, nil
	}

	entries, err := s.readConcurrent(filtered)
	if err != nil {
		return nil, err
	}

	out := entries[:0]
	for _, e := range entries {
		if entryMatchesTags(e, q) {
			out = append(out, e)
		}
	}
	return out, nil
```

Replace the two `entryMatchesTags(...)` calls with `matches(..., q)`:

```go
	if firstOnly {
		for _, r := range filtered {
			entry, err := s.readEntry(r)
			if err != nil {
				return nil, err
			}
			if matches(entry, q) {
				return []Entry{entry}, nil
			}
		}
		return nil, nil
	}

	entries, err := s.readConcurrent(filtered)
	if err != nil {
		return nil, err
	}

	out := entries[:0]
	for _, e := range entries {
		if matches(e, q) {
			out = append(out, e)
		}
	}
	return out, nil
```

- [ ] **Step 4: Remove the now-unused `entryMatchesTags` helper**

In `note/os_store.go`, delete this function (it was the only post-read helper and no longer has callers):

```go
// entryMatchesTags reports whether entry satisfies the WithTag filters in q.
// Called after readEntry so Meta.Tags has body hashtags merged in.
func entryMatchesTags(entry Entry, q query) bool {
	if len(q.tags) == 0 {
		return true
	}
	return hasAllTags(entry.Meta.Tags, q.tags)
}
```

`matches()` in `query.go` applies the `WithTag` check via `hasAllTags`, so the tag filter is preserved.

- [ ] **Step 5: Run the OSStore tests and confirm all pass**

Run: `go test ./note/ -run TestOSStore -v`

Expected: all OSStore tests (including the existing tag/type/date filter tests and the new public filter test) `PASS`. If any existing test fails, the failure is almost certainly a bug in the refactor — **do not** adjust tests to match; re-check the `matches()` vs `entryMatchesTags` parity.

- [ ] **Step 6: Run the full test suite and `make lint`**

Run: `make test && make lint`

Expected: tests pass, lint clean. (`go vet` via lint will catch the removed helper if a stray reference was left.)

- [ ] **Step 7: Commit**

```bash
git add note/os_store.go note/os_store_test.go
git commit -m "note: wire WithPublic through OSStore via matches()"
```

---

## Task 4: CHANGELOG entry (follow-up commit after PR is opened)

**Files:**
- Modify: `CHANGELOG.md`

Per `CLAUDE.md`, open the PR first so the PR number is assigned, then add this entry as its own atomic commit on the same branch and push.

- [ ] **Step 1: Identify the next version**

The topmost entry as of 2026-04-24 is `## [0.3.19] - 2026-04-24`. Bump the patch → `## [0.3.20]`. If another PR has merged ahead of this one, bump from whatever the topmost version is at the time this step runs.

- [ ] **Step 2: Add the entry**

In `CHANGELOG.md`, insert directly below the `# Changelog` heading (and above the current topmost entry):

```markdown
## [0.3.20] - 2026-04-24

### Added

- `note.WithPublic(v bool)` `QueryOpt` filters entries by `Meta.Public`. Downstream consumers (notes-pub) can now call `store.All(note.WithPublic(true))` instead of reading every entry and skipping non-public ones. Internally, `OSStore.collect` evaluates all post-read filters through the shared `matches()` predicate; the single-purpose `entryMatchesTags` helper is removed ([#NNN]).

[#NNN]: https://github.com/dreikanter/notesctl/pull/NNN
```

Replace `NNN` in **both** the `[#NNN]` reference and the link target with the real PR number assigned when the PR was opened. Update the date if the entry lands on a different day. If the top entry is already on the bumped date, re-use it; otherwise use today's date in `YYYY-MM-DD` form.

- [ ] **Step 3: Verify the file**

Run: `head -n 12 CHANGELOG.md`

Expected: the new `## [0.3.20]` block appears above the previous top entry with the PR number substituted.

- [ ] **Step 4: Commit and push**

```bash
git add CHANGELOG.md
git commit -m "changelog: 0.3.20 — WithPublic query opt"
git push
```

The PR updates in place.

---

## Self-Review

**Spec coverage**

- API `WithPublic(v bool)` → Task 1.
- `query` struct fields `publicSet`/`public` → Task 1.
- `matches()` clause → Task 1.
- `MemStore` works via shared `matches()` → verified by Task 2 (no impl change needed).
- `OSStore` post-read refactor (drop `entryMatchesTags`, use `matches()`) → Task 3.
- Tests for `matches()` (unit) + `MemStore.All` + `OSStore.All` (integration) → Tasks 1–3.
- CHANGELOG entry → Task 4.
- No `Frontmatter`/filename layout change → unchanged.
- No CLI flag → unchanged.

**Placeholder scan:** no TBDs, no "add error handling," no "similar to…" — every code step has the actual code. PR number is the only placeholder (`NNN`) and is marked with explicit replace instructions — unavoidable since the number is assigned on PR open.

**Type / name consistency:**

- Struct fields: `publicSet`, `public` — same in Task 1 definition and `matches()` clause.
- Function: `WithPublic(v bool) QueryOpt` — consistent throughout tasks.
- Removed helper: `entryMatchesTags` — called out in Task 3 Step 4 and not referenced afterwards.
- Test helpers: `mustPut`, `day`, `entryIDs`, `newOSTestStore` — all already exist in the respective `_test.go` files (verified during plan writing).
