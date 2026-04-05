package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
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
	target := filepath.Join(root, "2026/01/20260106_8823_999.md")
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

	out, err := runResolve(t, root, "2026/01/20260106_8823_999.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestResolveByIDWithWhitespace(t *testing.T) {
	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260106_8823_999.md")

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

func TestResolveTodayFilterExcludesOldNotes(t *testing.T) {
	root := testdataPath(t)
	// "meeting" slug exists but is from 20260104, not today
	_, err := runResolve(t, root, "--today", "meeting")
	if err == nil {
		t.Fatal("expected error when --today excludes matching note")
	}
}

func TestResolveTodayFilterMatchesToday(t *testing.T) {
	root := t.TempDir()
	today := time.Now().Format("20060102")
	month := today[:6]
	dir := filepath.Join(root, today[:4], month[4:6])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fname := today + "_0001_daily.md"
	if err := os.WriteFile(filepath.Join(dir, fname), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runResolve(t, root, "--today", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(dir, fname)
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
