# Tag Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `notes tag` command that prints the sorted, deduplicated union of frontmatter `tags:` and body hashtags from every note in the store.

**Architecture:** Two-layer split. `note/tag.go` owns extraction: `extractHashtags(body []byte) []string` is a hand-rolled byte scanner that handles heading/code-block/inline-code/word-boundary rules; `ExtractTags(root string) ([]string, error)` fans note reads out over a `runtime.NumCPU()` worker pool, merges per-worker sets, sorts, and returns. `internal/cli/tag.go` is a thin cobra wrapper that calls `ExtractTags` and prints one tag per line.

**Tech Stack:** Go 1.x, cobra (existing CLI), standard library only (no regex; byte-level scan for perf).

---

## File Structure

| Path | Purpose |
|------|---------|
| `note/tag.go` | `ExtractTags` (parallel scan + merge) and `extractHashtags` (byte scanner). |
| `note/tag_test.go` | Unit tests for `extractHashtags` and `ExtractTags`. |
| `internal/cli/tag.go` | Cobra command: no args, no flags, prints sorted tag list. |
| `internal/cli/tag_test.go` | Integration tests that drive the cobra command against a temp store. |
| `CHANGELOG.md` | One entry under the next patch version. |
| `README.md` | One usage example under the Usage block. |

---

## Task 1: Byte scanner for body hashtags

**Files:**
- Create: `note/tag.go`
- Create: `note/tag_test.go`

- [ ] **Step 1: Write failing tests for `extractHashtags`**

Create `note/tag_test.go`:

