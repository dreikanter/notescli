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
