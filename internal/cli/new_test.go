package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runNew(t *testing.T, root string, stdin string, args ...string) (string, error) {
	t.Helper()

	newCmd.ResetFlags()
	registerNewFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(append([]string{"new", "--path", root}, args...))

	execErr := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), execErr
}

func TestNewDefault(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, root) {
		t.Errorf("expected path under root, got %q", out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("created file does not exist: %v", err)
	}
}

func TestNewWithSlug(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--slug", "myslug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(filepath.Base(out), "_myslug") {
		t.Errorf("expected slug in filename, got %q", filepath.Base(out))
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "slug: myslug") {
		t.Errorf("expected slug in frontmatter, got:\n%s", string(data))
	}
}

func TestNewWithType(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(filepath.Base(out), ".todo.") {
		t.Errorf("expected type in filename, got %q", filepath.Base(out))
	}
}

func TestNewWithTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--tag", "work", "--tag", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "tags:\n    - work\n    - daily\n") {
		t.Errorf("expected tags in frontmatter, got:\n%s", string(data))
	}
}

func TestNewWithTitleAndDescription(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--title", "My Note", "--description", "A description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	content := string(data)
	if !strings.Contains(content, "title: My Note") {
		t.Errorf("expected title in frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "description: A description") {
		t.Errorf("expected description in frontmatter, got:\n%s", content)
	}
}

func TestNewWithPublic(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--public")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(data), "public: true") {
		t.Errorf("expected public: true in frontmatter, got:\n%s", string(data))
	}
}

func TestNewAllDigitSlugErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runNew(t, root, "", "--slug", "999")
	if err == nil {
		t.Fatal("expected error for all-digit slug, got nil")
	}
}

func TestNewWithBody(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "hello world\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "hello world") {
		t.Errorf("expected body content in file, got:\n%s", string(data))
	}
}

func TestNewUpsertCreatesWhenNoMatch(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--slug", "report", "--upsert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(filepath.Base(out), "_report") {
		t.Errorf("expected slug in filename, got %s", filepath.Base(out))
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("created file does not exist: %v", err)
	}
}

func TestNewUpsertReturnsExisting(t *testing.T) {
	root := copyTestdata(t)

	// Create a note first
	first, err := runNew(t, root, "", "--slug", "report", "--upsert")
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	// Second call should return the same note
	second, err := runNew(t, root, "", "--slug", "report", "--upsert")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if first != second {
		t.Errorf("expected same path, got %q and %q", first, second)
	}
}

func TestNewUpsertByType(t *testing.T) {
	root := copyTestdata(t)

	first, err := runNew(t, root, "", "--type", "weekly", "--upsert")
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	second, err := runNew(t, root, "", "--type", "weekly", "--upsert")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if first != second {
		t.Errorf("expected same path, got %q and %q", first, second)
	}
}

func TestNewUpsertWithoutFilterErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runNew(t, root, "", "--upsert")
	if err == nil {
		t.Fatal("expected error when --upsert used without --type or --slug, got nil")
	}
}

func TestNewWithoutUpsertAlwaysCreates(t *testing.T) {
	root := copyTestdata(t)

	first, err := runNew(t, root, "", "--slug", "report")
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	second, err := runNew(t, root, "", "--slug", "report")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if first == second {
		t.Error("expected different paths without --upsert, got same path")
	}
}

func TestNewWithCustomType(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--type", "meeting", "--slug", "sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(filepath.Base(out), "sync.meeting.md") {
		t.Errorf("expected slug+type cache in filename, got %q", filepath.Base(out))
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "type: meeting") {
		t.Errorf("expected type: meeting in frontmatter, got:\n%s", string(data))
	}
}

func TestNewWithKnownTypeStillWrites(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(filepath.Base(out), ".todo.md") {
		t.Errorf("expected .todo.md suffix, got %q", filepath.Base(out))
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "type: todo") {
		t.Errorf("expected type: todo in frontmatter, got:\n%s", string(data))
	}
}
