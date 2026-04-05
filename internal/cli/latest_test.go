package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func testdataPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}
	return abs
}

func runLatest(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	// Reset flags to avoid state leaking between tests.
	latestCmd.ResetFlags()
	latestCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	latestCmd.Flags().String("slug", "", "filter by slug")
	latestCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	latestCmd.Flags().Bool("today", false, "filter to notes created today")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"latest", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestLatestNoArgs(t *testing.T) {
	out, err := runLatest(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestLatestWithType(t *testing.T) {
	out, err := runLatest(t, "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestLatestWithSlug(t *testing.T) {
	out, err := runLatest(t, "--slug", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestLatestWithTag(t *testing.T) {
	out, err := runLatest(t, "--tag", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestLatestCombinedFilters(t *testing.T) {
	out, err := runLatest(t, "--tag", "work", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestLatestSlugNotFound(t *testing.T) {
	_, err := runLatest(t, "--slug", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent slug, got nil")
	}
}

func TestLatestTypeNotFound(t *testing.T) {
	_, err := runLatest(t, "--type", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent type, got nil")
	}
}

func TestLatestWithTodayNoMatch(t *testing.T) {
	_, err := runLatest(t, "--today")
	if err == nil {
		t.Fatal("expected error when no notes exist for today, got nil")
	}
}
