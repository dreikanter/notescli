package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTestdata copies testdata into a temporary directory so append tests
// can modify files without affecting other tests.
func copyTestdata(t *testing.T) string {
	t.Helper()
	src := testdataPath(t)
	dst := t.TempDir()

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("cannot copy testdata: %v", err)
	}
	return dst
}

func runAppend(t *testing.T, root string, stdin string, args ...string) (string, error) {
	t.Helper()

	appendCmd.ResetFlags()
	appendCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	appendCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"append", "--path", root}, args...))

	// Replace stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		_, _ = io.WriteString(w, stdin)
		w.Close()
	}()

	execErr := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), execErr
}

func TestAppendByID(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "appended text", "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}

	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "appended text") {
		t.Error("appended text not found in file")
	}
}

func TestAppendByAbsolutePath(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")
	out, err := runAppend(t, root, "path append", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// EvalSymlinks may resolve /var → /private/var on macOS
	resolved, _ := filepath.EvalSymlinks(target)
	if out != resolved {
		t.Errorf("got output %q, want %q", out, resolved)
	}

	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), "path append") {
		t.Error("appended text not found in file")
	}
}

func TestAppendByRelativePath(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")

	out, err := runAppend(t, root, "rel append", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolved, _ := filepath.EvalSymlinks(target)
	if out != resolved {
		t.Errorf("got output %q, want %q", out, resolved)
	}
}

func TestAppendByTagFilter(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "tag append", "--tag", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}

	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "tag append") {
		t.Error("appended text not found in file")
	}
}

func TestAppendBySlugFilter(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "slug append", "--slug", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}
}

func TestAppendByTypeFilter(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "type append", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}
}

func TestAppendPathOutsideRoot(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "escape attempt", "/tmp/evil.md")
	if err == nil {
		t.Fatal("expected error for path outside notes directory, got nil")
	}
}

func TestAppendNonExistentNote(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "9999")
	if err == nil {
		t.Fatal("expected error for non-existent note, got nil")
	}
}

func TestAppendEmptyStdin(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")
	before, _ := os.ReadFile(target)

	_, err := runAppend(t, root, "", "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, _ := os.ReadFile(target)
	if string(before) != string(after) {
		t.Error("file should not have changed with empty stdin")
	}
}

func TestAppendWhitespaceOnlyStdin(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")
	before, _ := os.ReadFile(target)

	_, err := runAppend(t, root, "   \n\n  \t  ", "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, _ := os.ReadFile(target)
	if string(before) != string(after) {
		t.Error("file should not have changed with whitespace-only stdin")
	}
}

func TestAppendMultipleProducesSeparation(t *testing.T) {
	root := copyTestdata(t)

	_, _ = runAppend(t, root, "first fragment", "8823")
	_, _ = runAppend(t, root, "second fragment", "8823")

	data, _ := os.ReadFile(filepath.Join(root, "2026/01/20260106_8823.md"))
	content := string(data)

	if !strings.Contains(content, "first fragment") {
		t.Error("first fragment not found")
	}
	if !strings.Contains(content, "second fragment") {
		t.Error("second fragment not found")
	}

	// Fragments should be separated by a blank line
	idx1 := strings.Index(content, "first fragment")
	idx2 := strings.Index(content, "second fragment")
	between := content[idx1+len("first fragment") : idx2]
	if !strings.Contains(between, "\n\n") {
		t.Errorf("fragments not separated by blank line, got between: %q", between)
	}
}

func TestAppendPositionalArgWithFilterErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "8823", "--type", "todo")
	if err == nil {
		t.Fatal("expected error when combining positional arg and filter flags, got nil")
	}
}

func TestAppendNoTargetErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text")
	if err == nil {
		t.Fatal("expected error when no target specified, got nil")
	}
}