```go
package note

import (
	"reflect"
	"testing"
)

func TestExtractHashtagsBasic(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"simple", "hello #world", []string{"world"}},
		{"multiple", "#alpha and #beta here", []string{"alpha", "beta"}},
		{"digits and dashes", "#a-b_c #123 #x1", []string{"a-b_c", "123", "x1"}},
		{"slash terminates", "see #foo/bar", []string{"foo"}},
		{"punctuation after", "ok #tag, next.", []string{"tag"}},
		{"parens", "(#tag)", []string{"tag"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractHashtags([]byte(c.in))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestExtractHashtagsNegative(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"heading h1", "# heading not a tag"},
		{"heading h2", "## another heading"},
		{"indented heading", "   # still a heading"},
		{"word-prefixed", "foo#bar baz"},
		{"bare hash", "look here: # not-tag"},
		{"lone hash", "just # alone"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractHashtags([]byte(c.in))
			if len(got) != 0 {
				t.Fatalf("expected no tags, got %v", got)
			}
		})
	}
}

func TestExtractHashtagsInlineCode(t *testing.T) {
	in := "real #out and `inline #in` and #back"
	want := []string{"out", "back"}
	got := extractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsFencedBlock(t *testing.T) {
	in := "before #a\n```\n#hidden\n#also-hidden\n```\nafter #b\n"
	want := []string{"a", "b"}
	got := extractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractHashtagsFencedBlockWithInfoString(t *testing.T) {
	in := "top #ok\n```go\n// #comment\n```\nend #done\n"
	want := []string{"ok", "done"}
	got := extractHashtags([]byte(in))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./note/ -run TestExtractHashtags -v`
Expected: compile error — `undefined: extractHashtags`.

- [ ] **Step 3: Implement `extractHashtags` in `note/tag.go`**

Create `note/tag.go`:

```go
package note

import (
	"bytes"
)

// extractHashtags scans body text and returns hashtag tokens (without the
// leading '#'), preserving source order and including duplicates. Rules:
//   - Lines whose first non-whitespace character is '#' are skipped (headings).
//   - Fenced code blocks (``` on a line, optionally indented, with optional
//     info string) are skipped until the next fence line.
//   - Inline backtick spans on a single line are skipped.
//   - A '#' preceded by a word character ([A-Za-z0-9_]) is not a tag.
//   - Tag characters are [A-Za-z0-9_-]; other characters terminate a tag.
func extractHashtags(body []byte) []string {
	var out []string
	inFence := false

	for len(body) > 0 {
		nl := bytes.IndexByte(body, '\n')
		var line []byte
		if nl < 0 {
			line = body
			body = nil
		} else {
			line = body[:nl]
			body = body[nl+1:]
		}

		trim := 0
		for trim < len(line) && (line[trim] == ' ' || line[trim] == '\t') {
			trim++
		}

		// Fence toggle: a line whose first non-ws content starts with ```
		if trim+3 <= len(line) && line[trim] == '`' && line[trim+1] == '`' && line[trim+2] == '`' {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// Heading: first non-ws is '#'
		if trim < len(line) && line[trim] == '#' {
			continue
		}

		inInline := false
		for j := 0; j < len(line); j++ {
			c := line[j]
			if c == '`' {
				inInline = !inInline
				continue
			}
			if c != '#' || inInline {
				continue
			}
			if j > 0 && isWordByte(line[j-1]) {
				continue
			}
			k := j + 1
			for k < len(line) && isTagByte(line[k]) {
				k++
			}
			if k > j+1 {
				out = append(out, string(line[j+1:k]))
			}
			j = k - 1
		}
	}
	return out
}

func isTagByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./note/ -run TestExtractHashtags -v`
Expected: all `TestExtractHashtags*` subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add note/tag.go note/tag_test.go
git commit -m "Add hashtag byte scanner"
```

---

## Task 2: `ExtractTags` — parallel store scan

**Files:**
- Modify: `note/tag.go` (append `ExtractTags`)
- Modify: `note/tag_test.go` (append new tests)

- [ ] **Step 1: Write failing tests for `ExtractTags`**

Append to `note/tag_test.go`:

```go
import (
	"os"
	"path/filepath"
	// keep reflect, testing already imported above
)

func writeNote(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtractTagsEmpty(t *testing.T) {
	root := t.TempDir()
	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no tags, got %v", got)
	}
}

func TestExtractTagsFrontmatterOnly(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, planning]\n---\n\nbody here.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"---\ntags: [work]\n---\n\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"planning", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsBodyHashtagsOnly(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"Text with #alpha and #beta.\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"More text #alpha only.\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsMergedAndDeduped(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, shared]\n---\n\nBody #shared #body-only\n")
	writeNote(t, root, "2026/01/20260102_1002.md",
		"no frontmatter here #work #another\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"another", "body-only", "shared", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "2026/01/20260101_1001.md",
		"real #kept\n```\n#ignored\n```\nafter #also-kept\n")

	got, err := ExtractTags(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"also-kept", "kept"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestExtractTagsNonexistentRoot(t *testing.T) {
	_, err := ExtractTags(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./note/ -run TestExtractTags -v`
Expected: compile error — `undefined: ExtractTags`.

- [ ] **Step 3: Implement `ExtractTags` in `note/tag.go`**

Append to `note/tag.go`:

```go
import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// ExtractTags scans the note store under root and returns a sorted,
// deduplicated list of tags. Sources: frontmatter `tags:` fields and body
// hashtags (#word) in the prose. File reads run concurrently across
// runtime.NumCPU() workers.
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

	jobs := make(chan Note)
	results := make(chan map[string]struct{}, workers)
	errCh := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make(map[string]struct{})
			for n := range jobs {
				data, err := os.ReadFile(filepath.Join(root, n.RelPath))
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				for _, t := range ParseFrontmatterFields(data).Tags {
					if t != "" {
						local[t] = struct{}{}
					}
				}
				for _, t := range extractHashtags(StripFrontmatter(data)) {
					local[t] = struct{}{}
				}
			}
			results <- local
		}()
	}

	for _, n := range notes {
		jobs <- n
	}
	close(jobs)
	wg.Wait()
	close(results)
	close(errCh)

	if err := <-errCh; err != nil {
		return nil, err
	}

	merged := make(map[string]struct{})
	for local := range results {
		for t := range local {
			merged[t] = struct{}{}
		}
	}

	out := make([]string, 0, len(merged))
	for t := range merged {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}
```

Note: the `import` block shown above is the full set needed by `tag.go` after this task; merge it with the `import "bytes"` from Task 1 into one block.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./note/ -run TestExtractTags -v`
Expected: all `TestExtractTags*` subtests PASS.

- [ ] **Step 5: Run the full `note/` test suite**

Run: `go test ./note/ -v`
Expected: all existing + new tests PASS.

- [ ] **Step 6: Commit**

```bash
git add note/tag.go note/tag_test.go
git commit -m "Add ExtractTags for parallel store scan"
```

---

## Task 3: `tag` CLI command

**Files:**
- Create: `internal/cli/tag.go`
- Create: `internal/cli/tag_test.go`

- [ ] **Step 1: Write failing integration tests**

Create `internal/cli/tag_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runTag(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	tagCmd.ResetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"tag", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func writeTagTestNote(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTagEmptyStore(t *testing.T) {
	root := t.TempDir()
	out, err := runTag(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestTagMergedSourcesSorted(t *testing.T) {
	root := t.TempDir()
	writeTagTestNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, planning]\n---\n\nHere is #coffee and #work again.\n")
	writeTagTestNote(t, root, "2026/01/20260102_1002.md",
		"no fm, just #tea and #work.\n")

	out, err := runTag(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Split(out, "\n")
	want := []string{"coffee", "planning", "tea", "work"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTagIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	writeTagTestNote(t, root, "2026/01/20260101_1001.md",
		"kept #real\n```\n#should-not-appear\n```\nalso #done\n")

	out, err := runTag(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "should-not-appear") {
		t.Errorf("expected code-block hashtag to be excluded, got:\n%s", out)
	}
	if !strings.Contains(out, "real") || !strings.Contains(out, "done") {
		t.Errorf("expected real and done tags, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestTag -v`
Expected: compile error — `undefined: tagCmd`.

- [ ] **Step 3: Implement the command in `internal/cli/tag.go`**

Create `internal/cli/tag.go`:

```go
package cli

import (
	"fmt"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "List all tags from frontmatter and body hashtags",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		tags, err := note.ExtractTags(root)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		for _, t := range tags {
			fmt.Fprintln(out, t)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestTag -v`
Expected: all `TestTag*` subtests PASS.

- [ ] **Step 5: Run the full CLI test suite**

Run: `go test ./internal/cli/ -v`
Expected: all existing + new tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/tag.go internal/cli/tag_test.go
git commit -m "Add tag command"
```

---

## Task 4: Documentation

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `README.md`

- [ ] **Step 1: Determine the next version**

Run: `git describe --tags`
Use the output (e.g. `v0.1.66`) and increment the patch number (→ `0.1.67`). If the tip has moved, use whatever the actual next patch is.

- [ ] **Step 2: Add a CHANGELOG entry**

At the top of `CHANGELOG.md` (just under the `# Changelog` heading), insert a new section using the next version from Step 1. Replace `{{VERSION}}` with that version and `{{PR}}` with the PR number once known (leave it as-is if the PR has not been opened yet — update it before merge):

```markdown
## [{{VERSION}}] - 2026-04-18

### Added

- Add `tag` command that lists unique tags from frontmatter and body hashtags across the store ([#{{PR}}])
```

Also add the footer link alongside the other entries at the bottom:

```markdown
[#{{PR}}]: https://github.com/dreikanter/notes-cli/pull/{{PR}}
```

- [ ] **Step 3: Add a README usage example**

In `README.md`, find the `# Search note contents` block in the Usage section (`notes grep`, `notes rg`). Immediately after it, append:

```markdown

# List all tags (frontmatter + body hashtags)
notes tag
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md README.md
git commit -m "Document tag command"
```

---

## Task 5: Verification

- [ ] **Step 1: Run the linter**

Run: `make lint`
Expected: no errors.

- [ ] **Step 2: Run the full test suite**

Run: `make test`
Expected: all tests PASS.

- [ ] **Step 3: Smoke-test against the real store**

Run: `make build && ./notes tag --path ./testdata`
Expected: sorted list including at least `meeting`, `planning`, `work` (from testdata frontmatter).

- [ ] **Step 4: Confirm performance on a synthetic large store**

Run in a shell:

```bash
root=$(mktemp -d)
for y in 2024 2025 2026; do
  for m in 01 02 03 04 05 06 07 08 09 10 11 12; do
    mkdir -p "$root/$y/$m"
    for i in $(seq 1 100); do
      printf -- "---\ntags: [t%d, t%d]\n---\n\nBody with #h%d and #common.\n" \
        "$i" "$((i%20))" "$((i%50))" \
        > "$root/$y/$m/${y}${m}01_$((RANDOM+1000+i)).md"
    done
  done
done
time ./notes tag --path "$root" | wc -l
```

Expected: finishes in well under a second on a modern machine (3600 notes total), with a plausible tag count printed. This is a manual sanity check — no assertion beyond "feels fast."

---

## Self-Review

**Spec coverage:**
- Command shape (no args/flags, sorted unique list) → Task 3.
- Frontmatter tag source → Task 2 (uses `ParseFrontmatterFields`).
- Body hashtag source with heading / code-block / inline-code / word-boundary rules → Task 1 (`extractHashtags`) + tests.
- Parallel `runtime.NumCPU()` pipeline, no caching → Task 2 (`ExtractTags`).
- File layout (`note/tag.go`, `internal/cli/tag.go`) → Tasks 1–3.
- Unit + integration tests per spec table → Tasks 1–3.
- CHANGELOG entry under next patch version → Task 4.

**Placeholder scan:** `{{VERSION}}` and `{{PR}}` in Task 4 are the only templates — explicitly explained in-place. No "TBD" / "implement later" / unspecified error handling.

**Type consistency:** `ExtractTags(root string) ([]string, error)` is used identically in Tasks 2 and 3. `extractHashtags([]byte) []string`, `isTagByte(byte) bool`, `isWordByte(byte) bool` — signatures consistent between definition and tests. Cobra variable `tagCmd` used in both `internal/cli/tag.go` and `internal/cli/tag_test.go`.
