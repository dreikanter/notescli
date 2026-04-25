# Notes schema protocol implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the note frontmatter format extensible, round-trip-safe, and clearly contracted across notesctl / notes-pub / notes-view — implementing the design at `docs/superpowers/specs/2026-04-19-notes-schema-protocol-design.md`.

**Architecture:** All work lives inside notesctl. The `note` package's `Frontmatter` gains a `Type` field and an `Extra map[string]yaml.Node` so unknown frontmatter keys survive edits. Preservation is implemented by adding custom `UnmarshalYAML` / `MarshalYAML` methods on `Frontmatter`; the public `ParseNote` / `FormatNote` signatures are unchanged. `ParseFilename` is relaxed to accept any dot-suffix as a filename-reported type. `KnownTypes` / `IsKnownType` are renamed to reflect their soft-registry semantics. The `new` and `update` commands stop rejecting unknown types and no longer auto-rename files; a new `--sync-filename` flag on `update` is the explicit reconciliation hook. A `SCHEMA.md` at repo root documents the reserved frontmatter keys.

**Tech Stack:** Go, `gopkg.in/yaml.v3` (a dependency since PR #111), cobra CLI, standard Go testing.

---

## Prereqs

- Run from the worktree at `/Users/alex/src/notesctl/.claude/worktrees/humming-foraging-wozniak`.
- Branch is rebased onto `origin/main` (commit `b30f330`, which is [PR #113](https://github.com/dreikanter/notesctl/pull/113) — the `ParseNote` / `FormatNote` API refactor).
- The design spec is already committed on the branch.
- PR #113 also moved `CHANGELOG.md`'s "next patch" to `[0.1.72]`. This plan's CHANGELOG entry targets `[0.1.73]`.

---

## File structure

### New files

- `SCHEMA.md` — repo-root schema registry.

### Modified files

- `note/frontmatter.go` — add `Type` and `Extra` to the `Frontmatter` struct; implement `UnmarshalYAML` and `MarshalYAML` on `Frontmatter` so `ParseNote` / `FormatNote` preserve unknowns and emit deterministic output; update `IsZero`.
- `note/frontmatter_test.go` — add cases for `Type` round-trip, `Extra` round-trip, and the preserved strictness guarantees (non-mapping root, duplicate keys, per-field type errors) under the custom marshalers.
- `note/note.go` — rename `KnownTypes`→`TypesWithSpecialBehavior`, `IsKnownType`→`HasSpecialBehavior`; relax `ParseFilename` to accept any dot-suffix.
- `note/note_test.go` — update `TestIsKnownType`→`TestHasSpecialBehavior`; update/add `ParseFilename` cases.
- `note/store.go` — update call site of `IsKnownType` (ResolveRef type-match step).
- `internal/cli/create.go` — pass `Type` into the `note.Frontmatter{}` literal when building a new note.
- `internal/cli/new.go` — drop validation gate against `IsKnownType`; continue to cache `slug` and `type` in the filename at creation.
- `internal/cli/update.go` — drop validation gate; `Extra` preservation is automatic (through `ParseNote`+`FormatNote`); prefer `fm.Type`/`fm.Slug` as canonical input; stop auto-renaming on content changes; add `--sync-filename` flag.
- `internal/cli/update_test.go` — update for new behavior and new flag.
- `internal/cli/new_test.go` — add cases for `Type` being in frontmatter.
- `CHANGELOG.md` — new `[0.1.73]` entry.

---

## Task 1: Add `Extra` to Frontmatter via custom `UnmarshalYAML` / `MarshalYAML`

**Files:**
- Modify: `note/frontmatter.go`
- Test: `note/frontmatter_test.go`

**Approach.** The `Frontmatter` struct gains an `Extra map[string]yaml.Node` field. Custom `UnmarshalYAML(node *yaml.Node) error` walks the YAML mapping and routes known keys into the typed fields while copying unknowns into `Extra`. Custom `MarshalYAML() (interface{}, error)` composes a `*yaml.Node` mapping with reserved fields first (in fixed order) and then `Extra` alpha-sorted. `ParseNote` and `FormatNote` are untouched — they continue to call `yaml.Unmarshal` / `yaml.Marshal`, which delegate to our custom methods. `IsZero` is rewritten explicitly so it returns `true` only when every reserved field is zero AND `Extra` is empty.

**Strictness preserved.** The existing `TestParseNoteErrors` cases (non-mapping root, duplicate keys, per-field type errors, control characters, alias bomb) must keep passing. Our `UnmarshalYAML` rejects non-mapping nodes explicitly and tracks seen keys to reject duplicates. Per-field decode errors propagate out. Parser-level issues (control chars, alias bomb) continue to be rejected by yaml.v3 before our method is invoked.

- [ ] **Step 1.1: Write the failing parse test — unknown key is captured into Extra**

Append these sub-tests to `note/frontmatter_test.go`. The file currently uses table-driven `TestParseNoteSuccess` / `TestParseNoteErrors` / `TestFormatNote*` etc.; sub-tests can live at the end of the file as standalone `TestXxx` functions. Use standalone functions (not nested `t.Run`) to match the file's existing style:

```go
func TestParseNoteExtraPreservesUnknownKeys(t *testing.T) {
    in := []byte("---\ntitle: T\nfeatured: true\ncustom: hello\n---\n\nbody\n")
    fm, body, err := ParseNote(in)
    if err != nil {
        t.Fatalf("ParseNote: %v", err)
    }
    if fm.Title != "T" {
        t.Errorf("Title = %q, want %q", fm.Title, "T")
    }
    if string(body) != "body\n" {
        t.Errorf("body = %q, want %q", string(body), "body\n")
    }
    if _, ok := fm.Extra["featured"]; !ok {
        t.Error("Extra missing key 'featured'")
    }
    if _, ok := fm.Extra["custom"]; !ok {
        t.Error("Extra missing key 'custom'")
    }
    featuredNode := fm.Extra["featured"]
    var featured bool
    if err := featuredNode.Decode(&featured); err != nil {
        t.Fatalf("decode featured: %v", err)
    }
    if !featured {
        t.Errorf("featured = %v, want true", featured)
    }
}
```

- [ ] **Step 1.2: Run test to verify failure**

Run: `go test ./note/ -run TestParseNoteExtraPreservesUnknownKeys -v`
Expected: compile error — `fm.Extra` doesn't exist on `Frontmatter`.

- [ ] **Step 1.3: Add the `Extra` field to `Frontmatter`**

In `note/frontmatter.go`, modify the struct. The `yaml:"-"` tag keeps the default (un)marshaler from touching it; custom methods will handle Extra explicitly.

```go
type Frontmatter struct {
    Title       string               `yaml:"title,omitempty"`
    Slug        string               `yaml:"slug,omitempty"`
    Tags        []string             `yaml:"tags,omitempty"`
    Description string               `yaml:"description,omitempty"`
    Public      bool                 `yaml:"public,omitempty"`
    Extra       map[string]yaml.Node `yaml:"-"`
}
```

- [ ] **Step 1.4: Rewrite `IsZero` to account for Extra**

In `note/frontmatter.go`, replace the reflection-based `IsZero`:

```go
// IsZero reports whether f has no fields set, including Extra.
func (f Frontmatter) IsZero() bool {
    return f.Title == "" && f.Slug == "" && len(f.Tags) == 0 &&
        f.Description == "" && !f.Public && len(f.Extra) == 0
}
```

Remove the `reflect` import (check all its uses in the file first — `grep -n reflect note/frontmatter.go`).

- [ ] **Step 1.5: Add `UnmarshalYAML` on `*Frontmatter`**

Append to `note/frontmatter.go`. This replaces the implicit struct unmarshaler that yaml.v3 would otherwise use; when `yaml.Unmarshal` is called in `ParseNote`, it now dispatches to this method:

```go
// UnmarshalYAML decodes a mapping node into f. Reserved keys populate the
// typed fields; unknown keys are captured in f.Extra as yaml.Node values.
// Duplicate top-level keys are rejected (matching PR #113's strictness).
func (f *Frontmatter) UnmarshalYAML(node *yaml.Node) error {
    if node.Kind != yaml.MappingNode {
        return fmt.Errorf("frontmatter: expected mapping, got kind %d", node.Kind)
    }
    seen := make(map[string]bool, len(node.Content)/2)
    for i := 0; i+1 < len(node.Content); i += 2 {
        key, value := node.Content[i], node.Content[i+1]
        if seen[key.Value] {
            return fmt.Errorf("frontmatter: duplicate key %q", key.Value)
        }
        seen[key.Value] = true
        switch key.Value {
        case "title":
            if err := value.Decode(&f.Title); err != nil {
                return fmt.Errorf("frontmatter title: %w", err)
            }
        case "slug":
            if err := value.Decode(&f.Slug); err != nil {
                return fmt.Errorf("frontmatter slug: %w", err)
            }
        case "tags":
            if err := value.Decode(&f.Tags); err != nil {
                return fmt.Errorf("frontmatter tags: %w", err)
            }
        case "description":
            if err := value.Decode(&f.Description); err != nil {
                return fmt.Errorf("frontmatter description: %w", err)
            }
        case "public":
            if err := value.Decode(&f.Public); err != nil {
                return fmt.Errorf("frontmatter public: %w", err)
            }
        default:
            if f.Extra == nil {
                f.Extra = make(map[string]yaml.Node)
            }
            f.Extra[key.Value] = *value
        }
    }
    return nil
}
```

- [ ] **Step 1.6: Run the parse test + existing error tests to verify**

Run: `go test ./note/ -run "TestParseNoteExtraPreservesUnknownKeys|TestParseNoteErrors|TestParseNote" -v`
Expected: PASS. In particular, `TestParseNoteErrors` must still pass — including `duplicate keys rejected`, `non-mapping top level`, `invalid bool value`, `alias bomb`. If any of those fail, debug before proceeding.

- [ ] **Step 1.7: Write the failing marshal test — Extra round-trips in alpha order**

Append to `note/frontmatter_test.go`:

```go
func TestFormatNoteExtraPreservedInAlphaOrder(t *testing.T) {
    in := []byte("---\ntitle: T\nzebra: striped\nalpha: 1\nfeatured: true\n---\n\nbody\n")
    fm, body, err := ParseNote(in)
    if err != nil {
        t.Fatalf("ParseNote: %v", err)
    }
    out := string(FormatNote(fm, body))
    // Reserved "title" first; Extra keys alpha-sorted: alpha, featured, zebra.
    want := "---\ntitle: T\nalpha: 1\nfeatured: true\nzebra: striped\n---\n\nbody\n"
    if out != want {
        t.Errorf("FormatNote =\n%q\nwant:\n%q", out, want)
    }
}

func TestFormatNoteEmptyFrontmatterWithExtraOnly(t *testing.T) {
    fm := Frontmatter{Extra: map[string]yaml.Node{
        "featured": {Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
    }}
    want := "---\nfeatured: true\n---\n\nbody\n"
    got := string(FormatNote(fm, []byte("body\n")))
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}

func TestIsZeroIncludesExtra(t *testing.T) {
    if (Frontmatter{}).IsZero() == false {
        t.Error("empty Frontmatter should be zero")
    }
    fm := Frontmatter{Extra: map[string]yaml.Node{
        "featured": {Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
    }}
    if fm.IsZero() {
        t.Error("Frontmatter with Extra should not be zero")
    }
}
```

- [ ] **Step 1.8: Run the marshal tests to verify failure**

Run: `go test ./note/ -run "TestFormatNoteExtra|TestIsZero|TestFormatNoteEmpty" -v`
Expected: `TestFormatNoteExtraPreservedInAlphaOrder` and `TestFormatNoteEmptyFrontmatterWithExtraOnly` FAIL — yaml.v3's default struct marshaler (without `MarshalYAML` defined) emits only the reserved fields, dropping Extra entirely. `TestIsZeroIncludesExtra` passes (IsZero was updated in Step 1.4).

- [ ] **Step 1.9: Add `MarshalYAML` on `Frontmatter`**

Append to `note/frontmatter.go`. This composes a `*yaml.Node` manually so field order is controlled and Extra is emitted alpha-sorted:

```go
// MarshalYAML composes a mapping node with reserved fields first (in fixed
// order) and Extra keys alpha-sorted. Zero-valued reserved fields are omitted,
// matching the `omitempty` struct-tag discipline.
func (f Frontmatter) MarshalYAML() (interface{}, error) {
    node := &yaml.Node{Kind: yaml.MappingNode}

    appendString := func(key, value string) {
        if value == "" {
            return
        }
        valNode := &yaml.Node{}
        if err := valNode.Encode(value); err == nil {
            node.Content = append(node.Content,
                &yaml.Node{Kind: yaml.ScalarNode, Value: key},
                valNode,
            )
        }
    }
    appendList := func(key string, value []string) {
        if len(value) == 0 {
            return
        }
        valNode := &yaml.Node{}
        if err := valNode.Encode(value); err == nil {
            node.Content = append(node.Content,
                &yaml.Node{Kind: yaml.ScalarNode, Value: key},
                valNode,
            )
        }
    }
    appendBool := func(key string, value bool) {
        if !value {
            return
        }
        valNode := &yaml.Node{}
        if err := valNode.Encode(value); err == nil {
            node.Content = append(node.Content,
                &yaml.Node{Kind: yaml.ScalarNode, Value: key},
                valNode,
            )
        }
    }

    appendString("title", f.Title)
    appendString("slug", f.Slug)
    appendList("tags", f.Tags)
    appendString("description", f.Description)
    appendBool("public", f.Public)

    if len(f.Extra) > 0 {
        keys := make([]string, 0, len(f.Extra))
        for k := range f.Extra {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        for _, k := range keys {
            v := f.Extra[k]
            node.Content = append(node.Content,
                &yaml.Node{Kind: yaml.ScalarNode, Value: k},
                &v,
            )
        }
    }

    return node, nil
}
```

Add the `sort` import at the top of the file if not already present.

- [ ] **Step 1.10: Run the full test suite to verify**

Run: `go test ./note/ -v`
Expected: PASS on all existing `TestParseNoteSuccess`, `TestParseNoteErrors`, `TestFormatNoteSnapshotAllFields`, `TestFormatNoteEmptyFrontmatter`, `TestRoundtrip`, and the three new tests from this task.

Watch especially for `TestFormatNoteSnapshotAllFields` — the existing snapshot `"---\ntitle: T\nslug: s\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"` must still match under the new marshaler. The `appendList` helper uses `valNode.Encode([]string{...})`, which yaml.v3 renders in block style with 4-space indent by default, matching the existing snapshot. If your local yaml.v3 renders it differently, surface as a BLOCKED status — something about the encoder options in the existing code differs from what `Encode` gives us.

- [ ] **Step 1.11: Run `make lint`**

Run: `make lint`
Expected: clean.

- [ ] **Step 1.12: Commit**

```bash
git add note/frontmatter.go note/frontmatter_test.go
git commit -m "Preserve unknown frontmatter keys in Frontmatter.Extra"
```

---

## Task 2: Add `Type` as a typed frontmatter field

**Files:**
- Modify: `note/frontmatter.go`
- Test: `note/frontmatter_test.go`

**Approach.** Add `Type string` to the `Frontmatter` struct between `Slug` and `Tags`. Add a `case "type":` branch in `UnmarshalYAML`. Add `appendString("type", f.Type)` in `MarshalYAML` between the slug and tags emitters. Update `IsZero` to check `Type` too.

- [ ] **Step 2.1: Write the failing tests — Type round-trips and field order**

Append to `note/frontmatter_test.go`:

```go
func TestTypeRoundTrips(t *testing.T) {
    in := []byte("---\ntitle: T\ntype: meeting\n---\n\nbody\n")
    fm, body, err := ParseNote(in)
    if err != nil {
        t.Fatalf("ParseNote: %v", err)
    }
    if fm.Type != "meeting" {
        t.Errorf("Type = %q, want meeting", fm.Type)
    }
    out := string(FormatNote(fm, body))
    want := "---\ntitle: T\ntype: meeting\n---\n\nbody\n"
    if out != want {
        t.Errorf("out = %q, want %q", out, want)
    }
}

func TestTypeFieldOrder(t *testing.T) {
    fm := Frontmatter{
        Title: "T", Slug: "s", Type: "meeting",
        Tags: []string{"a"}, Description: "D", Public: true,
    }
    got := string(FormatNote(fm, []byte("body\n")))
    want := "---\ntitle: T\nslug: s\ntype: meeting\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"
    if got != want {
        t.Errorf("FormatNote =\n%q\nwant:\n%q", got, want)
    }
}
```

- [ ] **Step 2.2: Run to verify failure**

Run: `go test ./note/ -run "TestTypeRoundTrips|TestTypeFieldOrder" -v`
Expected: compile error — `Type` field does not exist on `Frontmatter`.

- [ ] **Step 2.3: Add `Type` to `Frontmatter`**

In `note/frontmatter.go`, reorder the struct fields so `Type` sits between `Slug` and `Tags`:

```go
type Frontmatter struct {
    Title       string               `yaml:"title,omitempty"`
    Slug        string               `yaml:"slug,omitempty"`
    Type        string               `yaml:"type,omitempty"`
    Tags        []string             `yaml:"tags,omitempty"`
    Description string               `yaml:"description,omitempty"`
    Public      bool                 `yaml:"public,omitempty"`
    Extra       map[string]yaml.Node `yaml:"-"`
}
```

- [ ] **Step 2.4: Extend `UnmarshalYAML` to decode `type`**

Add a `case "type":` branch in the switch, next to the other scalar-string cases:

```go
case "type":
    if err := value.Decode(&f.Type); err != nil {
        return fmt.Errorf("frontmatter type: %w", err)
    }
```

- [ ] **Step 2.5: Extend `MarshalYAML` to emit `type`**

Insert `appendString("type", f.Type)` between the `slug` and `tags` emitters, matching the field order:

```go
appendString("title", f.Title)
appendString("slug", f.Slug)
appendString("type", f.Type)
appendList("tags", f.Tags)
appendString("description", f.Description)
appendBool("public", f.Public)
```

- [ ] **Step 2.6: Update `IsZero` to include `Type`**

```go
func (f Frontmatter) IsZero() bool {
    return f.Title == "" && f.Slug == "" && f.Type == "" && len(f.Tags) == 0 &&
        f.Description == "" && !f.Public && len(f.Extra) == 0
}
```

- [ ] **Step 2.7: Run tests to verify pass**

Run: `go test ./note/ -v`
Expected: PASS on all existing and new cases.

- [ ] **Step 2.8: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 2.9: Commit**

```bash
git add note/frontmatter.go note/frontmatter_test.go
git commit -m "Add Type to Frontmatter"
```

---

## Task 3: Rename `KnownTypes` → `TypesWithSpecialBehavior` and `IsKnownType` → `HasSpecialBehavior`

**Files:**
- Modify: `note/note.go`
- Modify: `note/note_test.go`
- Modify: `note/store.go`
- Modify: `internal/cli/new.go`
- Modify: `internal/cli/update.go`

- [ ] **Step 3.1: Rewrite the test `TestIsKnownType` as `TestHasSpecialBehavior`**

In `note/note_test.go`, replace the entire `TestIsKnownType` function:

```go
func TestHasSpecialBehavior(t *testing.T) {
    if !HasSpecialBehavior("todo") {
        t.Error("expected todo to have special behavior")
    }
    if !HasSpecialBehavior("backlog") {
        t.Error("expected backlog to have special behavior")
    }
    if !HasSpecialBehavior("weekly") {
        t.Error("expected weekly to have special behavior")
    }
    if HasSpecialBehavior("random") {
        t.Error("expected random to have no special behavior")
    }
    if HasSpecialBehavior("") {
        t.Error("expected empty string to have no special behavior")
    }
}
```

- [ ] **Step 3.2: Run to verify failure**

Run: `go test ./note/ -run TestHasSpecialBehavior -v`
Expected: FAIL / compile error — `HasSpecialBehavior` does not yet exist.

- [ ] **Step 3.3: Rename in `note/note.go`**

Replace the top of the file:

```go
// TypesWithSpecialBehavior lists note types that trigger notesctl-specific
// handling (e.g., daily rollover, weekly review conventions). Any string is a
// valid `type` value; this list is a soft registry, not a validation gate.
var TypesWithSpecialBehavior = []string{"todo", "backlog", "weekly"}

// HasSpecialBehavior reports whether s is a type with special notesctl behavior.
func HasSpecialBehavior(s string) bool {
    for _, t := range TypesWithSpecialBehavior {
        if s == t {
            return true
        }
    }
    return false
}
```

Also update the internal call inside `ParseFilename` that currently calls `IsKnownType(suffix)` — change it to `HasSpecialBehavior(suffix)` for now (it will be removed entirely in Task 4, but keeping the rename mechanical here).

- [ ] **Step 3.4: Update call sites in the repo**

Search for stragglers:

Run: `grep -rn "IsKnownType\|KnownTypes" --include='*.go' .`

Update each:
- `note/store.go:101` — `if IsKnownType(query)` → `if HasSpecialBehavior(query)`
- `internal/cli/new.go:29-30` — `!note.IsKnownType(noteType)` → `!note.HasSpecialBehavior(noteType)`; `strings.Join(note.KnownTypes, ", ")` → `strings.Join(note.TypesWithSpecialBehavior, ", ")`
- `internal/cli/update.go:46-47` — same rename as `new.go`.

These validation gates will be removed entirely in subsequent tasks. For this commit, they stay but use the new names.

- [ ] **Step 3.5: Run all tests to verify pass**

Run: `go test ./... -v`
Expected: PASS across all packages.

- [ ] **Step 3.6: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 3.7: Commit**

```bash
git add note/note.go note/note_test.go note/store.go internal/cli/new.go internal/cli/update.go
git commit -m "Rename KnownTypes to TypesWithSpecialBehavior; IsKnownType to HasSpecialBehavior"
```

---

## Task 4: Relax `ParseFilename` to accept any dot-suffix as filename-reported `Type`

**Files:**
- Modify: `note/note.go`
- Modify: `note/note_test.go`

- [ ] **Step 4.1: Update existing `TestParseFilename` cases**

In `note/note_test.go`, find the `"unknown dot suffix treated as slug"` test case (it currently expects `wantSlug: "foo.bar", wantType: ""`). Replace with:

```go
{
    name:         "unknown dot suffix treated as filename-reported type",
    input:        "20260312_9219_foo.bar",
    wantDate:     "20260312",
    wantID:       "9219",
    wantSlug:     "foo",
    wantType:     "bar",
    wantBaseName: "20260312_9219_foo",
},
```

Add a new case for a custom-named type:

```go
{
    name:         "custom type name (no registry gate)",
    input:        "20260106_8823.meeting",
    wantDate:     "20260106",
    wantID:       "8823",
    wantSlug:     "",
    wantType:     "meeting",
    wantBaseName: "20260106_8823",
},
```

- [ ] **Step 4.2: Run to verify failure**

Run: `go test ./note/ -run TestParseFilename -v`
Expected: FAIL — current logic still gates on `HasSpecialBehavior`, so `foo.bar` → slug="foo.bar" type="" and `meeting` is not recognized.

- [ ] **Step 4.3: Drop the `HasSpecialBehavior` gate inside `ParseFilename`**

In `note/note.go`, find the block:

```go
// Check for known type as a dot-suffix, e.g. "20260102_8814.todo"
if idx := strings.LastIndex(baseName, "."); idx >= 0 {
    suffix := baseName[idx+1:]
    if HasSpecialBehavior(suffix) {
        noteType = suffix
        remaining = baseName[:idx]
    }
}
```

Replace with:

```go
// Any single dot-suffix on the base name is treated as a filename-reported
// type (a fast-path hint used by scan-based filters). No registry gate here:
// any string is accepted. Frontmatter `type` is canonical when available.
if idx := strings.LastIndex(baseName, "."); idx >= 0 {
    noteType = baseName[idx+1:]
    remaining = baseName[:idx]
}
```

- [ ] **Step 4.4: Run tests to verify pass**

Run: `go test ./note/ -v`
Expected: PASS on all cases.

- [ ] **Step 4.5: Run the full suite — the `store.go` resolve-by-type path still uses `HasSpecialBehavior`, so `notes resolve todo` continues to work; other callers (filters) rely on `Note.Type` which is now populated by any dot-suffix.**

Run: `go test ./... -v`
Expected: PASS.

- [ ] **Step 4.6: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 4.7: Commit**

```bash
git add note/note.go note/note_test.go
git commit -m "Relax ParseFilename to accept any dot-suffix as filename-reported type"
```

---

## Task 5: Update `notes new` and `createNote` to write `Type` into frontmatter and drop the validation gate

**Files:**
- Modify: `internal/cli/create.go`
- Modify: `internal/cli/new.go`
- Test: `internal/cli/new_test.go`

- [ ] **Step 5.1: Write the failing test — `notes new --type meeting` succeeds and writes `type: meeting` to frontmatter**

The test harness in `internal/cli/new_test.go` uses a `runNew(t, root, stdin, ...args)` helper and `copyTestdata(t)` (defined in `append_test.go`). Add these two cases at the end of `new_test.go`, matching the existing style:

```go
func TestNewWithCustomType(t *testing.T) {
    root := copyTestdata(t)
    out, err := runNew(t, root, "", "--type", "meeting", "--slug", "sync")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(filepath.Base(out), "sync.meeting.md") {
        t.Errorf("expected slug+type cache in filename, got %q", filepath.Base(out))
    }
    data, err := os.ReadFile(out)
    if err != nil {
        t.Fatalf("read: %v", err)
    }
    if !strings.Contains(string(data), "type: meeting") {
        t.Errorf("expected type: meeting in frontmatter, got:\n%s", string(data))
    }
}

func TestNewWithKnownTypeStillWrites(t *testing.T) {
    root := copyTestdata(t)
    out, err := runNew(t, root, "", "--type", "todo")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.HasSuffix(filepath.Base(out), ".todo.md") {
        t.Errorf("expected .todo.md suffix, got %q", filepath.Base(out))
    }
    data, _ := os.ReadFile(out)
    if !strings.Contains(string(data), "type: todo") {
        t.Errorf("expected type: todo in frontmatter, got:\n%s", string(data))
    }
}
```

Also update the harness at the top of `new_test.go`: `runNew` currently resets flags with `newCmd.ResetFlags()` and re-registers them. When Task 5 removes the validation gate and adds no new flags, this block does not need to change. When it does need changes in later tasks (e.g., Task 6 for `update`), follow the same ResetFlags + re-register pattern.

- [ ] **Step 5.2: Run to verify failure**

Run: `go test ./internal/cli/ -run TestNew -v` (or the specific test name your harness uses)
Expected: FAIL — either the validation gate rejects `"meeting"` with "unknown note type", or `fm.Type` is empty because `createNote` doesn't set it.

- [ ] **Step 5.3: Drop the validation gate in `new.go`**

In `internal/cli/new.go`, remove lines 29–31:

```go
if noteType != "" && !note.HasSpecialBehavior(noteType) {
    return fmt.Errorf("unknown note type %q (valid types: %s)", noteType, strings.Join(note.TypesWithSpecialBehavior, ", "))
}
```

Also remove the now-unused `strings` import if it isn't used elsewhere in the file.

- [ ] **Step 5.4: Include `Type` in the frontmatter in `createNote`**

In `internal/cli/create.go`, update the `note.Frontmatter{}` literal passed to `FormatNote`. The current code (as of `b30f330`):

```go
fm := note.Frontmatter{
    Title:       p.Title,
    Slug:        p.Slug,
    Tags:        p.Tags,
    Description: p.Description,
    Public:      p.Public,
}
content := note.FormatNote(fm, []byte(p.Body))
```

Add `Type: p.Type` between `Slug` and `Tags`:

```go
fm := note.Frontmatter{
    Title:       p.Title,
    Slug:        p.Slug,
    Type:        p.Type,
    Tags:        p.Tags,
    Description: p.Description,
    Public:      p.Public,
}
content := note.FormatNote(fm, []byte(p.Body))
```

The `Type` field in `createNoteParams` already exists; it was only being written to the filename, not the frontmatter.

- [ ] **Step 5.5: Run tests to verify pass**

Run: `go test ./internal/cli/ -v`
Expected: PASS.

- [ ] **Step 5.6: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 5.7: Commit**

```bash
git add internal/cli/new.go internal/cli/create.go internal/cli/new_test.go
git commit -m "notes new: write Type to frontmatter and accept free-form type values"
```

---

## Task 6: Update `notes update` — preserve Extra, stop auto-renaming, add `--sync-filename`

**Files:**
- Modify: `internal/cli/update.go`
- Test: `internal/cli/update_test.go`

Before adding new tests, the harness `runUpdate` helper at the top of `update_test.go` needs the new flag registered. Update it in the same commit as the tests so the harness stays in sync (ResetFlags + re-register pattern):

```go
func runUpdate(t *testing.T, root string, args ...string) (string, error) {
    t.Helper()

    updateCmd.ResetFlags()
    updateCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable); replaces existing tags")
    updateCmd.Flags().Bool("no-tags", false, "remove all tags from frontmatter")
    updateCmd.Flags().String("title", "", "title for frontmatter (empty string clears it)")
    updateCmd.Flags().String("description", "", "description for frontmatter (empty string clears it)")
    updateCmd.Flags().String("slug", "", "update slug in frontmatter; does not rename the file")
    updateCmd.Flags().Bool("no-slug", false, "remove slug from frontmatter")
    updateCmd.Flags().String("type", "", "update type in frontmatter; does not rename the file")
    updateCmd.Flags().Bool("no-type", false, "remove type from frontmatter")
    updateCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
    updateCmd.Flags().Bool("private", false, "mark note as private in frontmatter")
    updateCmd.Flags().Bool("sync-filename", false, "rename the file to match the frontmatter's slug/type cache")
    updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
    updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
    updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
    updateCmd.MarkFlagsMutuallyExclusive("public", "private")

    buf := new(bytes.Buffer)
    rootCmd.SetOut(buf)
    rootCmd.SetErr(buf)
    rootCmd.SetArgs(append([]string{"update", "--path", root}, args...))

    err := rootCmd.Execute()
    return strings.TrimSpace(buf.String()), err
}
```

- [ ] **Step 6.1: Update existing tests that assert the old auto-rename behavior**

Four existing tests in `update_test.go` assert the removed auto-rename semantics. Rewrite each to match the new contract (frontmatter change without filename change) OR rename + pass `--sync-filename`. Minimally invasive approach: keep the tests' original intent (did the rename happen?) but update the expectation.

Replace `TestUpdateSlugRenamesFile`:

```go
// TestUpdateSlugChangesFrontmatterOnly: --slug rewrites frontmatter but leaves filename.
func TestUpdateSlugChangesFrontmatterOnly(t *testing.T) {
    root := copyTestdata(t)
    origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

    out, err := runUpdate(t, root, "8823", "--slug", "renamed")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if out != origPath {
        t.Errorf("got path %q, want %q (should not rename)", out, origPath)
    }
    if _, err := os.Stat(origPath); err != nil {
        t.Errorf("original file missing: %v", err)
    }
    data, _ := os.ReadFile(origPath)
    if !strings.Contains(string(data), "slug: renamed") {
        t.Errorf("expected updated slug in frontmatter, got:\n%s", string(data))
    }
}

// TestUpdateSlugWithSyncFilenameRenames: --slug + --sync-filename renames the file.
func TestUpdateSlugWithSyncFilenameRenames(t *testing.T) {
    root := copyTestdata(t)
    out, err := runUpdate(t, root, "8823", "--slug", "renamed", "--sync-filename")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    want := filepath.Join(root, "2026/01/20260106_8823_renamed.md")
    if out != want {
        t.Errorf("got path %q, want %q", out, want)
    }
    if _, err := os.Stat(want); err != nil {
        t.Errorf("new file does not exist: %v", err)
    }
    if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823_999.md")); err == nil {
        t.Error("old file should have been removed")
    }
}
```

Apply the same transformation to `TestUpdateNoSlugRemovesSlugFromFilename` → `TestUpdateNoSlugClearsSlugFromFrontmatter` + `TestUpdateNoSlugWithSyncFilenameRenames`:

```go
func TestUpdateNoSlugClearsSlugFromFrontmatter(t *testing.T) {
    root := copyTestdata(t)
    origPath := filepath.Join(root, "2026/01/20260104_8818_meeting.md")

    out, err := runUpdate(t, root, "8818", "--no-slug")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if out != origPath {
        t.Errorf("got path %q, want %q (should not rename)", out, origPath)
    }
    data, _ := os.ReadFile(origPath)
    if strings.Contains(string(data), "slug:") {
        t.Errorf("expected slug removed, got:\n%s", string(data))
    }
}

func TestUpdateNoSlugWithSyncFilenameRenames(t *testing.T) {
    root := copyTestdata(t)
    out, err := runUpdate(t, root, "8818", "--no-slug", "--sync-filename")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    want := filepath.Join(root, "2026/01/20260104_8818.md")
    if out != want {
        t.Errorf("got path %q, want %q", out, want)
    }
    if _, err := os.Stat(filepath.Join(root, "2026/01/20260104_8818_meeting.md")); err == nil {
        t.Error("old file should have been removed")
    }
}
```

Apply the same transformation to `TestUpdateTypeRenamesFile` → `TestUpdateTypeChangesFrontmatterOnly` + `TestUpdateTypeWithSyncFilenameRenames`, and to `TestUpdateNoTypeRemovesTypeSuffix` → `TestUpdateNoTypeClearsTypeFromFrontmatter` + `TestUpdateNoTypeWithSyncFilenameRenames`. The pattern is identical; substitute `--slug`→`--type`, `renamed`→`todo`, etc.

- [ ] **Step 6.2: Add the new Extra-preservation test**

```go
// TestUpdatePreservesExtraFields ensures unknown frontmatter keys survive an update.
func TestUpdatePreservesExtraFields(t *testing.T) {
    root := copyTestdata(t)
    // Pick an existing fixture note, overwrite its content to include custom keys.
    notePath := filepath.Join(root, "2026/01/20260106_8823_999.md")
    seed := "---\ntitle: Original\nfeatured: true\ncustom_rating: 5\n---\n\nbody\n"
    if err := os.WriteFile(notePath, []byte(seed), 0o644); err != nil {
        t.Fatal(err)
    }

    if _, err := runUpdate(t, root, "8823", "--title", "New Title"); err != nil {
        t.Fatalf("update: %v", err)
    }

    data, _ := os.ReadFile(notePath)
    if !strings.Contains(string(data), "title: New Title") {
        t.Errorf("expected new title, got:\n%s", string(data))
    }
    if !strings.Contains(string(data), "featured: true") {
        t.Errorf("featured dropped, got:\n%s", string(data))
    }
    if !strings.Contains(string(data), "custom_rating: 5") {
        t.Errorf("custom_rating dropped, got:\n%s", string(data))
    }
}
```

- [ ] **Step 6.3: Add the `--sync-filename`-only (idempotent) test**

```go
// TestUpdateSyncFilenameOnly reconciles filename without any content flags.
func TestUpdateSyncFilenameOnly(t *testing.T) {
    root := copyTestdata(t)
    // Seed a note whose frontmatter slug disagrees with its filename.
    dir := filepath.Join(root, "2026", "01")
    origPath := filepath.Join(dir, "20260106_8823_999.md")
    seed := "---\ntitle: T\nslug: my-slug\ntype: meeting\n---\n\nbody\n"
    if err := os.WriteFile(origPath, []byte(seed), 0o644); err != nil {
        t.Fatal(err)
    }

    out, err := runUpdate(t, root, "8823", "--sync-filename")
    if err != nil {
        t.Fatalf("--sync-filename: %v", err)
    }
    want := filepath.Join(dir, "20260106_8823_my-slug.meeting.md")
    if out != want {
        t.Errorf("got path %q, want %q", out, want)
    }
    if _, err := os.Stat(want); err != nil {
        t.Errorf("new file missing: %v", err)
    }
    if _, err := os.Stat(origPath); !os.IsNotExist(err) {
        t.Errorf("old file should be gone: err=%v", err)
    }
}

// TestUpdateSyncFilenameNoOp: --sync-filename on an already-in-sync note is a no-op.
func TestUpdateSyncFilenameNoOp(t *testing.T) {
    root := copyTestdata(t)
    origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")
    // The fixture has empty fm slug; the filename-reported slug is "999" and
    // fills in as fallback. Running sync should produce the same filename.
    out, err := runUpdate(t, root, "8823", "--sync-filename")
    if err != nil {
        t.Fatalf("--sync-filename: %v", err)
    }
    if out != origPath {
        t.Errorf("expected no rename, got path %q", out)
    }
    if _, err := os.Stat(origPath); err != nil {
        t.Errorf("file moved or lost: %v", err)
    }
}
```

The fixture `testdata/2026/01/20260106_8823_999.md` has `tags: [work]` but no `slug` or `type` in frontmatter. Because frontmatter is empty for those fields, the update path falls back to the filename-reported values (`slug = "999"`, `type = ""`), producing the same filename on sync.

- [ ] **Step 6.4: Run the tests to verify failure**

Run: `go test ./internal/cli/ -run TestUpdate -v`
Expected: FAIL across the rewritten and new tests — current `update` still auto-renames, drops Extra, and the `--sync-filename` flag does not exist. Some of the harness's own `ResetFlags` re-registration will also fail to compile until Step 6.5 lands `--sync-filename` on `updateCmd`.

If the harness re-registration block (updated earlier in this task) fails to compile because `--sync-filename` is registered but `updateCmd` has no such flag after execution, land Step 6.5 next so the command definition matches.

- [ ] **Step 6.5: Rewrite the `update` command**

Replace `internal/cli/update.go` in its entirety:

```go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"

    "github.com/dreikanter/notesctl/note"
    "github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
    Use:   "update <id|type|query>",
    Short: "Update frontmatter; use --sync-filename to reconcile the filename",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        updateTags, _ := cmd.Flags().GetStringSlice("tag")
        updateNoTags, _ := cmd.Flags().GetBool("no-tags")
        updateTitle, _ := cmd.Flags().GetString("title")
        updateDescription, _ := cmd.Flags().GetString("description")
        updateSlug, _ := cmd.Flags().GetString("slug")
        updateNoSlug, _ := cmd.Flags().GetBool("no-slug")
        updateType, _ := cmd.Flags().GetString("type")
        updateNoType, _ := cmd.Flags().GetBool("no-type")
        updatePrivate, _ := cmd.Flags().GetBool("private")
        syncFilename, _ := cmd.Flags().GetBool("sync-filename")

        updateFlags := []string{
            "tag", "no-tags", "title", "description",
            "slug", "no-slug", "type", "no-type",
            "public", "private", "sync-filename",
        }
        hasFlag := false
        for _, name := range updateFlags {
            if cmd.Flags().Changed(name) {
                hasFlag = true
                break
            }
        }
        if !hasFlag {
            return fmt.Errorf("at least one update flag is required")
        }

        if cmd.Flags().Changed("slug") {
            if err := note.ValidateSlug(updateSlug); err != nil {
                return err
            }
        }

        root := mustNotesPath()
        n, err := note.ResolveRef(root, args[0])
        if err != nil {
            return err
        }

        oldPath := filepath.Join(root, n.RelPath)
        data, err := os.ReadFile(oldPath)
        if err != nil {
            return fmt.Errorf("cannot read note: %w", err)
        }

        existing, body, err := note.ParseNote(data)
        if err != nil {
            return fmt.Errorf("%s: %w", oldPath, err)
        }

        // frontmatter is canonical; filename values are fallbacks only.
        if existing.Slug == "" {
            existing.Slug = n.Slug
        }
        if existing.Type == "" {
            existing.Type = n.Type
        }

        updated := existing // includes preserved Extra

        if cmd.Flags().Changed("title") {
            updated.Title = updateTitle
        }
        if cmd.Flags().Changed("description") {
            updated.Description = updateDescription
        }
        if updateNoTags {
            updated.Tags = nil
        } else if cmd.Flags().Changed("tag") {
            updated.Tags = updateTags
        }

        if updateNoSlug {
            updated.Slug = ""
        } else if cmd.Flags().Changed("slug") {
            updated.Slug = updateSlug
        }
        if updatePrivate {
            updated.Public = false
        } else if cmd.Flags().Changed("public") {
            updated.Public = true
        }
        if updateNoType {
            updated.Type = ""
        } else if cmd.Flags().Changed("type") {
            updated.Type = updateType
        }

        // Any non-sync flag ⇒ rewrite the frontmatter in place (no rename).
        contentChanged := cmd.Flags().Changed("title") ||
            cmd.Flags().Changed("description") ||
            cmd.Flags().Changed("tag") || updateNoTags ||
            cmd.Flags().Changed("slug") || updateNoSlug ||
            cmd.Flags().Changed("type") || updateNoType ||
            cmd.Flags().Changed("public") || updatePrivate

        if contentChanged {
            newContent := note.FormatNote(updated, body)
            if err := writeAtomic(oldPath, newContent); err != nil {
                return err
            }
        }

        // --sync-filename: reconcile filename to match (already-updated) frontmatter.
        newPath := oldPath
        if syncFilename {
            id, _ := strconv.Atoi(n.ID)
            newFilename := note.NoteFilename(n.Date, id, updated.Slug, updated.Type)
            dir := filepath.Dir(oldPath)
            newPath = filepath.Join(dir, newFilename)
            if newPath != oldPath {
                if err := os.Rename(oldPath, newPath); err != nil {
                    return fmt.Errorf("cannot rename note: %w", err)
                }
            }
        }

        fmt.Fprintln(cmd.OutOrStdout(), newPath)
        return nil
    },
}

// writeAtomic writes data to path via a tmp+rename so partial writes don't
// leave a corrupted file behind.
func writeAtomic(path string, data []byte) error {
    tmpPath := path + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
        return fmt.Errorf("cannot write note: %w", err)
    }
    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("cannot replace note: %w", err)
    }
    return nil
}

func init() {
    updateCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable); replaces existing tags")
    updateCmd.Flags().Bool("no-tags", false, "remove all tags from frontmatter")
    updateCmd.Flags().String("title", "", "title for frontmatter (empty string clears it)")
    updateCmd.Flags().String("description", "", "description for frontmatter (empty string clears it)")
    updateCmd.Flags().String("slug", "", "update slug in frontmatter; does not rename the file")
    updateCmd.Flags().Bool("no-slug", false, "remove slug from frontmatter")
    updateCmd.Flags().String("type", "", "update type in frontmatter; does not rename the file")
    updateCmd.Flags().Bool("no-type", false, "remove type from frontmatter")
    updateCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
    updateCmd.Flags().Bool("private", false, "mark note as private in frontmatter")
    updateCmd.Flags().Bool("sync-filename", false, "rename the file to match the frontmatter's slug/type cache")
    updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
    updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
    updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
    updateCmd.MarkFlagsMutuallyExclusive("public", "private")
    rootCmd.AddCommand(updateCmd)
}
```

- [ ] **Step 6.6: Run all update tests and any touching tests to verify pass**

Run: `go test ./internal/cli/ -v`
Expected: PASS on all four new cases and all existing update/new cases.

If existing tests assert the old auto-rename behavior (likely — `update --slug` previously renamed), update them to match the new contract: they either no longer expect a rename, or they pass `--sync-filename`. Adjust line-by-line; keep the assertion's spirit (did the user's intent happen?), not the letter (the file used to move).

- [ ] **Step 6.7: Run the full suite**

Run: `go test ./... -v`
Expected: PASS.

- [ ] **Step 6.8: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 6.9: Commit**

```bash
git add internal/cli/update.go internal/cli/update_test.go
git commit -m "notes update: preserve Extra, stop auto-renaming, add --sync-filename"
```

---

## Task 7: Add `SCHEMA.md` at repo root

**Files:**
- Create: `SCHEMA.md`

- [ ] **Step 7.1: Create the file**

Write `SCHEMA.md` at the repo root with this content:

```markdown
# Note frontmatter schema

Reserved keys are the fields declared on `Frontmatter` in
`github.com/dreikanter/notesctl/note`. Any key not listed below is
preserved verbatim on read/write (stored in `Frontmatter.Extra`)
and ignored by notesctl itself.

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
- **Consumers:** notesctl (`new`, `update --sync-filename`),
  notes-pub (URL path segment).

### type
- **Type:** string
- **Semantics:** note category. Any value is valid. A small set of
  values (`todo`, `backlog`, `weekly`) trigger special notesctl
  behavior; see `note.TypesWithSpecialBehavior`. The filename may
  carry a cached copy as a `.type` dot-suffix; on mismatch,
  frontmatter wins.
- **Consumers:** notesctl (filters, rollover), notes-pub / notes-view
  (optional rendering).

### tags
- **Type:** list of strings
- **Semantics:** free-form tags, matched case-sensitively. In-body
  `#hashtag` usage is a separate feature not governed by this field.
- **Consumers:** notesctl (`tags`, filters), notes-pub (tag pages,
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

Any other top-level key is preserved untouched by notesctl. Nested
structures (mappings, sequences) are preserved intact.

Duplicate top-level keys are rejected at the document level (per
PR #113). Non-string keys and anchors/aliases in the YAML tree
are preserved inside `Extra` values as-is but are not specifically
tested in notesctl; use at your own risk.

## Process

Adding a key to `Frontmatter` requires updating this file in
the same PR. `CHANGELOG.md` entries reference both the PR and the
new schema entry.
```

- [ ] **Step 7.2: Commit**

```bash
git add SCHEMA.md
git commit -m "Add SCHEMA.md documenting reserved frontmatter keys"
```

---

## Task 8: Update `CHANGELOG.md`

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 8.1: Determine the next version**

Run: `git describe --tags` and inspect the top of `CHANGELOG.md` — the highest-numbered `[0.1.x]` heading plus one is the next version. As of `b30f330` the CHANGELOG top entry is `[0.1.72]`, so this plan's entry is `[0.1.73]`. If the changelog has moved further, use the actual next patch.

- [ ] **Step 8.2: Add a `[0.1.73]` entry at the top**

Insert above the existing `## [0.1.72]` heading in `CHANGELOG.md`:

```markdown
## [0.1.73] - 2026-04-19

### Changed

- Note frontmatter format: unknown keys are now preserved through `notes update` and any other format-rewriting command (via `Frontmatter.Extra`), enabling downstream tools and users to add custom fields without waiting for a notesctl release. `type` moves from filename-only to a typed frontmatter field (filename still cached as a `.type` dot-suffix). `KnownTypes`/`IsKnownType` renamed to `TypesWithSpecialBehavior`/`HasSpecialBehavior` — the list is now a soft registry, not a validation gate; any string is a valid `type` value. `notes update` no longer auto-renames on `--slug`/`--type` changes; use the new `--sync-filename` flag to explicitly reconcile the filename with frontmatter. A repo-root `SCHEMA.md` documents reserved frontmatter keys. See [design spec](docs/superpowers/specs/2026-04-19-notes-schema-protocol-design.md) and [#104]. ([#TBD])
```

(Replace `#TBD` with the PR number once the PR is opened; this is a plan-time placeholder.)

Also add a link-reference line at the bottom of the file for the new PR, matching the `[#110]: https://github.com/...` style.

- [ ] **Step 8.3: Verify the format matches existing entries**

Run: `head -15 CHANGELOG.md`
Expected: the new `## [0.1.73]` entry sits at the top, in the same shape as `## [0.1.72]`.

- [ ] **Step 8.4: Lint / test (sanity)**

Run: `make test`
Expected: PASS.

- [ ] **Step 8.5: Commit**

```bash
git add CHANGELOG.md
git commit -m "CHANGELOG: v0.1.73 — schema protocol"
```

---

## Task 9: Final verification

- [ ] **Step 9.1: Full test run**

Run: `go test ./...`
Expected: PASS across all packages.

- [ ] **Step 9.2: Lint**

Run: `make lint`
Expected: clean.

- [ ] **Step 9.3: Build**

Run: `make build`
Expected: builds cleanly; `./notes --help` lists `update`, and `./notes update --help` shows the new `--sync-filename` flag.

- [ ] **Step 9.4: Smoke-test the contract manually**

```bash
mkdir -p /tmp/schema-smoke/2026/04
cat > /tmp/schema-smoke/2026/04/20260419_9999_sample.md <<'EOF'
---
title: Sample
featured: true
aliases:
    - alt-one
    - alt-two
---

body
EOF

NOTESCTL_PATH=/tmp/schema-smoke ./notes update --title "Sample 2" 9999
cat /tmp/schema-smoke/2026/04/20260419_9999_sample.md
```

Expected: `title: Sample 2` is present; `featured: true` and the `aliases` list are still present.

```bash
NOTESCTL_PATH=/tmp/schema-smoke ./notes update --slug renamed --sync-filename 9999
ls /tmp/schema-smoke/2026/04/
```

Expected: file is now `20260419_9999_renamed.md`; previous name is gone.

- [ ] **Step 9.5: Push the branch and open a PR**

Follow the repo's PR convention (`.github/pull_request_template.md`); title something like "Notes schema protocol: extensible frontmatter and explicit filename sync (#104)". PR body should link to the design spec and `SCHEMA.md`, reference issue #104, and point out the behavior change on `notes update` (no more auto-rename).

Then update `CHANGELOG.md` to replace `#TBD` with the actual PR number, and push the follow-up commit.

---

## Self-review checklist

Run through the spec once the plan is complete:

- [x] **Data model — `Type` and `Extra` added.** Covered by Task 1 (Extra via custom marshalers) and Task 2 (Type).
- [x] **Parser preserves unknowns.** Task 1 Step 1.5 (`UnmarshalYAML` default case).
- [x] **Writer emits reserved fields first, then Extra alpha-sorted.** Task 1 Step 1.9 (`MarshalYAML` composes a node).
- [x] **KnownTypes renamed to TypesWithSpecialBehavior (soft registry, not a gate).** Task 3, Task 5 (drops gate in `new`), Task 6 (drops gate in `update`).
- [x] **ParseFilename accepts any dot-suffix.** Task 4.
- [x] **notes new writes Type to fm; always caches slug/type in filename.** Task 5.
- [x] **notes update no longer auto-renames.** Task 6 (`contentChanged` branch writes in place).
- [x] **notes update preserves Extra.** Task 6 (`updated := existing` carries `Extra`).
- [x] **`--sync-filename` flag added, prints new path, idempotent.** Task 6.
- [x] **fm canonical when both fm and filename have slug/type.** Task 6 Step 6.5 (fills `existing.Slug` from `n.Slug` only when fm has none, so fm always wins when present).
- [x] **SCHEMA.md present, reserved keys documented.** Task 7.
- [x] **CHANGELOG entry.** Task 8.
- [x] **No migration command, no backwards-compat shims.** Not introduced anywhere.
- [x] **All steps have concrete code or concrete commands — no placeholders.** Reviewed above; only `#TBD` is marked explicitly as a PR-time placeholder.
