package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dreikanter/notes-cli/note"
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

func TestAnnotateCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"annotate"})
	if err != nil {
		t.Fatalf("annotate command not registered: %v", err)
	}
	if cmd.Use == "" || !strings.HasPrefix(cmd.Use, "annotate") {
		t.Errorf("expected annotate Use, got %q", cmd.Use)
	}
}

// annotateSampleEnvelope is the JSON shape observed in Task 2.
// NOTE: Task 2 could not run the live probe (sandbox-blocked); this fixture
// is based on the documented claude -p --output-format json shape. If the
// actual CLI shape differs, adjust this fixture AND the annotateEnvelope /
// parseAnnotation implementation together.
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
