package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runLs(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)
	lsCmd.ResetFlags()
	registerLsFlags()

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
	for _, line := range lines {
		if !allDigits(line) {
			t.Fatalf("expected integer ID per line, got %q", line)
		}
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

func TestLsWithTagMixedCase(t *testing.T) {
	out, err := runLs(t, "--tag", "WORK")
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
	if lines[0] != "8814" {
		t.Errorf("expected ID 8814, got %q", lines[0])
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

func TestLsMultipleTagsAND(t *testing.T) {
	out, err := runLs(t, "--tag", "work", "--tag", "planning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1 (only todo has both work+planning):\n%s", len(lines), out)
	}
	if lines[0] != "8814" {
		t.Errorf("expected ID 8814, got %q", lines[0])
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
	if lines[0] != "8818" {
		t.Errorf("expected ID 8818, got %q", lines[0])
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

func TestLsOutputsIntegerIDs(t *testing.T) {
	out, err := runLs(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if !allDigits(line) {
			t.Errorf("expected integer ID per line, got %q", line)
		}
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
	if lines[0] != "8818" {
		t.Errorf("expected ID 8818, got %q", lines[0])
	}
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
