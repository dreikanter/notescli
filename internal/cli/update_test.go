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
	updateCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable); replaces existing tags")
	updateCmd.Flags().Bool("no-tags", false, "remove all tags from frontmatter")
	updateCmd.Flags().String("title", "", "title for frontmatter (empty string clears it)")
	updateCmd.Flags().String("description", "", "description for frontmatter (empty string clears it)")
	updateCmd.Flags().String("slug", "", "update slug and rename file")
	updateCmd.Flags().Bool("no-slug", false, "remove slug from filename")
	updateCmd.Flags().String("type", "", "update note type and rename file (todo, backlog, weekly)")
	updateCmd.Flags().Bool("no-type", false, "remove type suffix from filename")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"update", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

// TestUpdateTagsByID replaces tags on a note resolved by numeric ID.
func TestUpdateTagsByID(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--tag", "new1", "--tag", "new2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}

	data, _ := os.ReadFile(want)
	content := string(data)
	if !strings.Contains(content, "tags: [new1, new2]") {
		t.Errorf("expected updated tags in frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "tags: [work]") {
		t.Error("old tags should be gone")
	}
}

// TestUpdateNoTags clears all tags.
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

// TestUpdateSlugRenamesFile updates slug and renames the file.
func TestUpdateSlugRenamesFile(t *testing.T) {
	root := copyTestdata(t)
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
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823.md")); err == nil {
		t.Error("old file should have been removed")
	}
}

// TestUpdateNoSlugRemovesSlugFromFilename drops the slug and renames the file.
func TestUpdateNoSlugRemovesSlugFromFilename(t *testing.T) {
	root := copyTestdata(t)
	// 8818 has slug "meeting"
	out, err := runUpdate(t, root, "8818", "--no-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260104_8818_meeting.md")); err == nil {
		t.Error("old file should have been removed")
	}
}

// TestUpdateTypeRenamesFile adds a type suffix and renames the file.
func TestUpdateTypeRenamesFile(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260106_8823.todo.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823.md")); err == nil {
		t.Error("old file should have been removed")
	}
}

// TestUpdateNoTypeRemovesTypeSuffix drops the type suffix and renames the file.
func TestUpdateNoTypeRemovesTypeSuffix(t *testing.T) {
	root := copyTestdata(t)
	// 8814 has type "todo"
	out, err := runUpdate(t, root, "8814", "--no-type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260102_8814.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260102_8814.todo.md")); err == nil {
		t.Error("old file should have been removed")
	}
}

// TestUpdateTitle updates the title field in frontmatter.
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

// TestUpdateClearTitle removes the title field when set to empty string.
func TestUpdateClearTitle(t *testing.T) {
	root := copyTestdata(t)
	// First set a title
	if _, err := runUpdate(t, root, "8823", "--title", "To Remove"); err != nil {
		t.Fatalf("unexpected error setting title: %v", err)
	}
	// Then clear it
	out, err := runUpdate(t, root, "8823", "--title", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	if strings.Contains(string(data), "title:") {
		t.Errorf("expected title removed from frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateDescription updates the description field.
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

// TestUpdateNoFlagsUnchanged verifies no change when no flags are provided.
func TestUpdateNoFlagsUnchanged(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823.md")
	before, _ := os.ReadFile(target)

	out, err := runUpdate(t, root, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != target {
		t.Errorf("got path %q, want %q", out, target)
	}

	after, _ := os.ReadFile(target)
	if string(before) != string(after) {
		t.Error("file should not have changed when no flags provided")
	}
}

// TestUpdateNonExistentNoteErrors returns an error for an unknown ref.
func TestUpdateNonExistentNoteErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "9999", "--tag", "x")
	if err == nil {
		t.Fatal("expected error for non-existent note, got nil")
	}
}

// TestUpdateInvalidTypeErrors returns an error for an unknown note type.
func TestUpdateInvalidTypeErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--type", "invalid")
	if err == nil {
		t.Fatal("expected error for unknown note type, got nil")
	}
}

// TestUpdateNoSlugTakesPrecedenceOverSlug verifies --no-slug wins when combined with --slug.
func TestUpdateNoSlugTakesPrecedenceOverSlug(t *testing.T) {
	root := copyTestdata(t)
	// 8818 already has slug "meeting"; pass both --slug and --no-slug
	out, err := runUpdate(t, root, "8818", "--slug", "other", "--no-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260104_8818.md")
	if out != want {
		t.Errorf("got path %q, want %q (--no-slug should win)", out, want)
	}
}

// TestUpdateNoTypeTakesPrecedenceOverType verifies --no-type wins when combined with --type.
func TestUpdateNoTypeTakesPrecedenceOverType(t *testing.T) {
	root := copyTestdata(t)
	// 8814 has type "todo"; pass both --type backlog and --no-type
	out, err := runUpdate(t, root, "8814", "--type", "backlog", "--no-type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(root, "2026/01/20260102_8814.md")
	if out != want {
		t.Errorf("got path %q, want %q (--no-type should win)", out, want)
	}
}

// TestUpdateBodyPreserved ensures note body is preserved after frontmatter update.
func TestUpdateBodyPreserved(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--tag", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(out)
	content := string(data)
	if !strings.Contains(content, "# Plain note") {
		t.Errorf("body content not preserved after update, got:\n%s", content)
	}
}
