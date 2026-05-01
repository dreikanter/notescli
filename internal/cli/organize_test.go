package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestOrganizeDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "organize-test-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	noteWithFrontmatter := `---
date: 2024-05-15
tags: [work, project]
---

# Test Note

This is a test note.
`
	err = os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte(noteWithFrontmatter), 0o644)
	require.NoError(t, err)

	noteNoFrontmatter := `# No Frontmatter

This note has no frontmatter.
`
	err = os.WriteFile(filepath.Join(tmpDir, "note2.md"), []byte(noteNoFrontmatter), 0o644)
	require.NoError(t, err)

	noteMissingDate := `---
tags: [personal]
---

# Missing Date

This note has tags but no date.
`
	err = os.WriteFile(filepath.Join(tmpDir, "note3.md"), []byte(noteMissingDate), 0o644)
	require.NoError(t, err)

	noteMissingTags := `---
date: 2024-06-20
---

# Missing Tags

This note has date but no tags.
`
	err = os.WriteFile(filepath.Join(tmpDir, "note4.md"), []byte(noteMissingTags), 0o644)
	require.NoError(t, err)

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0o755)
	require.NoError(t, err)

	noteInSubdir := `---
date: 2025-03-10
tags: [review]
---

# In Subdir

This note is in a subdirectory.
`
	err = os.WriteFile(filepath.Join(subDir, "note5.md"), []byte(noteInSubdir), 0o644)
	require.NoError(t, err)

	return tmpDir
}

func runOrganize(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	organizeCmd.ResetFlags()
	registerOrganizeFlags()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(append([]string{"organize", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(stdout.String()), err
}

func TestOrganizeDryRun(t *testing.T) {
	root := createTestOrganizeDir(t)

	out, err := runOrganize(t, root)
	require.NoError(t, err)

	assert.Contains(t, out, "Planned moves")
	assert.Contains(t, out, "note1.md -> 2024/work/note1.md")
	assert.Contains(t, out, "note2.md -> uncategorized/note2.md")
	assert.Contains(t, out, "note3.md -> uncategorized/note3.md")
	assert.Contains(t, out, "note4.md -> uncategorized/note4.md")
	assert.Contains(t, out, "note5.md -> 2025/review/note5.md")
	assert.Contains(t, out, "Dry-run complete")

	_, err = os.Stat(filepath.Join(root, "note1.md"))
	require.NoError(t, err, "file should not be moved in dry-run")

	_, err = os.Stat(filepath.Join(root, "2024"))
	assert.True(t, os.IsNotExist(err), "directory should not be created in dry-run")
}

func TestOrganizeApply(t *testing.T) {
	root := createTestOrganizeDir(t)

	out, err := runOrganize(t, root, "--apply")
	require.NoError(t, err)

	assert.Contains(t, out, "Executing moves")
	assert.Contains(t, out, "Moved:")
	assert.Contains(t, out, "Done.")

	_, err = os.Stat(filepath.Join(root, "2024", "work", "note1.md"))
	require.NoError(t, err, "note1 should be moved to 2024/work/")

	_, err = os.Stat(filepath.Join(root, "uncategorized", "note2.md"))
	require.NoError(t, err, "note2 should be moved to uncategorized/")

	_, err = os.Stat(filepath.Join(root, "uncategorized", "note3.md"))
	require.NoError(t, err, "note3 should be moved to uncategorized/")

	_, err = os.Stat(filepath.Join(root, "uncategorized", "note4.md"))
	require.NoError(t, err, "note4 should be moved to uncategorized/")

	_, err = os.Stat(filepath.Join(root, "2025", "review", "note5.md"))
	require.NoError(t, err, "note5 should be moved to 2025/review/")

	_, err = os.Stat(filepath.Join(root, "note1.md"))
	assert.True(t, os.IsNotExist(err), "original note1 should not exist")
}

func TestOrganizeNoFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "organize-empty-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	out, err := runOrganize(t, tmpDir)
	require.NoError(t, err)

	assert.Contains(t, out, "No files to organize")
}

func TestOrganizeConflictExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "organize-conflict-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	note1 := `---
date: 2024-05-15
tags: [work]
---

