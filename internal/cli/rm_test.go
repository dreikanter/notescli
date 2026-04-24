package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runRm(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	rmCmd.ResetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"rm", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestRmByID(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runRm(t, root, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("got %q, want %q", out, target)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmNonExistentErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runRm(t, root, "9999")
	if err == nil {
		t.Fatal("expected error for non-existent id, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}
}

func TestRmNonIntegerArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runRm(t, root, "meeting")
	if err == nil {
		t.Fatal("expected error for non-integer id, got nil")
	}
}
