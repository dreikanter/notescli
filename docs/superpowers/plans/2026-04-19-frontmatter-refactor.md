# Frontmatter API Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `ParseFrontmatterFields` / `BuildFrontmatter` / `StripFrontmatter` trio with an error-returning `ParseNote` / `FormatNote` pair, rename `FrontmatterFields` → `Frontmatter`, add `Frontmatter.IsZero`, collapse call-site boilerplate, and clean up adjacent code (`ExtractTags` concurrency, CRLF handling, `frontmatterDelim` constness).

**Architecture:** The `note` package gains a focused `ParseNote(data) (Frontmatter, body, error)` primitive built on `yaml.Unmarshal` into the struct directly (no field switch, no per-field tolerance). `FormatNote(f, body) []byte` pairs it. Bulk readers (`FilterByTags`, `ExtractTags`) log warn-and-continue via `log.Printf` on a note-level parse error; single-note writers (`update`, `annotate`) surface the error. `StripFrontmatter` is kept for `read -F` (body-only, no parse needed). The existing `findFrontmatterBlock` is replaced by an offset-returning helper that slices `data` directly, preserving CRLF interior bytes. `ExtractTags` is rewritten using `golang.org/x/sync/errgroup`.

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v3`, `golang.org/x/sync/errgroup` (already in indirect deps — needs to become direct), `log` (stdlib).

---

## File Structure

- `note/frontmatter.go` — full rewrite: `Frontmatter`, `IsZero`, `ParseNote`, `FormatNote`, `StripFrontmatter`, internal `frontmatterEnd`.
- `note/frontmatter_test.go` — consolidated table tests; one byte-exact snapshot; round-trip tests; CRLF coverage; adversarial kept.
- `note/store.go` — `FilterByTags` switches to `ParseNote`, logs warn on per-note parse error, continues.
- `note/tags.go` — `ExtractTags` rewritten using `errgroup`; switches to `ParseNote`, logs warn on per-note parse error, continues.
- `internal/cli/update.go` — call-site collapsed to `ParseNote` / `FormatNote`; surfaces parse error.
- `internal/cli/annotate.go` — same collapse; surfaces parse error; `annotateEmptyFields` and `mergeAnnotation` take `note.Frontmatter`.
- `internal/cli/create.go` — `createNote` uses `FormatNote` directly.
- `internal/cli/read.go` — unchanged; still uses `StripFrontmatter`.
- `internal/cli/annotate_test.go` — `FrontmatterFields` → `Frontmatter`.
- `CHANGELOG.md` — entry for v0.1.72.
- `go.mod` — promote `golang.org/x/sync` to a direct require.

---

## Proposed Public API

```go
// Frontmatter holds optional fields for note frontmatter.
type Frontmatter struct {
    Title       string   `yaml:"title,omitempty"`
    Slug        string   `yaml:"slug,omitempty"`
    Tags        []string `yaml:"tags,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Public      bool     `yaml:"public,omitempty"`
}

// IsZero reports whether f has no fields set.
func (f Frontmatter) IsZero() bool {
    return reflect.ValueOf(f).IsZero()
}

// ParseNote splits a note file into its frontmatter and body.
func ParseNote(data []byte) (Frontmatter, []byte, error)

// FormatNote serialises frontmatter followed by body.
func FormatNote(f Frontmatter, body []byte) []byte

// StripFrontmatter returns the body portion only; used by `read -F`.
func StripFrontmatter(data []byte) []byte
```

Semantics:
- `ParseNote` returns `(zero, data, nil)` when no frontmatter block is present.
- `ParseNote` returns `(zero, nil, err)` when a frontmatter block is present but malformed (delimiter pair mismatch, non-mapping document, yaml-unmarshal error, type mismatch).
- `FormatNote` emits `"---\n<yaml>---\n\n" + body` when `!f.IsZero()`, else just `body`.
- `StripFrontmatter` keeps its existing contract: if no valid frontmatter block, returns `data`; else returns `data[bodyStart:]`.

---

## Task 1: New frontmatter.go API with failing tests first

**Files:**
- Modify: `note/frontmatter.go`
- Modify: `note/frontmatter_test.go`

- [ ] **Step 1: Write failing tests for `Frontmatter.IsZero`**

Append to `note/frontmatter_test.go`:

```go
func TestFrontmatterIsZero(t *testing.T) {
    tests := []struct {
        name string
        f    Frontmatter
        want bool
    }{
        {"empty", Frontmatter{}, true},
        {"title set", Frontmatter{Title: "T"}, false},
        {"slug set", Frontmatter{Slug: "s"}, false},
        {"tags empty slice counts as zero", Frontmatter{Tags: []string{}}, false},
        {"tags with value", Frontmatter{Tags: []string{"a"}}, false},
        {"description set", Frontmatter{Description: "d"}, false},
        {"public true", Frontmatter{Public: true}, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.f.IsZero(); got != tt.want {
                t.Errorf("IsZero() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Note: `reflect.ValueOf(Frontmatter{Tags: []string{}}).IsZero()` is false because a non-nil empty slice is not the zero value. That is acceptable — call sites build slices as nil, not empty. The test pins this.

- [ ] **Step 2: Write failing tests for `ParseNote` error return**

Add to `note/frontmatter_test.go`:

```go
func TestParseNoteErrors(t *testing.T) {
    cases := []struct {
        name  string
        input string
    }{
        {"unclosed flow sequence", "---\ntags: [a, b\n---\n\nbody\n"},
        {"control character", "---\ntitle: \"A\x00B\"\n---\n\nbody\n"},
        {"non-mapping top level", "---\n[1, 2, 3]\n---\n\nbody\n"},
        {"type mismatch on public", "---\npublic: maybe\n---\n\nbody\n"},
        {"type mismatch on tags", "---\ntags: not a list\n---\n\nbody\n"},
    }
    for _, tt := range cases {
        t.Run(tt.name, func(t *testing.T) {
            f, body, err := ParseNote([]byte(tt.input))
            if err == nil {
                t.Fatalf("expected error, got f=%+v body=%q", f, string(body))
            }
            if !f.IsZero() {
                t.Errorf("expected zero Frontmatter, got %+v", f)
            }
            if body != nil {
                t.Errorf("expected nil body, got %q", string(body))
            }
        })
    }
}

func TestParseNoteNoFrontmatter(t *testing.T) {
    input := []byte("# Heading\n\nbody text\n")
    f, body, err := ParseNote(input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !f.IsZero() {
        t.Errorf("expected zero Frontmatter, got %+v", f)
    }
    if string(body) != string(input) {
        t.Errorf("body = %q, want full input", string(body))
    }
}

func TestParseNoteHappyPath(t *testing.T) {
    input := []byte("---\ntitle: T\ntags: [a, b]\n---\n\n# Body\n")
    f, body, err := ParseNote(input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if f.Title != "T" {
        t.Errorf("Title = %q", f.Title)
    }
    if len(f.Tags) != 2 || f.Tags[0] != "a" || f.Tags[1] != "b" {
        t.Errorf("Tags = %v", f.Tags)
    }
    if string(body) != "# Body\n" {
        t.Errorf("body = %q", string(body))
    }
}

func TestParseNoteBodyIsSliceOfInput(t *testing.T) {
    input := []byte("---\ntitle: T\n---\n\nhello\n")
    _, body, err := ParseNote(input)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // body must point into input, not be a separate allocation.
    if len(body) == 0 {
        t.Fatal("body is empty")
    }
    if &body[0] != &input[len(input)-len(body)] {
        t.Error("body is not a sub-slice of input (extra allocation)")
    }
}
```

- [ ] **Step 3: Write failing tests for `FormatNote`**

```go
func TestFormatNoteEmptyFrontmatter(t *testing.T) {
    out := FormatNote(Frontmatter{}, []byte("body\n"))
    if string(out) != "body\n" {
        t.Errorf("got %q, want %q", string(out), "body\n")
    }
}

func TestFormatNoteWithFrontmatter(t *testing.T) {
    out := FormatNote(Frontmatter{Title: "T"}, []byte("body\n"))
    want := "---\ntitle: T\n---\n\nbody\n"
    if string(out) != want {
        t.Errorf("got %q, want %q", string(out), want)
    }
}

func TestFormatNoteRoundtrip(t *testing.T) {
    cases := []Frontmatter{
        {},
        {Title: "T"},
        {Tags: []string{"a", "b"}},
        {Title: "Re: Project", Tags: []string{"go", "rust, elixir"}, Description: "D", Public: true},
        {Slug: "my-slug", Public: true},
    }
    for i, fm := range cases {
        t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
            out := FormatNote(fm, []byte("body\n"))
            gotF, gotBody, err := ParseNote(out)
            if err != nil {
                t.Fatalf("parse failed: %v", err)
            }
            if !reflect.DeepEqual(gotF, fm) {
                t.Errorf("frontmatter: got %+v, want %+v", gotF, fm)
            }
            if string(gotBody) != "body\n" {
                t.Errorf("body: got %q, want %q", string(gotBody), "body\n")
            }
        })
    }
}
```

Add imports for `fmt` and `reflect` to the test file.

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./note/... -run 'TestFrontmatterIsZero|TestParseNote|TestFormatNote'`
Expected: compilation errors (types don't exist yet), which is a valid failure.

- [ ] **Step 5: Rewrite `note/frontmatter.go`**

Replace the entire file contents with:

```go
package note

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

const frontmatterDelim = "---"

// Frontmatter holds optional fields for note frontmatter.
// Adding a field is a one-line struct addition — no other changes required.
type Frontmatter struct {
	Title       string   `yaml:"title,omitempty"`
	Slug        string   `yaml:"slug,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Public      bool     `yaml:"public,omitempty"`
}

// IsZero reports whether f has no fields set.
func (f Frontmatter) IsZero() bool {
	return reflect.ValueOf(f).IsZero()
}

// ParseNote splits a note file into its frontmatter and body.
// If no frontmatter block is present, the zero Frontmatter is returned along
// with the full input as body and a nil error.
// If the frontmatter block is present but malformed, a non-nil error is
// returned along with the zero Frontmatter and a nil body.
// The returned body is a sub-slice of the input — no allocation.
func ParseNote(data []byte) (Frontmatter, []byte, error) {
	bodyStart, fmEnd, ok := frontmatterEnd(data)
	if !ok {
		return Frontmatter{}, data, nil
	}
	var f Frontmatter
	if err := yaml.Unmarshal(data[len(frontmatterDelim)+1:fmEnd], &f); err != nil {
		return Frontmatter{}, nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return f, data[bodyStart:], nil
}

// FormatNote serialises frontmatter followed by body. Omits the frontmatter
// block entirely when f.IsZero(). yaml.Marshal cannot fail for this struct,
// so marshal errors are treated as impossible and cause a panic.
func FormatNote(f Frontmatter, body []byte) []byte {
	if f.IsZero() {
		return body
	}
	out, err := yaml.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("yaml.Marshal Frontmatter: %v", err))
	}
	buf := make([]byte, 0, len(out)+len(body)+8)
	buf = append(buf, "---\n"...)
	buf = append(buf, out...)
	buf = append(buf, "---\n\n"...)
	buf = append(buf, body...)
	return buf
}

