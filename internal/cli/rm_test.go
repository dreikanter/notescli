package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runRm(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	rmCmd.ResetFlags()
	rmCmd.Flags().Bool("today", false, "only match notes created today")

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("got %q, want %q", out, target)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmBySlug(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260104_8818_meeting.md")

	out, err := runRm(t, root, "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("got %q, want %q", out, target)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmNonExistentErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runRm(t, root, "9999")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}

func TestRmTodayFlag(t *testing.T) {
	root := t.TempDir()
	today := time.Now().Format("20060102")
	dir := filepath.Join(root, today[:4], today[4:6])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fname := today + "_0001_daily.md"
	target := filepath.Join(dir, fname)
	if err := os.WriteFile(target, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runRm(t, root, "--today", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != target {
		t.Errorf("got %q, want %q", out, target)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmTodayExcludesOldNotes(t *testing.T) {
	root := copyTestdata(t)
	target := filepath.Join(root, "2026/01/20260104_8818_meeting.md")

	_, err := runRm(t, root, "--today", "meeting")
	if err == nil {
		t.Fatal("expected error when --today excludes matching note")
	}

	if _, err := os.Stat(target); err != nil {
		t.Error("file should NOT have been deleted")
	}
}
