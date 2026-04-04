package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	appendCmd.Flags().String("type", "", "filter by note type")
	appendCmd.Flags().String("slug", "", "filter by slug")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	appendCmd.Flags().Bool("create", false, "create note if no match found")
	appendCmd.Flags().Bool("today", false, "append to today's note or create a new one")
	appendCmd.Flags().String("title", "", "title for frontmatter (requires --create or --today)")
	appendCmd.Flags().String("description", "", "description for frontmatter (requires --create or --today)")

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

	if out != target {
		t.Errorf("got output %q, want %q", out, target)
	}

	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), "path append") {
		t.Error("appended text not found in file")
	}
}

func TestAppendByRelativePath(t *testing.T) {
	root := copyTestdata(t)

	// chdir into the temp root so a relative path containing "/" works
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("cannot chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	out, err := runAppend(t, root, "rel append", "2026/01/20260106_8823.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.md")
	if out != want {
		t.Errorf("got output %q, want %q", out, want)
	}

	data, _ := os.ReadFile(filepath.Join(root, "2026/01/20260106_8823.md"))
	if !strings.Contains(string(data), "rel append") {
		t.Error("appended text not found in file")
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

func TestAppendCreateNewNote(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "created content", "--type", "weekly", "--create")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out == "" {
		t.Fatal("expected output path, got empty")
	}

	// Verify file was created and contains the appended content
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read created file: %v", err)
	}
	if !strings.Contains(string(data), "created content") {
		t.Error("appended content not found in created file")
	}

	// Verify filename contains .weekly.
	if !strings.Contains(filepath.Base(out), ".weekly.md") {
		t.Errorf("expected .weekly.md extension, got %s", filepath.Base(out))
	}
}

func TestAppendCreateWithSlug(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "slugged", "--slug", "standup", "--create")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := filepath.Base(out)
	if !strings.Contains(base, "_standup") {
		t.Errorf("expected slug in filename, got %s", base)
	}

	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "slugged") {
		t.Error("appended content not found in created file")
	}
}

func TestAppendCreateWithTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "tagged content", "--tag", "work", "--tag", "daily", "--create")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	content := string(data)

	if !strings.Contains(content, "tags: [work, daily]") {
		t.Errorf("expected tags in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "tagged content") {
		t.Error("appended content not found in created file")
	}
}

func TestAppendCreateWithTitleAndDescription(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "body text", "--type", "backlog", "--create",
		"--title", "Daily TODO", "--description", "auto-generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	content := string(data)

	if !strings.Contains(content, "title: Daily TODO") {
		t.Errorf("expected title in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "description: auto-generated") {
		t.Errorf("expected description in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "body text") {
		t.Error("appended content not found")
	}
}

func TestAppendCreateAppendsToExistingMatch(t *testing.T) {
	root := copyTestdata(t)

	// There's already a todo note: 20260102_8814.todo.md
	target := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	before, _ := os.ReadFile(target)

	out, err := runAppend(t, root, "extra text", "--type", "todo", "--create")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("expected to append to existing %q, got %q", target, out)
	}

	after, _ := os.ReadFile(target)
	if !strings.Contains(string(after), "extra text") {
		t.Error("appended text not found in existing file")
	}

	// Verify original content is preserved
	if !strings.Contains(string(after), string(before[:len(before)-1])) {
		t.Error("original content was not preserved")
	}
}

func TestAppendCreateWithPositionalArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "8823", "--create")
	if err == nil {
		t.Fatal("expected error when combining --create with positional arg, got nil")
	}
}

func TestAppendTitleWithoutCreateErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--title", "Oops")
	if err == nil {
		t.Fatal("expected error when using --title without --create, got nil")
	}
}

func TestAppendDescriptionWithoutCreateErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--description", "Oops")
	if err == nil {
		t.Fatal("expected error when using --description without --create, got nil")
	}
}

func TestAppendCreateWithoutFiltersErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--create")
	if err == nil {
		t.Fatal("expected error when using --create without filter flags, got nil")
	}
}

func TestAppendCreateUnknownTypeErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "invalid", "--create")
	if err == nil {
		t.Fatal("expected error for unknown note type, got nil")
	}
}

func TestAppendTodayCreatesNewWhenOldDate(t *testing.T) {
	root := copyTestdata(t)
	// Existing meeting note is from 20260104, not today — should create new
	out, err := runAppend(t, root, "today content", "--slug", "meeting", "--today")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT be the old file
	oldFile := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	if out == oldFile {
		t.Error("expected a new file, but got the old one")
	}

	// New file should contain today's date and the slug
	base := filepath.Base(out)
	today := time.Now().Format("20060102")
	if !strings.HasPrefix(base, today) {
		t.Errorf("expected filename to start with %s, got %s", today, base)
	}
	if !strings.Contains(base, "_meeting") {
		t.Errorf("expected slug in filename, got %s", base)
	}

	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "today content") {
		t.Error("appended content not found in created file")
	}
}

func TestAppendTodayAppendsToExistingTodayNote(t *testing.T) {
	root := copyTestdata(t)

	// First create a note for today with the slug
	firstOut, err := runAppend(t, root, "first entry", "--slug", "daily", "--today")
	if err != nil {
		t.Fatalf("unexpected error on first append: %v", err)
	}

	// Second append with --today should reuse the same file
	secondOut, err := runAppend(t, root, "second entry", "--slug", "daily", "--today")
	if err != nil {
		t.Fatalf("unexpected error on second append: %v", err)
	}

	if firstOut != secondOut {
		t.Errorf("expected same file, got %q and %q", firstOut, secondOut)
	}

	data, _ := os.ReadFile(firstOut)
	content := string(data)
	if !strings.Contains(content, "first entry") {
		t.Error("first entry not found")
	}
	if !strings.Contains(content, "second entry") {
		t.Error("second entry not found")
	}
}

func TestAppendTodayCreatesWhenNoMatch(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "brand new", "--slug", "nonexistent", "--today")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := filepath.Base(out)
	today := time.Now().Format("20060102")
	if !strings.HasPrefix(base, today) {
		t.Errorf("expected filename to start with %s, got %s", today, base)
	}

	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "brand new") {
		t.Error("appended content not found")
	}
}

func TestAppendTodayWithTitle(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "titled content", "--slug", "sessions", "--today",
		"--title", "Claude Sessions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	content := string(data)
	if !strings.Contains(content, "title: Claude Sessions") {
		t.Errorf("expected title in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "titled content") {
		t.Error("appended content not found")
	}
}

func TestAppendTodayWithPositionalArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "8823", "--today")
	if err == nil {
		t.Fatal("expected error when combining --today with positional arg, got nil")
	}
}

func TestAppendTodayWithCreateErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--slug", "meeting", "--today", "--create")
	if err == nil {
		t.Fatal("expected error when combining --today with --create, got nil")
	}
}

func TestAppendTodayWithoutFiltersErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--today")
	if err == nil {
		t.Fatal("expected error when using --today without filter flags, got nil")
	}
}

func TestAppendTitleWithoutCreateOrTodayErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--title", "Oops")
	if err == nil {
		t.Fatal("expected error when using --title without --create or --today, got nil")
	}
}

func TestAppendDescriptionWithoutCreateOrTodayErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--description", "Oops")
	if err == nil {
		t.Fatal("expected error when using --description without --create or --today, got nil")
	}
}
