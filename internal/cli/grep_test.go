package cli

import (
	"os"
	"strings"
	"testing"
)

func runGrep(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	// Set notesPath directly since DisableFlagParsing prevents
	// --path from being parsed when passed after "grep".
	origPath := notesPath
	notesPath = root
	t.Cleanup(func() { notesPath = origPath })

	// Capture stdout since grep writes to os.Stdout directly.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w

	rootCmd.SetArgs(append([]string{"grep"}, args...))
	execErr := rootCmd.Execute()

	w.Close()
	os.Stdout = origStdout

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	return strings.TrimSpace(string(buf[:n])), execErr
}

func TestGrepFindsMatch(t *testing.T) {
	out, err := runGrep(t, "-rl", "Todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "20260102_8814_todo.md") {
		t.Errorf("expected output to contain todo note, got %q", out)
	}
}

func TestGrepNoMatch(t *testing.T) {
	_, err := runGrep(t, "-rl", "zzz_no_match_zzz")
	if err == nil {
		t.Fatal("expected error for no matches, got nil")
	}
}

func TestGrepCaseInsensitive(t *testing.T) {
	out, err := runGrep(t, "-ril", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "20260102_8814_todo.md") {
		t.Errorf("expected output to contain todo note, got %q", out)
	}
}
