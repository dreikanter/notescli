package cli

import (
	"bytes"
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Plain note") {
		t.Errorf("expected note content, got: %s", out)
	}
}

func TestReadMissingID(t *testing.T) {
	_, err := runRead(t, "999999")
	if err == nil {
		t.Fatal("expected error for non-existent id, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}
}

func TestReadNonIntegerArg(t *testing.T) {
	_, err := runRead(t, "not-an-id")
	if err == nil {
		t.Fatal("expected error for non-integer id, got nil")
	}
}

func TestReadNoArgsErrors(t *testing.T) {
	_, err := runRead(t)
	if err == nil {
		t.Fatal("expected error when no positional arg, got nil")
	}
}

func TestReadNoFrontmatter(t *testing.T) {
	out, err := runRead(t, "8818", "--no-frontmatter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Frontmatter should be stripped; "tags:" should not appear
	if strings.Contains(out, "tags:") {
		t.Errorf("expected frontmatter stripped, got: %s", out)
	}
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected note body, got: %s", out)
	}
}
