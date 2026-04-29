package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	data, _ := os.ReadFile(out)
	content := string(data)
	assert.Contains(t, content, "tags:\n    - new1\n    - new2\n")
	if strings.Contains(content, "- work") {
		t.Error("old tags should be gone")
	}
}

func TestUpdateNoTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--no-tags")
	require.NoError(t, err)

	data, _ := os.ReadFile(out)
	assert.NotContains(t, string(data), "tags:")
}

// TestUpdateSlugRenamesFile: --slug updates frontmatter AND renames the file.
func TestUpdateSlugRenamesFile(t *testing.T) {
	root := copyTestdata(t)
	origPath := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runUpdate(t, root, "8823", "--slug", "renamed")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260106_8823_renamed.md")
	assert.Equal(t, want, out)
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file does not exist: %v", err)
	}
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Errorf("old file should have been removed, err=%v", err)
	}

	data, _ := os.ReadFile(want)
	assert.Contains(t, string(data), "slug: renamed")
}

func TestUpdateNoSlugRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8818", "--no-slug")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260104_8818.md")
	assert.Equal(t, want, out)
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260104_8818_meeting.md")); !os.IsNotExist(err) {
		t.Errorf("old slugged file should be gone, err=%v", err)
	}
}

// TestUpdateTypeRenamesFile: --type rewrites frontmatter and cache suffix.
func TestUpdateTypeRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8823", "--type", "todo")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260106_8823_999.todo.md")
	assert.Equal(t, want, out)
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file missing: %v", err)
	}
	data, _ := os.ReadFile(want)
	assert.Contains(t, string(data), "type: todo")
}

func TestUpdateNoTypeRenamesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8814", "--no-type")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260102_8814.md")
	assert.Equal(t, want, out)
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260102_8814.todo.md")); !os.IsNotExist(err) {
		t.Errorf("old typed file should be gone, err=%v", err)
	}
}

func TestUpdateDateMovesFile(t *testing.T) {
	root := copyTestdata(t)

	out, err := runUpdate(t, root, "8823", "--date", "20260301")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/03/20260301_8823_999.md")
	assert.Equal(t, want, out)
	if _, err := os.Stat(want); err != nil {
		t.Errorf("new file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "2026/01/20260106_8823_999.md")); !os.IsNotExist(err) {
		t.Errorf("old file should be gone, err=%v", err)
	}
	data, err := os.ReadFile(want)
	require.NoError(t, err)
	assert.Contains(t, string(data), "date: 2026-03-01")
}

func TestUpdateDateInvalidFormat(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--date", "2026-03-01")
	require.Error(t, err)
}

func TestUpdateTitle(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--title", "My Title")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "title: My Title")
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
	assert.NotContains(t, string(data), "title:")
}

func TestUpdateDescription(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--description", "Some desc")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "description: Some desc")
}

func TestUpdateNoFlagsErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823")
	require.Error(t, err)
}

func TestUpdateNonExistentNoteErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "9999", "--tag", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateNonIntegerArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "meeting", "--tag", "x")
	require.Error(t, err)
}

func TestUpdateSlugAndNoSlugConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8818", "--slug", "other", "--no-slug")
	require.Error(t, err)
}

func TestUpdateTagAndNoTagsConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--tag", "foo", "--no-tags")
	require.Error(t, err)
}

func TestUpdateTypeAndNoTypeConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8814", "--type", "backlog", "--no-type")
	require.Error(t, err)
}

func TestUpdateBodyPreserved(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--tag", "updated")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "# Plain note")
}

func TestUpdatePublicSetsPublicField(t *testing.T) {
	root := copyTestdata(t)
	out, err := runUpdate(t, root, "8823", "--public")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "public: true")
}

func TestUpdatePrivateRemovesPublicField(t *testing.T) {
	root := copyTestdata(t)
	if _, err := runUpdate(t, root, "8823", "--public"); err != nil {
		t.Fatalf("pre-set: %v", err)
	}
	out, err := runUpdate(t, root, "8823", "--private")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.NotContains(t, string(data), "public:")
}

func TestUpdatePublicAndPrivateConflict(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--public", "--private")
	require.Error(t, err)
}

func TestUpdateAllDigitSlugErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823", "--slug", "123")
	require.Error(t, err)
}
