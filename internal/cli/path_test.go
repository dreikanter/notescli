package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
	root := testdataPath(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"path", "--path", root})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
}
