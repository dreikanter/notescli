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
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func TestTagsEmptyStore(t *testing.T) {
	root := t.TempDir()
	out, err := runTags(t, root)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestTagsMergedSourcesSorted(t *testing.T) {
	root := t.TempDir()
	writeTagsTestNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [work, planning]\n---\n\nHere is #coffee and #work again.\n")
	writeTagsTestNote(t, root, "2026/01/20260102_1002.md",
		"no fm, just #tea and #work.\n")

	out, err := runTags(t, root)
	require.NoError(t, err)
	assert.Equal(t, []string{"coffee", "planning", "tea", "work"}, strings.Split(out, "\n"))
}

func TestTagsLowercased(t *testing.T) {
	root := t.TempDir()
	writeTagsTestNote(t, root, "2026/01/20260101_1001.md",
		"---\ntags: [Work, PLANNING]\n---\n\nbody with #Coffee and #WORK.\n")

	out, err := runTags(t, root)
	require.NoError(t, err)
	assert.Equal(t, []string{"coffee", "planning", "work"}, strings.Split(out, "\n"))
}

func TestTagsIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	writeTagsTestNote(t, root, "2026/01/20260101_1001.md",
		"kept #real\n```\n#should-not-appear\n```\nalso #done\n")

	out, err := runTags(t, root)
	require.NoError(t, err)
	assert.NotContains(t, out, "should-not-appear")
	assert.Contains(t, out, "real")
	assert.Contains(t, out, "done")
}
