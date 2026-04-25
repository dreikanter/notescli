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

func runRm(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	rmCmd.ResetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"rm", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestRmByID(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260106_8823_999.md")

	out, err := runRm(t, root, "8823")
	require.NoError(t, err)

	assert.Equal(t, target, out)

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmNonExistentErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runRm(t, root, "9999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRmNonIntegerArgErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runRm(t, root, "meeting")
	require.Error(t, err)
}
