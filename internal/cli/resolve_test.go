package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runResolve(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(append([]string{"resolve", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(stdout.String()), err
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

func TestResolveByIDWithWhitespace(t *testing.T) {
	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260106_8823.md")

	tests := []struct {
		name  string
		query string
	}{
		{"trailing space", "8823 "},
		{"leading space", " 8823"},
		{"both spaces", " 8823 "},
		{"trailing tab", "8823\t"},
		{"trailing newline", "8823\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runResolve(t, root, tt.query)
			if err != nil {
				t.Fatalf("unexpected error for query %q: %v", tt.query, err)
			}
			if out != want {
				t.Errorf("got %q, want %q", out, want)
			}
		})
	}
}

func TestResolveNonExistentErrors(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "9999")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}
