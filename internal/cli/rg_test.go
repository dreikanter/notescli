package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func requireRg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not installed, skipping")
	}
}

func runRg(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	origPath := notesPath
	notesPath = root
	t.Cleanup(func() { notesPath = origPath })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w

	rootCmd.SetArgs(append([]string{"rg"}, args...))
	execErr := rootCmd.Execute()

	w.Close()
	os.Stdout = origStdout

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	return strings.TrimSpace(string(buf[:n])), execErr
}

func TestRgFindsMatch(t *testing.T) {
	requireRg(t)
	out, err := runRg(t, "-l", "Todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "20260102_8814.todo.md") {
		t.Errorf("expected output to contain todo note, got %q", out)
	}
}

func TestRgNoMatch(t *testing.T) {
	requireRg(t)
	_, err := runRg(t, "-l", "zzz_no_match_zzz")
	if err == nil {
		t.Fatal("expected error for no matches, got nil")
	}
}

func TestRgExcludesNonMarkdown(t *testing.T) {
	requireRg(t)
	out, err := runRg(t, "-l", "skipped")
	if err == nil {
		t.Fatalf("expected no matches, got output: %q", out)
	}
}

func TestRgCaseInsensitive(t *testing.T) {
	requireRg(t)
	out, err := runRg(t, "-il", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "20260102_8814.todo.md") {
		t.Errorf("expected output to contain todo note, got %q", out)
	}
}

func TestRgHelp(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}

	rootCmd.SetOut(w)
	t.Cleanup(func() { rootCmd.SetOut(nil) })

	rootCmd.SetArgs([]string{"rg", "--help"})
	execErr := rootCmd.Execute()
	w.Close()

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	out := string(buf[:n])
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "rg") {
		t.Errorf("expected help output to contain 'rg', got: %q", out)
	}
	if !strings.Contains(out, "--glob") {
		t.Errorf("expected help output to mention injected flags, got: %q", out)
	}
}
