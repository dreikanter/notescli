package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreikanter/notes-cli/note"
)

func runAnnotate(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	annotateCmd.ResetFlags()
	registerAnnotateFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"annotate", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestAnnotateEmptyFieldsAllEmpty(t *testing.T) {
	got := annotateEmptyFields(note.StoreMeta{})
	want := []string{"title", "description", "tags"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnnotateEmptyFieldsPartial(t *testing.T) {
	m := note.StoreMeta{Title: "Existing"}
	got := annotateEmptyFields(m)
	want := []string{"description", "tags"}
	if !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnnotateEmptyFieldsAllFilled(t *testing.T) {
	m := note.StoreMeta{Title: "T", Description: "D", Tags: []string{"x"}}
	got := annotateEmptyFields(m)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestBuildAnnotateSchemaAllFields(t *testing.T) {
	s, err := buildAnnotateSchema([]string{"title", "description", "tags"})
	if err != nil {
		t.Fatalf("buildAnnotateSchema: %v", err)
	}

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
	s, err := buildAnnotateSchema([]string{"tags"})
	if err != nil {
		t.Fatalf("buildAnnotateSchema: %v", err)
	}
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

func TestAnnotateCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"annotate"})
	if err != nil {
		t.Fatalf("annotate command not registered: %v", err)
	}
	if cmd.Use == "" || !strings.HasPrefix(cmd.Use, "annotate") {
		t.Errorf("expected annotate Use, got %q", cmd.Use)
	}
}

// annotateSampleEnvelope mirrors the stdout of
// `claude -p --output-format json --json-schema ...`.
const annotateSampleEnvelope = `{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "result": "Metadata generated.",
  "structured_output": {"title": "Weekly sync", "description": "Notes from the weekly team sync.", "tags": ["meeting", "weekly"]}
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

func TestParseAnnotationMissingStructuredOutput(t *testing.T) {
	bad := `{"type":"result","subtype":"success","is_error":false,"result":"Metadata generated."}`
	_, err := parseAnnotation([]byte(bad))
	if err == nil {
		t.Fatal("expected error when structured_output is absent")
	}
	if !strings.Contains(err.Error(), "structured_output") {
		t.Errorf("error should name structured_output: %v", err)
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

func TestMergeAnnotationFillsEmpty(t *testing.T) {
	existing := note.StoreMeta{Slug: "meeting", Public: true}
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
	existing := note.StoreMeta{
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
		"tags:\n    - meeting\n    - weekly\n",
	} {
		if !strings.Contains(content, s) {
			t.Errorf("expected %q in file, got:\n%s", s, content)
		}
	}
	if !strings.Contains(content, "# Weekly sync\n\nDiscussed Q2 roadmap") {
		t.Errorf("body missing or modified, got:\n%s", content)
	}
}

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

func TestAnnotateTimeout(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	withClaudeBinary(t, script)

	_, err := runAnnotate(t, root, ref, "--timeout", "100ms")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

// --timeout 0 disables the deadline, so a fake claude that sleeps briefly
// (shorter than any reasonable default) still completes successfully.
func TestAnnotateTimeoutZeroDisables(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	body := fmt.Sprintf("#!/bin/sh\nsleep 0.1\ncat <<'EOF'\n%s\nEOF\n", annotateSampleEnvelope)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	withClaudeBinary(t, script)

	if _, err := runAnnotate(t, root, ref, "--timeout", "0"); err != nil {
		t.Fatalf("unexpected error with --timeout 0: %v", err)
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

// TestAnnotateClaudeEmptyStderrIncludesStdout exercises the fallback path
// for opaque failures: when claude exits non-zero with nothing on stderr,
// the error should include the exit code and a snippet of stdout instead
// of a bare "exit status 1".
func TestAnnotateClaudeEmptyStderrIncludesStdout(t *testing.T) {
	root, ref := noteWithOnlyBody(t, "body text here")

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho partial output on stdout\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	withClaudeBinary(t, script)

	_, err := runAnnotate(t, root, ref)
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
	msg := err.Error()
	if !strings.Contains(msg, "exit 3") {
		t.Errorf("expected exit code in error, got: %v", err)
	}
	if !strings.Contains(msg, "partial output on stdout") {
		t.Errorf("expected stdout snippet in error, got: %v", err)
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
	env := `{"type":"result","subtype":"success","is_error":false,"result":"ok","structured_output":{"description":"d","tags":["t"]}}`
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

func TestAnnotateMaxCharsTruncates(t *testing.T) {
	body := strings.Repeat("a", 5000)
	root, ref := noteWithOnlyBody(t, body)
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	withClaudeBinary(t, writeFakeClaudeRecording(t, annotateSampleEnvelope, argsPath))

	_, err := runAnnotate(t, root, ref, "--max-chars", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	// The body is the final positional argv entry.
	sentBody := argv[len(argv)-1]
	if len(sentBody) != 100 {
		t.Errorf("expected body truncated to 100 chars, got %d: %q", len(sentBody), sentBody)
	}
}

func TestAnnotateMaxCharsZeroLeavesBodyUntouched(t *testing.T) {
	body := strings.Repeat("a", 5000)
	root, ref := noteWithOnlyBody(t, body)
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	withClaudeBinary(t, writeFakeClaudeRecording(t, annotateSampleEnvelope, argsPath))

	_, err := runAnnotate(t, root, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	sentBody := argv[len(argv)-1]
	if len(sentBody) != len(body) {
		t.Errorf("expected full body (%d chars) sent, got %d", len(body), len(sentBody))
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
