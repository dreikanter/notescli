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
	if !strings.Contains(content, "tags:\n    - new1\n    - new2\n") {
		t.Errorf("expected updated tags in frontmatter, got:\n%s", content)
	}
	if strings.Contains(content, "- work") {
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

// TestUpdateSlugChangesFrontmatterOnly: --slug rewrites frontmatter but leaves filename.
func TestUpdateSlugChangesFrontmatterOnly(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runUpdate(t, root, "8823", "--slug", "renamed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != origPath {
		t.Errorf("got path %q, want %q (should not rename)", out, origPath)
	}
	if _, err := os.Stat(origPath); err != nil {
		t.Errorf("original file missing: %v", err)
	}
	data, _ := os.ReadFile(origPath)
	if !strings.Contains(string(data), "slug: renamed") {
		t.Errorf("expected updated slug in frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateSlugWithSyncFilenameRenames: --slug + --sync-filename renames the file.
func TestUpdateSlugWithSyncFilenameRenames(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--slug", "renamed", "--sync-filename")
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

// TestUpdateNoSlugClearsSlugFromFrontmatter: --no-slug removes slug from frontmatter only.
func TestUpdateNoSlugClearsSlugFromFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260104_8818_meeting.md")

	out, err := runUpdate(t, root, "8818", "--no-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != origPath {
		t.Errorf("got path %q, want %q (should not rename)", out, origPath)
	}
	data, _ := os.ReadFile(origPath)
	if strings.Contains(string(data), "slug:") {
		t.Errorf("expected slug removed, got:\n%s", string(data))
	}
}

// TestUpdateNoSlugWithSyncFilenameRenames: --no-slug + --sync-filename renames file.
func TestUpdateNoSlugWithSyncFilenameRenames(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8818", "--no-slug", "--sync-filename")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260104_8818.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260104_8818_meeting.md")); err == nil {
		t.Error("old file should have been removed")
	}
}

// TestUpdateTypeChangesFrontmatterOnly: --type rewrites frontmatter but leaves filename.
func TestUpdateTypeChangesFrontmatterOnly(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runUpdate(t, root, "8823", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != origPath {
		t.Errorf("got path %q, want %q (should not rename)", out, origPath)
	}
	if _, err := os.Stat(origPath); err != nil {
		t.Errorf("original file missing: %v", err)
	}
	data, _ := os.ReadFile(origPath)
	if !strings.Contains(string(data), "type: todo") {
		t.Errorf("expected updated type in frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateTypeWithSyncFilenameRenames: --type + --sync-filename renames the file.
func TestUpdateTypeWithSyncFilenameRenames(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--type", "todo", "--sync-filename")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "2026/01/20260106_8823_999.todo.md")
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

// TestUpdateNoTypeClearsTypeFromFrontmatter: --no-type clears frontmatter type only.
func TestUpdateNoTypeClearsTypeFromFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	// 8814 has type "todo" reported by filename (no fm type).
	origPath := filepath.Join(root, "2026/01/20260102_8814.todo.md")

	out, err := runUpdate(t, root, "8814", "--no-type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != origPath {
		t.Errorf("got path %q, want %q (should not rename)", out, origPath)
	}
	if _, err := os.Stat(origPath); err != nil {
		t.Errorf("original file missing: %v", err)
	}
	data, _ := os.ReadFile(origPath)
	if strings.Contains(string(data), "type:") {
		t.Errorf("expected type removed, got:\n%s", string(data))
	}
}

// TestUpdateNoTypeWithSyncFilenameRenames: --no-type + --sync-filename renames file.
func TestUpdateNoTypeWithSyncFilenameRenames(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8814", "--no-type", "--sync-filename")
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

// TestUpdatePreservesExtraFields ensures unknown frontmatter keys survive an update.
func TestUpdatePreservesExtraFields(t *testing.T) {
	root := copyTestdata(t)
	// Pick an existing fixture note, overwrite its content to include custom keys.
	notePath := filepath.Join(root, "2026/01/20260106_8823_999.md")
	seed := "---\ntitle: Original\nfeatured: true\ncustom_rating: 5\n---\n\nbody\n"
	if err := os.WriteFile(notePath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runUpdate(t, root, "8823", "--title", "New Title"); err != nil {
		t.Fatalf("update: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	if !strings.Contains(string(data), "title: New Title") {
		t.Errorf("expected new title, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "featured: true") {
		t.Errorf("featured dropped, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "custom_rating: 5") {
		t.Errorf("custom_rating dropped, got:\n%s", string(data))
	}
}

// TestUpdateDoesNotPromoteFilenameSlugToFrontmatter: an ordinary --title update
// on a note whose filename carries a slug but whose frontmatter doesn't must
// leave the frontmatter's absent slug absent. Frontmatter is canonical.
func TestUpdateDoesNotPromoteFilenameSlugToFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	notePath := filepath.Join(root, "2026/01/20260106_8823_999.md")
	// Seed with no slug/type in frontmatter; filename still has slug "999".
	seed := "---\ntitle: Original\n---\n\nbody\n"
	if err := os.WriteFile(notePath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runUpdate(t, root, "8823", "--title", "New"); err != nil {
		t.Fatalf("update: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	if strings.Contains(string(data), "slug:") {
		t.Errorf("filename slug must not be promoted into frontmatter, got:\n%s", string(data))
	}
}

// TestUpdateSyncFilenameOnly reconciles filename without any content flags.
func TestUpdateSyncFilenameOnly(t *testing.T) {
	root := copyTestdata(t)
	// Seed a note whose frontmatter slug disagrees with its filename.
	dir := filepath.Join(root, "2026", "01")
	origPath := filepath.Join(dir, "20260106_8823_999.md")
	seed := "---\ntitle: T\nslug: my-slug\ntype: meeting\n---\n\nbody\n"
	if err := os.WriteFile(origPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runUpdate(t, root, "8823", "--sync-filename")
	if err != nil {
		t.Fatalf("--sync-filename: %v", err)
	}
	want := filepath.Join(dir, "20260106_8823_my-slug.meeting.md")
	if out != want {
		t.Errorf("got path %q, want %q", out, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file missing: %v", err)
	}
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Errorf("old file should be gone: err=%v", err)
	}
}

// TestUpdateSyncFilenameNoOp: --sync-filename on an already-in-sync note is a no-op.
func TestUpdateSyncFilenameNoOp(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")
	// The fixture has empty fm slug; the filename-reported slug is "999" and
	// fills in as fallback. Running sync should produce the same filename.
	out, err := runUpdate(t, root, "8823", "--sync-filename")
	if err != nil {
		t.Fatalf("--sync-filename: %v", err)
	}
	if out != origPath {
		t.Errorf("expected no rename, got path %q", out)
	}
	if _, err := os.Stat(origPath); err != nil {
		t.Errorf("file moved or lost: %v", err)
	}
}
