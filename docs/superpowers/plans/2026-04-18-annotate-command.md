# `notesctl annotate` Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `notesctl annotate <ref>` command that shells out to the Claude Code CLI to fill empty `title`, `description`, and `tags` fields in a note's frontmatter.

**Architecture:** One new file `internal/cli/annotate.go` holds the cobra command, Claude invocation, schema builder, response parser, and merge logic. Tests in `internal/cli/annotate_test.go` use a fake `claude` shell script (pattern from `edit_test.go`) swapped in via a package-level `claudeBinary` variable. The command is non-destructive: filled fields are never touched, file is left unchanged on any error.

**Tech Stack:** Go 1.x, Cobra, `os/exec` for subprocess, `encoding/json` for schema/response, Claude Code CLI as external dependency.

**Spec:** `docs/superpowers/specs/2026-04-18-annotate-command-design.md`

**Issue:** [#105](https://github.com/dreikanter/notesctl/issues/105)

---

## File Structure

- **Create** `internal/cli/annotate.go` — command, helpers, main flow
- **Create** `internal/cli/annotate_test.go` — all tests (pure helpers + integration via fake binary)
- **Modify** `CHANGELOG.md` — one entry for the next patch version
- **Modify** `README.md` — add `notesctl annotate` to usage section

---

## Task 1: Scaffold the command

**Files:**
- Create: `internal/cli/annotate.go`
- Create: `internal/cli/annotate_test.go`

- [ ] **Step 1: Write the failing test**

File: `internal/cli/annotate_test.go`

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runAnnotate(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	annotateCmd.ResetFlags()
	annotateCmd.Flags().String("model", annotateDefaultModel, "Claude model to use")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"annotate", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestAnnotateCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"annotate"})
	if err != nil {
		t.Fatalf("annotate command not registered: %v", err)
	}
	if cmd.Use == "" || !strings.HasPrefix(cmd.Use, "annotate") {
		t.Errorf("expected annotate Use, got %q", cmd.Use)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestAnnotateCommandRegistered -v`
Expected: FAIL — `annotateCmd` / `annotateDefaultModel` undefined.

- [ ] **Step 3: Write minimal implementation**

File: `internal/cli/annotate.go`

```go
package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// claudeBinary is the name or absolute path of the Claude Code CLI binary.
// Tests override this to point at a fake shell script.
var claudeBinary = "claude"

const annotateDefaultModel = "claude-haiku-4-5"

var annotateCmd = &cobra.Command{
	Use:   "annotate <id|type|query>",
	Short: "Fill empty frontmatter (title, description, tags) using Claude Code CLI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
}

func init() {
	annotateCmd.Flags().String("model", annotateDefaultModel, "Claude model to use")
	rootCmd.AddCommand(annotateCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestAnnotateCommandRegistered -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/annotate.go internal/cli/annotate_test.go
git commit -m "Scaffold notesctl annotate command"
```

---

## Task 2: Probe Claude CLI JSON envelope

**Files:**
- Modify: `internal/cli/annotate.go` (add a documented envelope shape as a comment + Go struct)

This is a one-time investigation. The spec requires inspecting what `claude -p --output-format json --json-schema ...` actually writes to stdout so the parser matches reality.

- [ ] **Step 1: Run the probe**

Run:
```bash
claude -p --model claude-haiku-4-5 \
  --output-format json \
  --json-schema '{"type":"object","properties":{"greeting":{"type":"string"}},"required":["greeting"],"additionalProperties":false}' \
  'Output a friendly greeting as the `greeting` field.'
```

Capture the stdout verbatim. It is expected to be a JSON object; common Claude Code CLI shape has fields like `type`, `subtype`, `is_error`, `result`, `session_id`, `total_cost_usd`. When `--json-schema` is supplied, the schema-conforming object is typically either the value of `result` (as a JSON string) or an object field in the envelope.

- [ ] **Step 2: Record the observed shape**

Add a comment at the top of `internal/cli/annotate.go` (below the package clause), pasting a pruned copy of the real stdout and naming the field that contains the schema-validated payload:

```go
// Claude CLI envelope (observed 2026-04-18, claude-haiku-4-5,
// --output-format json --json-schema):
//
//   {
//     "type": "result",
//     "subtype": "success",
//     "is_error": false,
//     "result": "{\"greeting\":\"hi\"}",   // <- schema-validated payload as a JSON string
//     ...
//   }
//
// If the observed shape differs, update annotateEnvelope and parseAnnotation below
// and refresh the testdata in annotateSampleEnvelope.
```

**If the observed shape is different from the template above (for example, the payload is nested as an object rather than a JSON string):**

- Update the `annotateEnvelope` struct defined in Task 4 to match the observed shape.
- Update the `parseAnnotation` implementation in Task 4 to extract the payload from the correct field.
- Update `annotateSampleEnvelope` (used by Task 4's test) so its bytes match the observed shape.

Do NOT guess. Paste the real observed JSON into the comment and derive the struct from it.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/annotate.go
git commit -m "Document observed Claude CLI JSON envelope shape"
```

---

## Task 3: Schema builder and empty-field detection

**Files:**
- Modify: `internal/cli/annotate.go`
- Modify: `internal/cli/annotate_test.go`

Two pure helpers:
- `annotateEmptyFields(f) []string` — returns a stable-order list of empty field names among `{"title", "description", "tags"}`.
- `buildAnnotateSchema(fields []string) string` — returns a JSON Schema requiring only the given fields.

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/annotate_test.go`:

```go
import (
	// ...existing imports...
	"encoding/json"

	"github.com/dreikanter/notesctl/note"
)

func TestAnnotateEmptyFieldsAllEmpty(t *testing.T) {
	got := annotateEmptyFields(note.FrontmatterFields{})
	want := []string{"title", "description", "tags"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnnotateEmptyFieldsPartial(t *testing.T) {
	f := note.FrontmatterFields{Title: "Existing"}
	got := annotateEmptyFields(f)
	want := []string{"description", "tags"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnnotateEmptyFieldsAllFilled(t *testing.T) {
	f := note.FrontmatterFields{Title: "T", Description: "D", Tags: []string{"x"}}
	got := annotateEmptyFields(f)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestBuildAnnotateSchemaAllFields(t *testing.T) {
	s := buildAnnotateSchema([]string{"title", "description", "tags"})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, s)
	}
	if parsed["type"] != "object" {
		t.Errorf("type = %v, want object", parsed["type"])
	}
	if parsed["additionalProperties"] != false {
		t.Errorf("additionalProperties = %v, want false", parsed["additionalProperties"])
	}

	req, ok := parsed["required"].([]any)
	if !ok || len(req) != 3 {
		t.Fatalf("required = %v, want 3 entries", parsed["required"])
	}

	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is not an object: %v", parsed["properties"])
	}
	for _, f := range []string{"title", "description", "tags"} {
		if _, ok := props[f]; !ok {
			t.Errorf("properties missing %q", f)
		}
	}
	tags, _ := props["tags"].(map[string]any)
	if tags["type"] != "array" {
		t.Errorf("tags.type = %v, want array", tags["type"])
	}
}

func TestBuildAnnotateSchemaTagsOnly(t *testing.T) {
	s := buildAnnotateSchema([]string{"tags"})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	props, _ := parsed["properties"].(map[string]any)
	if len(props) != 1 {
		t.Errorf("expected 1 property, got %d: %v", len(props), props)
	}
	if _, ok := props["tags"]; !ok {
		t.Errorf("missing tags property")
	}
}

// equalStrings reports whether two string slices have the same length and elements in order.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestAnnotateEmptyFields|TestBuildAnnotateSchema" -v`
Expected: FAIL — `annotateEmptyFields` and `buildAnnotateSchema` undefined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/annotate.go`:

```go
import (
	// add:
	"encoding/json"

	"github.com/dreikanter/notesctl/note"
)

// annotateEmptyFields returns the empty fields among {title, description, tags}
// in a deterministic order. "tags" counts as empty when the slice is empty.
func annotateEmptyFields(f note.FrontmatterFields) []string {
	var empty []string
	if f.Title == "" {
		empty = append(empty, "title")
	}
	if f.Description == "" {
		empty = append(empty, "description")
	}
	if len(f.Tags) == 0 {
		empty = append(empty, "tags")
	}
	return empty
}

// buildAnnotateSchema returns a JSON Schema string requiring only the given fields.
// Fields must be a subset of {"title", "description", "tags"}.
func buildAnnotateSchema(fields []string) string {
	props := map[string]any{}
	for _, f := range fields {
		switch f {
		case "title", "description":
			props[f] = map[string]string{"type": "string"}
		case "tags":
			props[f] = map[string]any{
				"type":  "array",
				"items": map[string]string{"type": "string"},
			}
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             fields,
		"additionalProperties": false,
	}
	b, _ := json.Marshal(schema)
	return string(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run "TestAnnotateEmptyFields|TestBuildAnnotateSchema" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/annotate.go internal/cli/annotate_test.go
git commit -m "Add empty-field detection and schema builder for annotate"
```

---

## Task 4: Response parser

**Files:**
- Modify: `internal/cli/annotate.go`
- Modify: `internal/cli/annotate_test.go`

Parses the full stdout of `claude` into an `annotateResult` struct. Use the envelope shape observed in Task 2.

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/annotate_test.go`:

```go
// annotateSampleEnvelope is the JSON shape observed in Task 2.
// If the shape differs on your system, update this fixture to match.
const annotateSampleEnvelope = `{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "result": "{\"title\":\"Weekly sync\",\"description\":\"Notes from the weekly team sync.\",\"tags\":[\"meeting\",\"weekly\"]}"
}`

func TestParseAnnotationHappyPath(t *testing.T) {
	res, err := parseAnnotation([]byte(annotateSampleEnvelope))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title != "Weekly sync" {
		t.Errorf("title = %q, want %q", res.Title, "Weekly sync")
	}
	if res.Description != "Notes from the weekly team sync." {
		t.Errorf("description = %q", res.Description)
	}
	if !equalStrings(res.Tags, []string{"meeting", "weekly"}) {
		t.Errorf("tags = %v", res.Tags)
	}
}

func TestParseAnnotationInvalidEnvelope(t *testing.T) {
	_, err := parseAnnotation([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid envelope")
	}
}

func TestParseAnnotationInvalidInnerJSON(t *testing.T) {
	bad := `{"type":"result","subtype":"success","is_error":false,"result":"not json"}`
	_, err := parseAnnotation([]byte(bad))
	if err == nil {
		t.Fatal("expected error for invalid inner JSON")
	}
}

func TestParseAnnotationErrorFlag(t *testing.T) {
	bad := `{"type":"result","subtype":"error","is_error":true,"result":"something broke"}`
	_, err := parseAnnotation([]byte(bad))
	if err == nil {
		t.Fatal("expected error when is_error=true")
	}
	if !strings.Contains(err.Error(), "something broke") {
		t.Errorf("error message should include server-side message: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestParseAnnotation -v`
Expected: FAIL — `parseAnnotation` / `annotateResult` undefined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/annotate.go`:

```go
// annotateEnvelope mirrors the outer JSON written by `claude -p --output-format json`.
// Only the fields we rely on are declared.
type annotateEnvelope struct {
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// annotateResult is the schema-validated payload carried by annotateEnvelope.Result.
type annotateResult struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// parseAnnotation unmarshals the claude CLI stdout into an annotateResult.
func parseAnnotation(raw []byte) (annotateResult, error) {
	var env annotateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return annotateResult{}, fmt.Errorf("cannot parse claude response: %w", err)
	}
	if env.IsError {
		return annotateResult{}, fmt.Errorf("claude returned error: %s", env.Result)
	}
	var res annotateResult
	if err := json.Unmarshal([]byte(env.Result), &res); err != nil {
		return annotateResult{}, fmt.Errorf("cannot parse claude response payload: %w", err)
	}
	return res, nil
}
```

Also add `"fmt"` to the imports if not already present.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestParseAnnotation -v`
Expected: PASS.

If any case fails because the real envelope observed in Task 2 has a different shape (e.g., the payload is a nested object, not a JSON string), adjust `annotateEnvelope`, `parseAnnotation`, and the test fixture together, then re-run.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/annotate.go internal/cli/annotate_test.go
git commit -m "Add claude response parser for annotate"
```

---

## Task 5: Merge helper

**Files:**
- Modify: `internal/cli/annotate.go`
- Modify: `internal/cli/annotate_test.go`

Non-destructive merge: only previously-empty fields are filled from the generated result; other fields (slug, public) preserved.

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/annotate_test.go`:

```go
func TestMergeAnnotationFillsEmpty(t *testing.T) {
	existing := note.FrontmatterFields{Slug: "meeting", Public: true}
	gen := annotateResult{
		Title:       "New",
		Description: "Generated desc",
		Tags:        []string{"a", "b"},
	}
	merged := mergeAnnotation(existing, gen)

	if merged.Title != "New" {
		t.Errorf("title = %q", merged.Title)
	}
	if merged.Description != "Generated desc" {
		t.Errorf("description = %q", merged.Description)
	}
	if !equalStrings(merged.Tags, []string{"a", "b"}) {
		t.Errorf("tags = %v", merged.Tags)
	}
	if merged.Slug != "meeting" {
		t.Errorf("slug should be preserved, got %q", merged.Slug)
	}
	if !merged.Public {
		t.Error("public should be preserved")
	}
}

func TestMergeAnnotationPreservesFilledFields(t *testing.T) {
	existing := note.FrontmatterFields{
		Title:       "Existing title",
		Description: "Existing desc",
		Tags:        []string{"keep"},
	}
	gen := annotateResult{
		Title:       "Should not win",
		Description: "Should not win",
		Tags:        []string{"bad"},
	}
	merged := mergeAnnotation(existing, gen)

	if merged.Title != "Existing title" {
		t.Errorf("title overwritten: %q", merged.Title)
	}
	if merged.Description != "Existing desc" {
		t.Errorf("description overwritten: %q", merged.Description)
	}
	if !equalStrings(merged.Tags, []string{"keep"}) {
		t.Errorf("tags overwritten: %v", merged.Tags)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestMergeAnnotation -v`
Expected: FAIL — `mergeAnnotation` undefined.

- [ ] **Step 3: Write minimal implementation**

Append to `internal/cli/annotate.go`:

```go
// mergeAnnotation fills empty fields in existing from gen.
// Non-empty fields in existing are preserved.
func mergeAnnotation(existing note.FrontmatterFields, gen annotateResult) note.FrontmatterFields {
	merged := existing
	if merged.Title == "" {
		merged.Title = gen.Title
	}
	if merged.Description == "" {
		merged.Description = gen.Description
	}
	if len(merged.Tags) == 0 && len(gen.Tags) > 0 {
		merged.Tags = gen.Tags
	}
	return merged
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestMergeAnnotation -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/annotate.go internal/cli/annotate_test.go
git commit -m "Add frontmatter merge helper for annotate"
```

---

## Task 6: Happy-path end-to-end with fake claude

**Files:**
- Modify: `internal/cli/annotate.go`
- Modify: `internal/cli/annotate_test.go`

Wire up the full command: read file, call `claude`, merge, rewrite. Happy-path test uses a fake `claude` shell script that echoes a canned envelope.

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/annotate_test.go`:

```go
import (
	// add:
	"fmt"
	"os"
	"path/filepath"
)

// writeFakeClaude writes a shell script named "claude" into a temp dir
// that echoes the given JSON envelope to stdout. Returns the script path.
func writeFakeClaude(t *testing.T, envelope string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := fmt.Sprintf("#!/bin/sh\ncat <<'EOF'\n%s\nEOF\n", envelope)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// noteWithOnlyBody writes a fresh note file in a temp root and returns the root + ref.
// The resulting note has no frontmatter — just body text.
func noteWithOnlyBody(t *testing.T, body string) (root, ref string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "2026", "04")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "20260418_9000.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, "9000"
}

func withClaudeBinary(t *testing.T, path string) {
	t.Helper()
	old := claudeBinary
	claudeBinary = path
	t.Cleanup(func() { claudeBinary = old })
}

func TestAnnotateFillsEmptyFields(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "# Weekly sync\n\nDiscussed Q2 roadmap, hiring, and launch dates.\n")
	withClaudeBinary(t, writeFakeClaude(t, annotateSampleEnvelope))

	out, err := runAnnotate(t, root, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/04/20260418_9000.md")
	if out != want {
		t.Errorf("stdout path = %q, want %q", out, want)
	}

	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, s := range []string{
		"title: Weekly sync",
		"description: Notes from the weekly team sync.",
		"tags: [meeting, weekly]",
	} {
		if !strings.Contains(content, s) {
			t.Errorf("expected %q in file, got:\n%s", s, content)
		}
	}
	if !strings.Contains(content, "# Weekly sync\n\nDiscussed Q2 roadmap") {
		t.Errorf("body missing or modified, got:\n%s", content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestAnnotateFillsEmptyFields -v`
Expected: FAIL — `runAnnotate` returns `"not implemented"`.

- [ ] **Step 3: Write the full command implementation**

Replace the body of `annotateCmd.RunE` and add supporting functions in `internal/cli/annotate.go`. The final file (after this step) imports:

```go
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)
```

Replace the command and append new helpers:

```go
const annotateSystemPrompt = `You are annotating a personal note stored as a markdown file.
Generate concise metadata for the provided note body, returning ONLY the fields required by the response schema.
- title: short title, <= 8 words.
- description: single-sentence summary, <= 140 characters.
- tags: 1-5 lowercase single-word slugs related to the content.`

var annotateCmd = &cobra.Command{
	Use:   "annotate <id|type|query>",
	Short: "Fill empty frontmatter (title, description, tags) using Claude Code CLI",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnnotate,
}

func runAnnotate(cmd *cobra.Command, args []string) error {
	model, _ := cmd.Flags().GetString("model")

	root := mustNotesPath()
	n, err := note.ResolveRef(root, args[0])
	if err != nil {
		return err
	}

	fullPath := filepath.Join(root, n.RelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("cannot read note: %w", err)
	}

	existing := note.ParseFrontmatterFields(data)
	body := note.StripFrontmatter(data)

	empty := annotateEmptyFields(existing)
	if len(empty) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return errors.New("note has no body content to annotate")
	}

	schema := buildAnnotateSchema(empty)
	out, err := runClaude(model, schema, string(body))
	if err != nil {
		return err
	}

	gen, err := parseAnnotation(out)
	if err != nil {
		return err
	}

	merged := mergeAnnotation(existing, gen)
	newContent := note.BuildFrontmatter(merged) + string(body)

	tmpPath := fullPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("cannot write note: %w", err)
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot rename note: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), fullPath)
	return nil
}

// runClaude executes the Claude Code CLI non-interactively and returns its stdout.
// Returns a clear error if the binary is not found or exits non-zero.
func runClaude(model, schema, prompt string) ([]byte, error) {
	bin, err := exec.LookPath(claudeBinary)
	if err != nil {
		return nil, errors.New("claude CLI not found in PATH")
	}

	args := []string{
		"-p",
		"--model", model,
		"--output-format", "json",
		"--json-schema", schema,
		"--append-system-prompt", annotateSystemPrompt,
		prompt,
	}

	c := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("claude failed: %s", msg)
	}
	return stdout.Bytes(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestAnnotateFillsEmptyFields -v`
Expected: PASS.

Also run the full file to confirm nothing else regressed:

Run: `go test ./internal/cli/ -v`
Expected: All previous tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/annotate.go internal/cli/annotate_test.go
git commit -m "Implement notesctl annotate end-to-end"
```

---

## Task 7: No-op when all fields are filled; empty body error

**Files:**
- Modify: `internal/cli/annotate_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/annotate_test.go`:

```go
// noteWithFrontmatter writes a note with the given frontmatter + body and returns (root, ref).
func noteWithFrontmatter(t *testing.T, fm, body string) (root, ref string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "2026", "04")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "20260418_9001.md")
	if err := os.WriteFile(path, []byte(fm+body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, "9001"
}

// fakeClaudeSentinel writes a fake claude script that fails if ever invoked.
// Used to assert the command never called claude.
func fakeClaudeSentinel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho SHOULD NOT BE CALLED >&2\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestAnnotateNoOpWhenAllFieldsFilled(t *testing.T) {
	fm := "---\ntitle: Existing\ndescription: Already here\ntags: [x, y]\n---\n\n"
	root, ref := noteWithFrontmatter(t, fm, "body content")
	withClaudeBinary(t, fakeClaudeSentinel(t))

	out, err := runAnnotate(t, root, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/04/20260418_9001.md")
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	data, _ := os.ReadFile(want)
	if string(data) != fm+"body content" {
		t.Errorf("file modified; got:\n%s", string(data))
	}
}

func TestAnnotateNoBodyErrors(t *testing.T) {
	fm := "---\ntitle: only title\n---\n\n"
	root, ref := noteWithFrontmatter(t, fm, "")
	withClaudeBinary(t, fakeClaudeSentinel(t))

	_, err := runAnnotate(t, root, ref)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "no body content") {
		t.Errorf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify the expected behavior**

Run: `go test ./internal/cli/ -run "TestAnnotateNoOp|TestAnnotateNoBody" -v`
Expected: PASS.

If either fails, the logic in Task 6's `runAnnotate` has a bug (likely an ordering issue between the `empty == 0` check and the empty-body check). Fix in `annotate.go`; do not loosen the test.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/annotate_test.go
git commit -m "Test annotate no-op and empty-body guards"
```

---

## Task 8: Claude invocation error paths

**Files:**
- Modify: `internal/cli/annotate_test.go`

Covers: binary not found, non-zero exit, malformed JSON. In each failure case, the file must be left untouched.

- [ ] **Step 1: Write the failing tests**

Append to `internal/cli/annotate_test.go`:

```go
func TestAnnotateClaudeNotFound(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")
	withClaudeBinary(t, filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := runAnnotate(t, root, ref)
	if err == nil {
		t.Fatal("expected error when claude binary missing")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAnnotateClaudeNonZeroExit(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho bad things happened >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	withClaudeBinary(t, script)

	_, err := runAnnotate(t, root, ref)
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
	if !strings.Contains(err.Error(), "bad things happened") {
		t.Errorf("stderr not surfaced: %v", err)
	}

	// File must be untouched.
	data, _ := os.ReadFile(filepath.Join(root, "2026/04/20260418_9000.md"))
	if string(data) != "body text here" {
		t.Errorf("file was modified: %q", string(data))
	}
}

func TestAnnotateMalformedJSON(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")
	withClaudeBinary(t, writeFakeClaude(t, `not valid json`))

	_, err := runAnnotate(t, root, ref)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "claude response") {
		t.Errorf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "2026/04/20260418_9000.md"))
	if string(data) != "body text here" {
		t.Errorf("file was modified: %q", string(data))
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run "TestAnnotateClaude|TestAnnotateMalformedJSON" -v`
Expected: PASS. If any fails, the bug is in `runClaude` or `parseAnnotation` (Tasks 4, 6). Fix there.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/annotate_test.go
git commit -m "Test annotate error paths (missing binary, non-zero exit, malformed JSON)"
```

---

## Task 9: Flag, schema, and body-preservation tests

**Files:**
- Modify: `internal/cli/annotate.go` (introduce an args-capturing hook for testing)
- Modify: `internal/cli/annotate_test.go`

These assertions need visibility into the exact `argv` passed to `claude`. Easiest mechanism: have the fake script dump its argv to a file via an env-var-declared path, which the test reads back.

- [ ] **Step 1: Add arg-dumping fake claude helper**

Append to `internal/cli/annotate_test.go`:

```go
// writeFakeClaudeRecording writes a fake claude script that dumps its argv
// one-per-line to argsPath, then echoes envelope to stdout.
func writeFakeClaudeRecording(t *testing.T, envelope, argsPath string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := fmt.Sprintf(`#!/bin/sh
for a in "$@"; do
  printf '%%s\n' "$a" >> %q
done
cat <<'EOF'
%s
EOF
`, argsPath, envelope)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestAnnotateModelFlagPropagates(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body for model test")
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	withClaudeBinary(t, writeFakeClaudeRecording(t, annotateSampleEnvelope, argsPath))

	_, err := runAnnotate(t, root, ref, "--model", "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	if !containsPair(argv, "--model", "claude-sonnet-4-6") {
		t.Errorf("expected --model claude-sonnet-4-6 in argv: %v", argv)
	}
}

func TestAnnotateSchemaOnlyContainsEmptyFields(t *testing.T) {
	// Start with title filled; only description + tags should be in schema.
	fm := "---\ntitle: Fixed title\n---\n\n"
	root, ref := noteWithFrontmatter(t, fm, "body for schema test")
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	// Envelope only needs to supply the two empty fields.
	env := `{"type":"result","subtype":"success","is_error":false,"result":"{\"description\":\"d\",\"tags\":[\"t\"]}"}`
	withClaudeBinary(t, writeFakeClaudeRecording(t, env, argsPath))

	if _, err := runAnnotate(t, root, ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	schema := nextValue(argv, "--json-schema")
	if schema == "" {
		t.Fatalf("--json-schema missing from argv: %v", argv)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("schema is not valid JSON: %v\n%s", err, schema)
	}
	req, _ := parsed["required"].([]any)
	if len(req) != 2 {
		t.Errorf("required should have 2 entries, got %v", req)
	}
	for _, f := range req {
		if f == "title" {
			t.Errorf("title should not be required (already filled): %v", req)
		}
	}
}

func TestAnnotatePreservesBody(t *testing.T) {
	body := "# heading\n\nparagraph one\n\n- list item 1\n- list item 2\n\nparagraph two with *emphasis* and `code`.\n"
	root, ref := noteWithOnlyBody(t, body)
	withClaudeBinary(t, writeFakeClaude(t, annotateSampleEnvelope))

	if _, err := runAnnotate(t, root, ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, "2026/04/20260418_9000.md"))
	// After the frontmatter closing "---\n\n", body must be byte-identical.
	idx := strings.Index(string(data), "\n---\n\n")
	if idx < 0 {
		t.Fatalf("could not find frontmatter terminator in:\n%s", string(data))
	}
	got := string(data)[idx+len("\n---\n\n"):]
	if got != body {
		t.Errorf("body modified.\ngot:\n%q\nwant:\n%q", got, body)
	}
}

// containsPair reports whether argv contains flag immediately followed by value.
func containsPair(argv []string, flag, value string) bool {
	for i := 0; i < len(argv)-1; i++ {
		if argv[i] == flag && argv[i+1] == value {
			return true
		}
	}
	return false
}

// nextValue returns the element immediately after the first occurrence of flag, or "".
func nextValue(argv []string, flag string) string {
	for i := 0; i < len(argv)-1; i++ {
		if argv[i] == flag {
			return argv[i+1]
		}
	}
	return ""
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run "TestAnnotateModelFlag|TestAnnotateSchemaOnly|TestAnnotatePreservesBody" -v`
Expected: PASS.

- [ ] **Step 3: Run the full test suite and lint**

Run: `go test ./...`
Expected: PASS across the repository.

Run: `make lint`
Expected: No warnings.

Fix any fallout inline. Common issues: unused imports, `_ = err` warnings, package comment missing on a new file.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/annotate_test.go
git commit -m "Test annotate --model flag, schema field set, and body preservation"
```

---

## Task 10: Docs and changelog

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Update README**

Add an `annotate` example in `README.md` after the `edit` block (around the "Open a note in $EDITOR" section), matching the existing comment-then-example style:

```markdown
# Fill empty frontmatter (title, description, tags) using Claude Code CLI
notesctl annotate 8823
notesctl annotate meeting --model claude-sonnet-4-6
```

- [ ] **Step 2: Update CHANGELOG**

Compute the next version:

```bash
git describe --tags
```

Increment the patch number (e.g., `v0.1.68` → `v0.1.69`). Find the upcoming PR number (either the one you'll create, or ask `gh` after pushing the branch).

Prepend a new section at the top of `CHANGELOG.md`, immediately after the `# Changelog` heading:

```markdown
## [0.1.69] - 2026-04-18

### Added

- `notesctl annotate <ref>` command that uses Claude Code CLI to fill empty frontmatter fields (`title`, `description`, `tags`). Defaults to `claude-haiku-4-5`; override with `--model`. Non-destructive: existing field values are never overwritten. ([#N])
```

Add the PR-number link footer entry at the bottom of `CHANGELOG.md` (sorted with the other `[#…]: …` entries):

```markdown
[#N]: https://github.com/dreikanter/notesctl/pull/N
```

Replace `N` with the actual PR number. Replace `0.1.69` if `git describe --tags` gives a different next patch version. Replace the date if today's date is different.

- [ ] **Step 3: Verify build and full test suite one more time**

```bash
make build
make test
make lint
```

Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "Document notesctl annotate command"
```

---

## Post-implementation checklist (not part of TDD loop)

- [ ] All tests pass: `go test ./...`
- [ ] Lint clean: `make lint`
- [ ] Binary builds: `make build`
- [ ] `./notesctl annotate --help` shows the expected short description
- [ ] End-to-end sanity: run `./notesctl annotate <some-ref>` against a real note with real `claude` in PATH; inspect the file; confirm it added only empty fields
- [ ] CHANGELOG entry uses the version that `git describe --tags` will produce on merge
- [ ] PR body uses `.github/pull_request_template.md`
