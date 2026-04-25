package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runRead(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	readCmd.ResetFlags()
	registerReadFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"read", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestReadByID(t *testing.T) {
	out, err := runRead(t, "8823")
	require.NoError(t, err)
	assert.Contains(t, out, "Plain note")
}

func TestReadMissingID(t *testing.T) {
	_, err := runRead(t, "999999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadNonIntegerArg(t *testing.T) {
	_, err := runRead(t, "not-an-id")
	require.Error(t, err)
}

func TestReadNoArgsErrors(t *testing.T) {
	_, err := runRead(t)
	require.Error(t, err)
}

func TestReadNoFrontmatter(t *testing.T) {
	out, err := runRead(t, "8818", "--no-frontmatter")
	require.NoError(t, err)
	// Frontmatter should be stripped; "tags:" should not appear
	assert.NotContains(t, out, "tags:")
	assert.Contains(t, out, "Standup notes")
}
