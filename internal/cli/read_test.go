package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
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

func TestReadByTagFilter(t *testing.T) {
	out, err := runRead(t, "--tag", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 20260104_8818_meeting.md contains "Standup notes"
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected meeting note content, got: %s", out)
	}
}

func TestReadByTypeFilter(t *testing.T) {
	out, err := runRead(t, "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 20260102_8814.todo.md contains "Todo"
	if !strings.Contains(out, "Todo") {
		t.Errorf("expected todo note content, got: %s", out)
	}
}

func TestReadBySlugFilter(t *testing.T) {
	out, err := runRead(t, "--slug", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected meeting note content, got: %s", out)
	}
}

func TestReadByTodayFilter(t *testing.T) {
	// No notes in testdata match today's date, so this should error.
	today := time.Now().Format("20060102")
	_, err := runRead(t, "--today")
	if err == nil {
		t.Fatalf("expected error for --today with no matching notes (today=%s), got nil", today)
	}
}

func TestReadPositionalArgWithFilterErrors(t *testing.T) {
	_, err := runRead(t, "8823", "--type", "todo")
	if err == nil {
		t.Fatal("expected error when combining positional arg with filter flags, got nil")
	}
}

func TestReadNoTargetErrors(t *testing.T) {
	_, err := runRead(t)
	if err == nil {
		t.Fatal("expected error when no positional arg and no filter flags, got nil")
	}
}

func TestReadNoMatchErrors(t *testing.T) {
	_, err := runRead(t, "--slug", "nonexistent-slug-xyz")
	if err == nil {
		t.Fatal("expected error when filters match nothing, got nil")
	}
}

func TestReadNoFrontmatterWithFilter(t *testing.T) {
	out, err := runRead(t, "--tag", "meeting", "--no-frontmatter")
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

func TestReadPositionalArgWithTodayErrors(t *testing.T) {
	_, err := runRead(t, "8823", "--today")
	if err == nil {
		t.Fatal("expected error when combining positional arg with --today, got nil")
	}
}
