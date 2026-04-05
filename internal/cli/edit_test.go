package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runEdit(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(append([]string{"edit", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(stdout.String() + stderr.String()), err
}

func TestEditOpensEditor(t *testing.T) {
	root := testdataPath(t)

	// Use a script that writes the received path to a temp file
	marker := filepath.Join(t.TempDir(), "edited")
	script := filepath.Join(t.TempDir(), "fake-editor.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" > "+marker+"\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", script)

	_, err = runEdit(t, root, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker file not created: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	if strings.TrimSpace(string(got)) != want {
		t.Errorf("editor received %q, want %q", strings.TrimSpace(string(got)), want)
	}
}

func TestEditPrefersVisual(t *testing.T) {
	root := testdataPath(t)

	marker := filepath.Join(t.TempDir(), "edited")
	script := filepath.Join(t.TempDir(), "fake-editor.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" > "+marker+"\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	badScript := filepath.Join(t.TempDir(), "bad-editor.sh")
	err = os.WriteFile(badScript, []byte("#!/bin/sh\nexit 1\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", script)
	t.Setenv("EDITOR", badScript)

	_, err = runEdit(t, root, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker file not created: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	if strings.TrimSpace(string(got)) != want {
		t.Errorf("editor received %q, want %q", strings.TrimSpace(string(got)), want)
	}
}

func TestEditNoEditorErrors(t *testing.T) {
	root := testdataPath(t)

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	_, err := runEdit(t, root, "8823")
	if err == nil {
		t.Fatal("expected error when no editor configured, got nil")
	}
	if !strings.Contains(err.Error(), "no editor configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEditNonExistentRefErrors(t *testing.T) {
	root := testdataPath(t)

	t.Setenv("EDITOR", "true")

	_, err := runEdit(t, root, "9999")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}

func TestEditBySlug(t *testing.T) {
	root := testdataPath(t)

	marker := filepath.Join(t.TempDir(), "edited")
	script := filepath.Join(t.TempDir(), "fake-editor.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" > "+marker+"\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", script)

	_, err = runEdit(t, root, "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker file not created: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if strings.TrimSpace(string(got)) != want {
		t.Errorf("editor received %q, want %q", strings.TrimSpace(string(got)), want)
	}
}

func TestEditByType(t *testing.T) {
	root := testdataPath(t)

	marker := filepath.Join(t.TempDir(), "edited")
	script := filepath.Join(t.TempDir(), "fake-editor.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" > "+marker+"\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", script)

	_, err = runEdit(t, root, "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker file not created: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	if strings.TrimSpace(string(got)) != want {
		t.Errorf("editor received %q, want %q", strings.TrimSpace(string(got)), want)
	}
}
