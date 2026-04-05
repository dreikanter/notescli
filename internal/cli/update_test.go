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
	updateCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	updateCmd.Flags().Bool("private", false, "mark note as private in frontmatter")
	updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
	updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
	updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
	updateCmd.MarkFlagsMutuallyExclusive("public", "private")

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

	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
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
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823_999.md")); err == nil {
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

	want := filepath.Join(root, "2026/01/20260106_8823_999.todo.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823_999.md")); err == nil {
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

// TestUpdateNoFlagsErrors verifies that update with no flags returns an error.
func TestUpdateNoFlagsErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823")
	if err == nil {
		t.Fatal("expected error when no update flags provided, got nil")
	}
	if !strings.Contains(err.Error(), "at least one update flag is required") {
		t.Errorf("unexpected error message: %v", err)
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

// TestUpdateSlugAndNoSlugConflict verifies --slug and --no-slug cannot be used together.
func TestUpdateSlugAndNoSlugConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8818", "--slug", "other", "--no-slug")
	if err == nil {
		t.Fatal("expected error for --slug and --no-slug together, got nil")
	}
	if !strings.Contains(err.Error(), "slug") || !strings.Contains(err.Error(), "no-slug") {
		t.Errorf("expected error mentioning both flags, got: %v", err)
	}
}

// TestUpdateTagAndNoTagsConflict verifies --tag and --no-tags cannot be used together.
func TestUpdateTagAndNoTagsConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--tag", "foo", "--no-tags")
	if err == nil {
		t.Fatal("expected error for --tag and --no-tags together, got nil")
	}
	if !strings.Contains(err.Error(), "tag") || !strings.Contains(err.Error(), "no-tags") {
		t.Errorf("expected error mentioning both flags, got: %v", err)
	}
}

// TestUpdateTypeAndNoTypeConflict verifies --type and --no-type cannot be used together.
func TestUpdateTypeAndNoTypeConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8814", "--type", "backlog", "--no-type")
	if err == nil {
		t.Fatal("expected error for --type and --no-type together, got nil")
	}
	if !strings.Contains(err.Error(), "type") || !strings.Contains(err.Error(), "no-type") {
		t.Errorf("expected error mentioning both flags, got: %v", err)
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

// TestUpdateSlugSyncsToFrontmatter verifies that --slug also sets the slug frontmatter field.
func TestUpdateSlugSyncsToFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--slug", "new-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "slug: new-slug") {
		t.Errorf("expected slug in frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateNoSlugRemovesSlugFromFrontmatter verifies that --no-slug removes the slug frontmatter field.
func TestUpdateNoSlugRemovesSlugFromFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	// First add a slug to frontmatter on note 8823
	_, err := runUpdate(t, root, "8823", "--slug", "to-remove")
	if err != nil {
		t.Fatalf("unexpected error setting slug: %v", err)
	}
	// Then remove it
	out, err := runUpdate(t, root, "8823", "--no-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if strings.Contains(string(data), "slug:") {
		t.Errorf("expected slug removed from frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateNoFlagDoesNotTouchFrontmatterSlug verifies that unrelated updates don't clobber an existing frontmatter slug.
func TestUpdateNoFlagDoesNotTouchFrontmatterSlug(t *testing.T) {
	root := copyTestdata(t)
	// Give note 8823 a slug in frontmatter
	_, err := runUpdate(t, root, "8823", "--slug", "keep-me")
	if err != nil {
		t.Fatalf("unexpected error setting slug: %v", err)
	}
	// Update only the title — slug flags are NOT passed
	out, err := runUpdate(t, root, "8823", "--title", "New Title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "slug: keep-me") {
		t.Errorf("expected slug frontmatter to be preserved, got:\n%s", string(data))
	}
}

// TestUpdatePublicSetsPublicField verifies that --public writes public: true to frontmatter.
func TestUpdatePublicSetsPublicField(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--public")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "public: true") {
		t.Errorf("expected public: true in frontmatter, got:\n%s", string(data))
	}
}

// TestUpdatePrivateRemovesPublicField verifies that --private removes public: true from frontmatter.
func TestUpdatePrivateRemovesPublicField(t *testing.T) {
	root := copyTestdata(t)
	// First mark as public
	_, err := runUpdate(t, root, "8823", "--public")
	if err != nil {
		t.Fatalf("unexpected error setting public: %v", err)
	}
	// Then mark as private
	out, err := runUpdate(t, root, "8823", "--private")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if strings.Contains(string(data), "public:") {
		t.Errorf("expected public field removed from frontmatter, got:\n%s", string(data))
	}
}

// TestUpdatePublicAndPrivateConflict verifies --public and --private cannot be used together.
func TestUpdatePublicAndPrivateConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--public", "--private")
	if err == nil {
		t.Fatal("expected error for --public and --private together, got nil")
	}
	if !strings.Contains(err.Error(), "public") || !strings.Contains(err.Error(), "private") {
		t.Errorf("expected error mentioning both flags, got: %v", err)
	}
}

func TestUpdateAllDigitSlugErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--slug", "123")
	if err == nil {
		t.Fatal("expected error for all-digit slug, got nil")
	}
}

// TestUpdateNoPublicFlagPreservesPublicField verifies unrelated updates don't touch the public field.
func TestUpdateNoPublicFlagPreservesPublicField(t *testing.T) {
	root := copyTestdata(t)
	// Mark as public
	_, err := runUpdate(t, root, "8823", "--public")
	if err != nil {
		t.Fatalf("unexpected error setting public: %v", err)
	}
	// Update only the title — no public/private flag
	out, err := runUpdate(t, root, "8823", "--title", "New Title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "public: true") {
		t.Errorf("expected public: true preserved after unrelated update, got:\n%s", string(data))
	}
}
