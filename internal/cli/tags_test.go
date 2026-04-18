package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runTags(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"tags", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func writeTagsTestNote(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTagsEmptyStore(t *testing.T) {
	root := t.TempDir()
	out, err := runTags(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

func TestTagsMergedSourcesSorted(t *testing.T) {
	root := t.TempDir()
	writeTagsTestNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, planning]\n---\n\nHere is #coffee and #work again.\n")
	writeTagsTestNote(t, root, "2026/01/20260102_1002.md",
		"no fm, just #tea and #work.\n")

	out, err := runTags(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Split(out, "\n")
	want := []string{"coffee", "planning", "tea", "work"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTagsIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	writeTagsTestNote(t, root, "2026/01/20260101_1001.md",
		"kept #real\n```\n#should-not-appear\n```\nalso #done\n")

	out, err := runTags(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "should-not-appear") {
		t.Errorf("expected code-block hashtag to be excluded, got:\n%s", out)
	}
	if !strings.Contains(out, "real") || !strings.Contains(out, "done") {
		t.Errorf("expected real and done tags, got:\n%s", out)
	}
}
