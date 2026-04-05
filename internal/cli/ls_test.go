package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runLs(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)
	lsCmd.ResetFlags()
	lsCmd.Flags().Int("limit", 0, "maximum number of notes to list (0 = no limit)")
	lsCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	lsCmd.Flags().String("slug", "", "filter by slug")
	lsCmd.Flags().StringSlice("tag", nil, "filter by frontmatter tag (repeatable, AND logic)")
	lsCmd.Flags().String("name", "", "filter by filename fragment (case-insensitive substring)")
	lsCmd.Flags().Bool("today", false, "filter notes created today")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"ls", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestLsNoArgs(t *testing.T) {
	out, err := runLs(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4:\n%s", len(lines), out)
	}
}

func TestLsWithTag(t *testing.T) {
	out, err := runLs(t, "--tag", "work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), out)
	}
}

func TestLsTagNoMatch(t *testing.T) {
	out, err := runLs(t, "--tag", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestLsTagAndType(t *testing.T) {
	out, err := runLs(t, "--tag", "work", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "todo") {
		t.Errorf("expected todo note, got %q", lines[0])
	}
}

func TestLsTagAndLimit(t *testing.T) {
	out, err := runLs(t, "--tag", "work", "--limit", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
}

func TestLsMultipleTags(t *testing.T) {
	out, err := runLs(t, "--tag", "work", "--tag", "planning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (only todo has both work+planning):\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "todo") {
		t.Errorf("expected todo note, got %q", lines[0])
	}
}

func TestLsMultipleTagsCommaSeparated(t *testing.T) {
	out, err := runLs(t, "--tag", "work,meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (only meeting note has both work+meeting):\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "meeting") {
		t.Errorf("expected meeting note, got %q", lines[0])
	}
}

func TestLsTagAndTypeNoOverlap(t *testing.T) {
	out, err := runLs(t, "--tag", "meeting", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != "" {
		t.Errorf("expected empty output (no todo with meeting tag), got %q", out)
	}
}

func TestLsWithName(t *testing.T) {
	out, err := runLs(t, "--name", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "meeting") {
		t.Errorf("expected meeting note, got %q", lines[0])
	}
}

func TestLsNameNoMatch(t *testing.T) {
	out, err := runLs(t, "--name", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestLsToday(t *testing.T) {
	// testdata notes are all in the past; --today should return nothing
	out, err := runLs(t, "--today")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for --today on past testdata, got %q", out)
	}
}

func TestLsUnlimitedByDefault(t *testing.T) {
	out, err := runLs(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected all 4 testdata notes without limit, got %d:\n%s", len(lines), out)
	}
}

func TestLsNameAndType(t *testing.T) {
	out, err := runLs(t, "--name", "8814", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "8814") {
		t.Errorf("expected note 8814, got %q", lines[0])
	}
}

func TestLsOutputsAbsolutePaths(t *testing.T) {
	out, err := runLs(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if !filepath.IsAbs(line) {
			t.Errorf("expected absolute path, got %q", line)
		}
		if !strings.HasPrefix(line, root) {
			t.Errorf("expected path under %s, got %q", root, line)
		}
	}
}

func TestLsMultipleTypes(t *testing.T) {
	// "todo" exists; "backlog" does not — union should return the 1 todo note
	out, err := runLs(t, "--type", "todo", "--type", "backlog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "todo") {
		t.Errorf("expected todo note, got %q", lines[0])
	}
}

func TestLsSlug(t *testing.T) {
	out, err := runLs(t, "--slug", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "meeting") {
		t.Errorf("expected meeting note, got %q", lines[0])
	}
}
