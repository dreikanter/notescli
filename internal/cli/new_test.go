package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runNew(t *testing.T, root string, stdin string, args ...string) (string, error) {
	t.Helper()

	newCmd.ResetFlags()
	newCmd.Flags().String("slug", "", "descriptive slug appended to filename")
	newCmd.Flags().String("type", "", "note type (todo, backlog, weekly)")
	newCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().String("description", "", "description for frontmatter")
	newCmd.Flags().String("title", "", "title for frontmatter")
	newCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	newCmd.Flags().Bool("private", false, "mark note as private in frontmatter (default; overrides --public)")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"new", "--path", root}, args...))

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create stdin pipe: %v", err)
	}
	os.Stdin = r

	go func() {
		_, _ = io.WriteString(w, stdin)
		w.Close()
	}()

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

func TestNewInvalidTypeErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runNew(t, root, "", "--type", "invalid")
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
}

func TestNewWithTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--tag", "work", "--tag", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "tags: [work, daily]") {
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

func TestNewPublicPrivateBothPrivateWins(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--public", "--private")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if strings.Contains(string(data), "public:") {
		t.Errorf("expected public field absent when --private wins, got:\n%s", string(data))
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
