package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runUpdate(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	updateCmd.ResetFlags()
	registerUpdateFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"update", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestUpdateTagsByID(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--tag", "new1", "--tag", "new2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	content := string(data)
	if !strings.Contains(content, "tags:\n    - new1\n    - new2\n") {
		t.Errorf("expected updated tags in frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "- work") {
		t.Error("old tags should be gone")
	}
}

func TestUpdateNoTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--no-tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "tags:") {
		t.Errorf("expected no tags line in frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateSlugRenamesFile: --slug updates frontmatter AND renames the file.
func TestUpdateSlugRenamesFile(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runUpdate(t, root, "8823", "--slug", "renamed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260106_8823_renamed.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file does not exist: %v", err)
	}
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Errorf("old file should have been removed, err=%v", err)
	}

	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "slug: renamed") {
		t.Errorf("expected slug in frontmatter, got:\n%s", string(data))
	}
}

func TestUpdateNoSlugRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8818", "--no-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260104_8818.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260104_8818_meeting.md")); !os.IsNotExist(err) {
		t.Errorf("old slugged file should be gone, err=%v", err)
	}
}

// TestUpdateTypeRenamesFile: --type rewrites frontmatter and cache suffix.
func TestUpdateTypeRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8823", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260106_8823_999.todo.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file missing: %v", err)
	}
	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "type: todo") {
		t.Errorf("expected type in frontmatter, got:\n%s", string(data))
	}
}

func TestUpdateNoTypeRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8814", "--no-type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260102_8814.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260102_8814.todo.md")); !os.IsNotExist(err) {
		t.Errorf("old typed file should be gone, err=%v", err)
	}
}

func TestUpdateDateMovesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8823", "--date", "20260301")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/03/20260301_8823_999.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823_999.md")); !os.IsNotExist(err) {
		t.Errorf("old file should be gone, err=%v", err)
	}
}

func TestUpdateDateInvalidFormat(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--date", "2026-03-01")
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

func TestUpdateTitle(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--title", "My Title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "title: My Title") {
		t.Errorf("expected title in frontmatter, got:\n%s", string(data))
	}
}

func TestUpdateClearTitle(t *testing.T) {
	root := copyTestdata(t)
	if _, err := runUpdate(t, root, "8823", "--title", "To Remove"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, err := runUpdate(t, root, "8823", "--title", "")
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "title:") {
		t.Errorf("title should be cleared, got:\n%s", string(data))
	}
}

func TestUpdateDescription(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--description", "Some desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "description: Some desc") {
		t.Errorf("expected description in frontmatter, got:\n%s", string(data))
	}
}

func TestUpdateNoFlagsErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823")
	if err == nil {
		t.Fatal("expected error when no update flags given")
	}
}

func TestUpdateNonExistentNoteErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "9999", "--tag", "x")
	if err == nil {
		t.Fatal("expected error for non-existent id")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestUpdateNonIntegerArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "meeting", "--tag", "x")
	if err == nil {
		t.Fatal("expected error for non-integer id")
	}
}

func TestUpdateSlugAndNoSlugConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8818", "--slug", "other", "--no-slug")
	if err == nil {
		t.Fatal("expected error when both --slug and --no-slug are set")
	}
}

func TestUpdateTagAndNoTagsConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--tag", "foo", "--no-tags")
	if err == nil {
		t.Fatal("expected error when both --tag and --no-tags are set")
	}
}

func TestUpdateTypeAndNoTypeConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8814", "--type", "backlog", "--no-type")
	if err == nil {
		t.Fatal("expected error when both --type and --no-type are set")
	}
}

func TestUpdateBodyPreserved(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--tag", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "# Plain note") {
		t.Errorf("body content missing, got:\n%s", string(data))
	}
}

func TestUpdatePublicSetsPublicField(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--public")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "public: true") {
		t.Errorf("expected public: true, got:\n%s", string(data))
	}
}

func TestUpdatePrivateRemovesPublicField(t *testing.T) {
	root := copyTestdata(t)
	if _, err := runUpdate(t, root, "8823", "--public"); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	out, err := runUpdate(t, root, "8823", "--private")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "public:") {
		t.Errorf("expected no public field, got:\n%s", string(data))
	}
}

func TestUpdatePublicAndPrivateConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--public", "--private")
	if err == nil {
		t.Fatal("expected error when both --public and --private are set")
	}
}

func TestUpdateAllDigitSlugErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--slug", "123")
	if err == nil {
		t.Fatal("expected error for all-digit slug")
	}
}
