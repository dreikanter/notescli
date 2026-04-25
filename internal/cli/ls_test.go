package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runLs(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)
	lsCmd.ResetFlags()
	registerLsFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"ls", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestLsNoArgs(t *testing.T) {
	out, err := runLs(t)
	require.NoError(t, err)

	lines := outputLines(out)
	assert.Len(t, lines, 4)
	for _, line := range lines {
		assert.True(t, allDigits(line), "expected integer ID per line, got %q", line)
	}
}

func TestLsFilters(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCount int
		wantIDs   []string
	}{
		{name: "tag", args: []string{"--tag", "work"}, wantCount: 3},
		{name: "tag mixed case", args: []string{"--tag", "WORK"}, wantCount: 3},
		{name: "tag no match", args: []string{"--tag", "nonexistent"}},
		{name: "tag and type", args: []string{"--tag", "work", "--type", "todo"}, wantIDs: []string{"8814"}},
		{name: "tag and limit", args: []string{"--tag", "work", "--limit", "1"}, wantCount: 1},
		{name: "multiple tags are AND", args: []string{"--tag", "work", "--tag", "planning"}, wantIDs: []string{"8814"}},
		{name: "comma-separated tags are AND", args: []string{"--tag", "work,meeting"}, wantIDs: []string{"8818"}},
		{name: "tag and type no overlap", args: []string{"--tag", "meeting", "--type", "todo"}},
		{name: "today excludes past testdata", args: []string{"--today"}},
		{name: "slug", args: []string{"--slug", "meeting"}, wantIDs: []string{"8818"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runLs(t, tt.args...)
			require.NoError(t, err)

			lines := outputLines(out)
			if tt.wantIDs != nil {
				assert.Equal(t, tt.wantIDs, lines)
				return
			}
			assert.Len(t, lines, tt.wantCount)
		})
	}
}

func outputLines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
