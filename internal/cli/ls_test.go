package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func runLs(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)
	lsTags = nil
	lsType = ""
	lsLimit = 20
	lsCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	lsCmd.SetOut(buf)
	lsCmd.SetErr(buf)
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