// StripFrontmatter returns data with any leading frontmatter block removed.
// If no valid frontmatter block is present, data is returned unchanged.
// This is a convenience for callers that want the body without parsing
// (e.g. `notesctl read --no-frontmatter`).
func StripFrontmatter(data []byte) []byte {
	bodyStart, _, ok := frontmatterEnd(data)
	if !ok {
		return data
	}
	return data[bodyStart:]
}

// frontmatterEnd locates the YAML frontmatter block at the start of data.
// Returns bodyStart (index after the closing delimiter line and optional
// blank line), fmEnd (index of the newline terminating the closing "---"
// line), and ok=true if a valid block was found. ok=false means data does
// not begin with a well-formed frontmatter block.
//
// Interior CRLF sequences inside the YAML body are preserved in the slice
// yaml.Unmarshal sees; only the opening and closing delimiter lines are
// trimmed of trailing \r for comparison.
func frontmatterEnd(data []byte) (bodyStart, fmEnd int, ok bool) {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(data, delim) {
		return 0, 0, false
	}
	rest := data[len(delim):]
	firstNL := bytes.IndexByte(rest, '\n')
	if firstNL < 0 {
		return 0, 0, false
	}
	// Characters on the opening delimiter line after "---" (other than a
	// trailing \r) disqualify the block.
	if len(bytes.TrimRight(rest[:firstNL], "\r")) > 0 {
		return 0, 0, false
	}
	// Scan subsequent lines for the closing "---" delimiter.
	offset := len(delim) + firstNL + 1
	for offset < len(data) {
		nl := bytes.IndexByte(data[offset:], '\n')
		var line []byte
		var lineEnd int
		if nl < 0 {
			line = data[offset:]
			lineEnd = len(data)
		} else {
			line = data[offset : offset+nl]
			lineEnd = offset + nl
		}
		if bytes.Equal(bytes.TrimRight(line, "\r"), delim) {
			fmEnd = lineEnd
			bodyStart = lineEnd
			if bodyStart < len(data) && data[bodyStart] == '\n' {
				bodyStart++
			} else if bodyStart+1 < len(data) && data[bodyStart] == '\r' && data[bodyStart+1] == '\n' {
				bodyStart += 2
			}
			return bodyStart, fmEnd, true
		}
		if nl < 0 {
			return 0, 0, false
		}
		offset += nl + 1
	}
	return 0, 0, false
}
```

- [ ] **Step 6: Run new tests to verify they pass**

Run: `go test ./note/... -run 'TestFrontmatterIsZero|TestParseNote|TestFormatNote'`
Expected: PASS.

- [ ] **Step 7: Run the full `note` package test suite — expect failures in old tests**

Run: `go test ./note/...`
Expected: compilation errors in old test functions that reference `FrontmatterFields`, `ParseFrontmatterFields`, `BuildFrontmatter`. This is expected — we will migrate them in Task 2.

- [ ] **Step 8: Commit the new API and its tests**

Stop. Don't commit yet — tests in the file still reference the old API, and the package won't compile. Proceed straight to Task 2, which migrates the tests. Commit at the end of Task 2.

---

## Task 2: Migrate `frontmatter_test.go` to the new API

**Files:**
- Modify: `note/frontmatter_test.go`

- [ ] **Step 1: Rename helpers and consolidate the two `TestParseFrontmatterFields*` tables**

Replace the whole file with a consolidated version that:

1. Has a single happy-path + adversarial table combined, checked against `ParseNote`.
2. Splits error cases (into `TestParseNoteErrors`, already written in Task 1) from success cases.
3. Keeps `TestBuildFrontmatter` as `TestFormatNoteSnapshot` with **one** canonical snapshot (all fields set); the other snapshot cases become round-trip assertions.
4. Keeps `TestStripFrontmatter` as-is but uses `FormatNote` where it used `BuildFrontmatter`.

Final test file:

```go
package note

