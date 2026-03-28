package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runResolve(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"resolve", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestResolveByID(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestResolveBySlug(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestResolveByType(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestResolveByAbsolutePath(t *testing.T) {
	root := testdataPath(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")
	out, err := runResolve(t, root, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("got %q, want %q", out, target)
	}
}

func TestResolveByRelativePath(t *testing.T) {
	root := testdataPath(t)

	t.Chdir(root)

	out, err := runResolve(t, root, "2026/01/20260106_8823.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestResolveNonExistentErrors(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "9999")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}
