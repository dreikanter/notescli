package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyTestdata copies testdata into a temporary directory so tests
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

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(append([]string{"append", "--path", root}, args...))

	execErr := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), execErr
}

func TestAppendByID(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "appended text", "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}

	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "appended text") {
		t.Error("appended text not found in file")
	}
}

func TestAppendNonIntegerArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "meeting")
	if err == nil {
		t.Fatal("expected error for non-integer id, got nil")
	}
}

func TestAppendNonExistentNote(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "9999")
	if err == nil {
		t.Fatal("expected error for non-existent note, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got: %v", err)
	}
}

func TestAppendEmptyStdin(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823_999.md")
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
	target := filepath.Join(root, "2026/01/20260106_8823_999.md")
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

	data, _ := os.ReadFile(filepath.Join(root, "2026/01/20260106_8823_999.md"))
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

func TestAppendNoArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text")
	if err == nil {
		t.Fatal("expected error when no target specified, got nil")
	}
}