import (
	"fmt"
	"reflect"
	"testing"
)

// --- Frontmatter.IsZero ---

func TestFrontmatterIsZero(t *testing.T) {
	tests := []struct {
		name string
		f    Frontmatter
		want bool
	}{
		{"empty", Frontmatter{}, true},
		{"title set", Frontmatter{Title: "T"}, false},
		{"slug set", Frontmatter{Slug: "s"}, false},
		{"tags empty slice not zero", Frontmatter{Tags: []string{}}, false},
		{"tags with value", Frontmatter{Tags: []string{"a"}}, false},
		{"description set", Frontmatter{Description: "d"}, false},
		{"public true", Frontmatter{Public: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- ParseNote: success cases ---

func TestParseNoteSuccess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Frontmatter
		body  string
	}{
		{"empty input", "", Frontmatter{}, ""},
		{"no frontmatter", "# Hello\n\nBody text.\n", Frontmatter{}, "# Hello\n\nBody text.\n"},
		{"title only", "---\ntitle: My Note\n---\n\n# Content\n", Frontmatter{Title: "My Note"}, "# Content\n"},
		{"slug only", "---\nslug: my-slug\n---\n\n# Content\n", Frontmatter{Slug: "my-slug"}, "# Content\n"},
		{"tags only", "---\ntags: [work, planning]\n---\n\n# Content\n", Frontmatter{Tags: []string{"work", "planning"}}, "# Content\n"},
		{"description only", "---\ndescription: Quick thought\n---\n\n# Content\n", Frontmatter{Description: "Quick thought"}, "# Content\n"},
		{"public true", "---\npublic: true\n---\n\n# Content\n", Frontmatter{Public: true}, "# Content\n"},
		{"public absent false", "---\ntitle: T\n---\n\n# Content\n", Frontmatter{Title: "T"}, "# Content\n"},
		{"all fields", "---\ntitle: T\nslug: s\ntags: [a]\ndescription: D\npublic: true\n---\n\n# Content\n",
			Frontmatter{Title: "T", Slug: "s", Tags: []string{"a"}, Description: "D", Public: true}, "# Content\n"},
		{"unclosed frontmatter treated as no frontmatter", "---\ntitle: Oops\n# Content\n", Frontmatter{}, "---\ntitle: Oops\n# Content\n"},
		{"duplicate keys: last wins", "---\ntitle: A\ntitle: B\n---\n", Frontmatter{Title: "B"}, ""},
		{"merge key applied with later override", "---\n<<: {title: X}\ntitle: Y\n---\n", Frontmatter{Title: "Y"}, ""},
		{"int coerced to string", "---\ntitle: 12345\n---\n", Frontmatter{Title: "12345"}, ""},
		{"null leaves field empty", "---\ntitle: null\nslug: s\n---\n", Frontmatter{Slug: "s"}, ""},
		{"unknown keys ignored", "---\ntitle: T\nrandom: whatever\nnested: {a: 1}\n---\n", Frontmatter{Title: "T"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, body, err := ParseNote([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(f, tt.want) {
				t.Errorf("frontmatter: got %+v, want %+v", f, tt.want)
			}
			if string(body) != tt.body {
				t.Errorf("body: got %q, want %q", string(body), tt.body)
			}
		})
	}
}

// --- ParseNote: error cases ---

func TestParseNoteErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unclosed flow sequence", "---\ntitle: T\ntags: [a, b\n---\n\n# Content\n"},
		{"invalid bool value", "---\npublic: maybe\n---\n\n# Content\n"},
		{"bad field alongside good", "---\ntitle: T\npublic: maybe\ntags: [a, b]\n---\n\n# Content\n"},
		{"control character", "---\ntitle: \"A\x00B\"\nslug: s\n---\n"},
		{"non-mapping top level", "---\n[1, 2, 3]\n---\n"},
		{"alias bomb", "---\n" +
			"a: &a [x]\n" +
			"b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]\n" +
			"c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]\n" +
			"tags: *c\n" +
			"title: T\n" +
			"---\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f, body, err := ParseNote([]byte(tt.input))
			if err == nil {
				t.Fatalf("expected error, got f=%+v body=%q", f, string(body))
			}
			if !f.IsZero() {
				t.Errorf("expected zero Frontmatter on error, got %+v", f)
			}
			if body != nil {
				t.Errorf("expected nil body on error, got %q", string(body))
			}
		})
	}
}

// --- ParseNote: body is a sub-slice (zero-allocation body) ---

func TestParseNoteBodyIsSliceOfInput(t *testing.T) {
	input := []byte("---\ntitle: T\n---\n\nhello\n")
	_, body, err := ParseNote(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	if &body[0] != &input[len(input)-len(body)] {
		t.Error("body is not a sub-slice of input (extra allocation)")
	}
}

// --- FormatNote: canonical snapshot (one, to pin output style) ---

func TestFormatNoteSnapshotAllFields(t *testing.T) {
	f := Frontmatter{
		Title:       "T",
		Slug:        "s",
		Tags:        []string{"a"},
		Description: "D",
		Public:      true,
	}
	want := "---\ntitle: T\nslug: s\ntags:\n    - a\ndescription: D\npublic: true\n---\n\nbody\n"
	got := string(FormatNote(f, []byte("body\n")))
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatNoteEmptyFrontmatter(t *testing.T) {
	if got := string(FormatNote(Frontmatter{}, []byte("body\n"))); got != "body\n" {
		t.Errorf("got %q, want %q", got, "body\n")
	}
}

// --- ParseNote / FormatNote round-trip ---

func TestRoundtrip(t *testing.T) {
	cases := []Frontmatter{
		{},
		{Title: "T"},
		{Tags: []string{"a", "b"}},
		{Tags: []string{"go", "rust, elixir"}},
		{Tags: []string{"foo: bar", "baz]"}},
		{Title: "Re: Project update"},
		{Title: "T", Slug: "s", Tags: []string{"a"}, Public: true},
		{Title: "T", Slug: "s", Tags: []string{"a"}, Description: "D", Public: true},
	}
	for i, fm := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			out := FormatNote(fm, []byte("body\n"))
			gotF, gotBody, err := ParseNote(out)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if !reflect.DeepEqual(gotF, fm) {
				t.Errorf("frontmatter: got %+v, want %+v", gotF, fm)
			}
			if string(gotBody) != "body\n" {
				t.Errorf("body: got %q, want %q", string(gotBody), "body\n")
			}
		})
	}
}

// --- StripFrontmatter ---

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no frontmatter", "# Hello\n\nBody text.\n", "# Hello\n\nBody text.\n"},
		{"with frontmatter", "---\nslug: todo\ntags: [journal]\n---\n\n# Hello\n\nBody text.\n", "# Hello\n\nBody text.\n"},
		{"frontmatter only", "---\nslug: todo\n---\n", ""},
		{"empty input", "", ""},
		{"unclosed frontmatter", "---\nslug: todo\n# Hello\n", "---\nslug: todo\n# Hello\n"},
		{"triple dash in body not at start", "# Hello\n\n---\n\nFooter.\n", "# Hello\n\n---\n\nFooter.\n"},
		{"preserves multiple blank lines after frontmatter", "---\nslug: todo\n---\n\n\n\nContent\n", "\n\nContent\n"},
		{"opening delimiter with trailing text", "---extra\nslug: x\n---\n\nBody\n", "---extra\nslug: x\n---\n\nBody\n"},
		{"opening delimiter only no newline", "---", "---"},
		{"opening delimiter only with newline", "---\nstuff\n", "---\nstuff\n"},
		{"empty frontmatter block", "---\n---\n\nBody\n", "Body\n"},
		{"malformed yaml still stripped", "---\n[bad: yaml\n---\n\nBody\n", "Body\n"},
		{"multiple closing delimiters", "---\na\n---\nb\n---\n\nBody\n", "b\n---\n\nBody\n"},
		{"roundtrip with FormatNote", string(FormatNote(Frontmatter{Tags: []string{"journal"}, Description: "A note"}, []byte("# Content\n"))), "# Content\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StripFrontmatter([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- CRLF handling: documented as "LF-only on write; LF-normalised on parse of delimiter lines;
//     CRLF interior bytes preserved through Unmarshal" ---

func TestParseNoteCRLFInteriorPreserved(t *testing.T) {
	// Note with CRLF line endings throughout: delimiters, fields, body.
	input := []byte("---\r\ntitle: T\r\ntags:\r\n  - a\r\n  - b\r\n---\r\n\r\nbody line\r\nsecond\r\n")
	f, body, err := ParseNote(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Title != "T" {
		t.Errorf("Title = %q", f.Title)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "a" || f.Tags[1] != "b" {
		t.Errorf("Tags = %v", f.Tags)
	}
	// Body must preserve CRLF bytes that appeared inside it.
	want := "body line\r\nsecond\r\n"
	if string(body) != want {
		t.Errorf("body: got %q, want %q", string(body), want)
	}
}

func TestFormatNoteWritesLFOnly(t *testing.T) {
	// FormatNote always emits LF, regardless of the body's line endings.
	out := FormatNote(Frontmatter{Title: "T"}, []byte("hello\r\nworld\r\n"))
	// Delimiter lines are LF.
	if string(out[:18]) != "---\ntitle: T\n---\n\n" {
		t.Errorf("delimiter lines not LF-only: %q", string(out[:18]))
	}
	// Body bytes pass through unchanged — preserves whatever the caller gave us.
	if string(out[18:]) != "hello\r\nworld\r\n" {
		t.Errorf("body modified: %q", string(out[18:]))
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./note/...`
Expected: all `note` package tests pass. Callers in `internal/cli/...` will still fail to compile — we address those in later tasks.

- [ ] **Step 3: Commit**

Don't commit yet. `internal/cli` still fails to compile. Proceed to Task 3.

---

## Task 3: Migrate `internal/cli/create.go` and `createNote`

**Files:**
- Modify: `internal/cli/create.go`

- [ ] **Step 1: Rewrite `createNote` to use `FormatNote`**

Replace the body of `createNote` so that the content is built via `FormatNote`. New code:

```go
func createNote(p createNoteParams) (string, error) {
	today := time.Now().Format("20060102")

	id, err := note.NextID(p.Root)
	if err != nil {
		return "", err
	}

	filename := note.NoteFilename(today, id, p.Slug, p.Type)
	dir := note.NoteDirPath(p.Root, today)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	fullPath := filepath.Join(dir, filename)

	fm := note.Frontmatter{
		Title:       p.Title,
		Slug:        p.Slug,
		Tags:        p.Tags,
		Description: p.Description,
		Public:      p.Public,
	}
	content := note.FormatNote(fm, []byte(p.Body))

	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return "", fmt.Errorf("cannot write note: %w", err)
	}

	return fullPath, nil
}
```

- [ ] **Step 2: Run tests (still won't fully compile)**

Run: `go test ./internal/cli/... -run TestNew`
Expected: compilation errors in `update.go` / `annotate.go` — those are migrated next.

---

## Task 4: Migrate `internal/cli/update.go`

**Files:**
- Modify: `internal/cli/update.go:62-117`

- [ ] **Step 1: Collapse the parse/strip/build dance**

Replace the block from the `oldPath := ...` line through `newContent := ...` with:

```go
		oldPath := filepath.Join(root, n.RelPath)
		data, err := os.ReadFile(oldPath)
		if err != nil {
			return fmt.Errorf("cannot read note: %w", err)
		}

		updated, body, err := note.ParseNote(data)
		if err != nil {
			return fmt.Errorf("%s: %w", oldPath, err)
		}

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

		// Determine new slug.
		newSlug := n.Slug
		if updateNoSlug {
			newSlug = ""
		} else if cmd.Flags().Changed("slug") {
			newSlug = updateSlug
		}
		if updateNoSlug || cmd.Flags().Changed("slug") {
			updated.Slug = newSlug
		}
		if updatePrivate {
			updated.Public = false
		} else if cmd.Flags().Changed("public") {
			updated.Public = true
		}

		// Determine new type.
		newType := n.Type
		if updateNoType {
			newType = ""
		} else if cmd.Flags().Changed("type") {
			newType = updateType
		}

		// n.ID is guaranteed to be a non-empty digit string by ParseFilename.
		id, _ := strconv.Atoi(n.ID)

		newFilename := note.NoteFilename(n.Date, id, newSlug, newType)
		dir := filepath.Dir(oldPath)
		newPath := filepath.Join(dir, newFilename)

		newContent := note.FormatNote(updated, body)
```

Then change the tmpfile write:

```go
		if err := os.WriteFile(tmpPath, newContent, 0o644); err != nil {
```

(remove the `[]byte(...)` cast since `newContent` is already `[]byte`).

- [ ] **Step 2: Run update tests**

Run: `go test ./internal/cli/... -run TestUpdate`
Expected: all update tests pass.

---

## Task 5: Migrate `internal/cli/annotate.go` and `annotate_test.go`

**Files:**
- Modify: `internal/cli/annotate.go`
- Modify: `internal/cli/annotate_test.go`

- [ ] **Step 1: Replace parse/strip dance with ParseNote**

In `annotate.go`, replace lines ~63-101 (read-through-writeback) with:

```go
	fullPath := filepath.Join(root, n.RelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("cannot read note: %w", err)
	}

	existing, body, err := note.ParseNote(data)
	if err != nil {
		return fmt.Errorf("%s: %w", fullPath, err)
	}

	empty := annotateEmptyFields(existing)
	if len(empty) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return errors.New("note has no body content to annotate")
	}

	prompt := string(body)
	if maxChars > 0 {
		if runes := []rune(prompt); len(runes) > maxChars {
			prompt = string(runes[:maxChars])
			fmt.Fprintf(cmd.ErrOrStderr(), "truncated note body to %d chars for annotation\n", maxChars)
		}
	}

	schema := buildAnnotateSchema(empty)
	out, err := runClaude(model, schema, prompt)
	if err != nil {
		return err
	}

	gen, err := parseAnnotation(out)
	if err != nil {
		return err
	}

	merged := mergeAnnotation(existing, gen)
	newContent := note.FormatNote(merged, body)

	tmpPath := fullPath + ".tmp"
	if err := os.WriteFile(tmpPath, newContent, 0o644); err != nil {
		return fmt.Errorf("cannot write note: %w", err)
	}
```

Also update the function signatures:

```go
func annotateEmptyFields(f note.Frontmatter) []string {
    // unchanged body
}

func mergeAnnotation(existing note.Frontmatter, gen annotateResult) note.Frontmatter {
    // unchanged body
}
```

- [ ] **Step 2: Update annotate_test.go to use `note.Frontmatter`**

In `internal/cli/annotate_test.go`, globally replace `note.FrontmatterFields` with `note.Frontmatter`. Six references at lines ~32, 40, 49, 184, 210.

- [ ] **Step 3: Run annotate tests**

Run: `go test ./internal/cli/... -run TestAnnotate`
Expected: all pass.

---

## Task 6: Migrate `note/store.go` with warn-on-error

**Files:**
- Modify: `note/store.go`

- [ ] **Step 1: Update `FilterByTags` to use ParseNote with warn-and-continue**

Replace the `FilterByTags` function with:

```go
// FilterByTags returns notesctl that contain all of the given tags in their frontmatter.
// Per-note frontmatter parse errors are logged via log.Printf and the note is skipped.
func FilterByTags(notes []Note, root string, tags []string) ([]Note, error) {
	var results []Note
	for _, n := range notesctl {
		path := filepath.Join(root, n.RelPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fm, _, parseErr := ParseNote(data)
		if parseErr != nil {
			log.Printf("warn: %s: %v", path, parseErr)
			continue
		}
		if hasAllTags(fm.Tags, tags) {
			results = append(results, n)
		}
	}
	return results, nil
}
```

Add `"log"` to the imports.

- [ ] **Step 2: Run tests**

Run: `go test ./note/...`
Expected: all pass.

---

## Task 7: Rewrite `ExtractTags` using errgroup; switch to ParseNote

**Files:**
- Modify: `note/tags.go`
- Modify: `go.mod`

- [ ] **Step 1: Promote `golang.org/x/sync` to a direct require**

Run: `go get golang.org/x/sync@v0.12.0`
This moves the existing indirect dep to direct.

- [ ] **Step 2: Rewrite `ExtractTags`**

Replace the `ExtractTags` function body (lines 17-91) with an errgroup-based version:

```go
// ExtractTags scans the note store under root and returns a sorted,
// deduplicated list of tags. Sources: frontmatter `tags:` fields and body
// hashtags (#word) in the prose. File reads run concurrently across
// runtime.NumCPU() workers. Returns a nil slice for an empty store.
// A per-note frontmatter parse error is logged via log.Printf and the
// note's frontmatter tags are skipped (body hashtags are still collected).
// Any file-read error aborts the scan.
func ExtractTags(root string) ([]string, error) {
	notes, err := Scan(root)
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, nil
	}

	workers := runtime.NumCPU()
	if workers > len(notes) {
		workers = len(notes)
	}

	g, ctx := errgroup.WithContext(context.Background())
	jobs := make(chan Note)
	var mu sync.Mutex
	merged := make(map[string]struct{})

	g.Go(func() error {
		defer close(jobs)
		for _, n := range notesctl {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case jobs <- n:
			}
		}
		return nil
	})

	for i := 0; i < workers; i++ {
		g.Go(func() error {
			local := make(map[string]struct{})
			for n := range jobs {
				path := filepath.Join(root, n.RelPath)
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				fm, body, parseErr := ParseNote(data)
				if parseErr != nil {
					log.Printf("warn: %s: %v", path, parseErr)
					body = StripFrontmatter(data)
				} else {
					for _, t := range fm.Tags {
						if t != "" {
							local[t] = struct{}{}
						}
					}
				}
				for _, t := range extractHashtags(body) {
					local[t] = struct{}{}
				}
			}
			mu.Lock()
			for t := range local {
				merged[t] = struct{}{}
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(merged))
	for t := range merged {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}
```

Update imports: add `context`, `log`, `golang.org/x/sync/errgroup`. Keep `runtime`, `sort`, `sync`, `os`, `path/filepath`, `bytes` (for extractHashtags).

- [ ] **Step 3: Run tests**

Run: `go test ./note/...`
Expected: all pass.

---

## Task 8: Verify full build and lint

**Files:** n/a

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 2: Run lint**

Run: `make lint`
Expected: no errors.

- [ ] **Step 3: Verify no remaining references to old API**

Run via Grep tool: `FrontmatterFields|ParseFrontmatterFields|BuildFrontmatter` across `**/*.go`.
Expected: zero matches.

---

## Task 9: Changelog and commit

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add v0.1.72 entry at the top**

Insert above the `## [0.1.71]` heading:

```markdown
## [0.1.72] - 2026-04-19

### Changed

- Replace `ParseFrontmatterFields` / `BuildFrontmatter` trio with error-returning `ParseNote` / `FormatNote` pair; rename `FrontmatterFields` → `Frontmatter` with `IsZero`. Single-note writers (`update`, `annotate`) now surface frontmatter parse errors; bulk readers (`FilterByTags`, `ExtractTags`) log per-note warnings and continue. Body is returned as a sub-slice of the input (no copy), and CRLF interior bytes round-trip through parsing. `ExtractTags` concurrency now uses `errgroup`. ([#112])
```

Then add the reference at the bottom of the file:

```markdown
[#112]: https://github.com/dreikanter/notesctl/pull/112
```

- [ ] **Step 2: Commit all changes**

```bash
git add -A
git commit -m "Refactor frontmatter API: ParseNote/FormatNote with real errors (#112)"
```

- [ ] **Step 3: Push branch and open PR**

```bash
git push -u origin <branch>
gh pr create --title "Refactor frontmatter API: ParseNote/FormatNote with real errors" --body "$(cat <<'EOF'
## Summary

- Replace `ParseFrontmatterFields` / `BuildFrontmatter` trio with `ParseNote` / `FormatNote` pair that returns real errors
- Rename `FrontmatterFields` → `Frontmatter`; add `IsZero` method; body is a zero-copy sub-slice of input
- Surface parse errors in single-note writers (`update`, `annotate`); log warn-and-continue in bulk readers (`FilterByTags`, `ExtractTags`)
- Rewrite `ExtractTags` concurrency using `errgroup`

## References

- closes #112
EOF
)"
```