# Note 1
`
	err = os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte(note1), 0o644)
	require.NoError(t, err)

	existingDir := filepath.Join(tmpDir, "2024", "work")
	err = os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err)

	existingNote := `---
date: 2024-01-01
tags: [old]
---

# Existing Note
`
	err = os.WriteFile(filepath.Join(existingDir, "note1.md"), []byte(existingNote), 0o644)
	require.NoError(t, err)

	out, err := runOrganize(t, tmpDir)
	require.Error(t, err)
	assert.Contains(t, out, "Conflicts detected")
	assert.Contains(t, out, "2024/work/note1.md")
	assert.Contains(t, out, "destination already exists")
	assert.Contains(t, out, "note1.md")

	_, err = os.Stat(filepath.Join(tmpDir, "note1.md"))
	require.NoError(t, err, "original file should not be moved when conflict detected")

	_, err = os.Stat(filepath.Join(existingDir, "note1.md"))
	require.NoError(t, err, "existing file should not be modified")
}

func TestOrganizeConflictMultipleSources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "organize-conflict-multi-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	note1 := `---
date: 2024-05-15
tags: [work]
---

# Note 1
`
	err = os.WriteFile(filepath.Join(tmpDir, "note1.md"), []byte(note1), 0o644)
	require.NoError(t, err)

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0o755)
	require.NoError(t, err)

	note2 := `---
date: 2024-06-20
tags: [work]
---

# Note 2 (same filename)
`
	err = os.WriteFile(filepath.Join(subDir, "note1.md"), []byte(note2), 0o644)
	require.NoError(t, err)

	out, err := runOrganize(t, tmpDir)
	require.Error(t, err)
	assert.Contains(t, out, "Conflicts detected")
	assert.Contains(t, out, "2024/work/note1.md")
	assert.Contains(t, out, "multiple sources map to same destination")
	assert.Contains(t, out, "note1.md")
	assert.Contains(t, out, "subdir/note1.md")

	_, err = os.Stat(filepath.Join(tmpDir, "note1.md"))
	require.NoError(t, err, "original note1 should not be moved")

	_, err = os.Stat(filepath.Join(subDir, "note1.md"))
	require.NoError(t, err, "subdir/note1 should not be moved")
}

func TestComputeDestination(t *testing.T) {
	tests := []struct {
		name       string
		srcRel     string
		date       time.Time
		tags       []string
		wantDstRel string
		wantReason string
	}{
		{
			name:       "valid date and tags",
			srcRel:     "old/note.md",
			date:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			tags:       []string{"work", "project"},
			wantDstRel: "2024/work/note.md",
			wantReason: "year=2024, tag=work",
		},
		{
			name:       "missing date",
			srcRel:     "note.md",
			date:       time.Time{},
			tags:       []string{"work"},
			wantDstRel: "uncategorized/note.md",
			wantReason: "uncategorized (missing date or tags)",
		},
		{
			name:       "missing tags",
			srcRel:     "note.md",
			date:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			tags:       nil,
			wantDstRel: "uncategorized/note.md",
			wantReason: "uncategorized (missing date or tags)",
		},
		{
			name:       "empty tags",
			srcRel:     "note.md",
			date:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			tags:       []string{},
			wantDstRel: "uncategorized/note.md",
			wantReason: "uncategorized (missing date or tags)",
		},
		{
			name:       "only empty string tags",
			srcRel:     "note.md",
			date:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			tags:       []string{"", ""},
			wantDstRel: "uncategorized/note.md",
			wantReason: "uncategorized (no valid tags)",
		},
		{
			name:       "first tag empty but second valid",
			srcRel:     "note.md",
			date:       time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
			tags:       []string{"", "valid"},
			wantDstRel: "2024/valid/note.md",
			wantReason: "year=2024, tag=valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstRel, reason := computeDestination(tt.srcRel, tt.date, tt.tags)
			assert.Equal(t, tt.wantDstRel, dstRel)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}
