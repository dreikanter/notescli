package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}
	return abs
}

func runResolve(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	resolveCmd.ResetFlags()
	registerResolveFlags()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(append([]string{"resolve", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(stdout.String()), err
}

func TestResolveNewestNoArgs(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root)
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	assert.Equal(t, want, out)
}

func TestResolveByID(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "--id", "8823")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260106_8823_999.md")
	assert.Equal(t, want, out)
}

func TestResolveByIDNotFound(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "--id", "99999")
	require.Error(t, err)
}

func TestResolveByIDNonInteger(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "--id", "notnumber")
	require.Error(t, err)
}

func TestResolveBySlug(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "--slug", "meeting")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	assert.Equal(t, want, out)
}

func TestResolveByType(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "--type", "todo")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260102_8814.todo.md")
	assert.Equal(t, want, out)
}

func TestResolveByTag(t *testing.T) {
	root := testdataPath(t)
	out, err := runResolve(t, root, "--tag", "meeting")
	require.NoError(t, err)
	want := filepath.Join(root, "2026/01/20260104_8818_meeting.md")
	assert.Equal(t, want, out)
}

func TestResolveNoMatchErrors(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "--slug", "nonexistent-slug-xyz")
	require.Error(t, err)
}

func TestResolveMultipleFlagsError(t *testing.T) {
	root := testdataPath(t)
	_, err := runResolve(t, root, "--id", "1", "--slug", "x")
	require.Error(t, err)
}
