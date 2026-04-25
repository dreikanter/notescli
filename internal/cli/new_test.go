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
	require.NoError(t, err)
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
	require.NoError(t, err)
	assert.Contains(t, filepath.Base(out), "_myslug")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	assert.Contains(t, string(data), "slug: myslug")
}

func TestNewWithType(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--type", "todo")
	require.NoError(t, err)
	assert.Contains(t, filepath.Base(out), ".todo.")
}

func TestNewWithTags(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--tag", "work", "--tag", "daily")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "tags:\n    - work\n    - daily\n")
}

func TestNewWithTitleAndDescription(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--title", "My Note", "--description", "A description")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	content := string(data)
	assert.Contains(t, content, "title: My Note")
	assert.Contains(t, content, "description: A description")
}

func TestNewWithPublic(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--public")
	require.NoError(t, err)
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	assert.Contains(t, string(data), "public: true")
}

func TestNewAllDigitSlugErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runNew(t, root, "", "--slug", "999")
	require.Error(t, err)
}

func TestNewWithBody(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "hello world\n")
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "hello world")
}

func TestNewUpsertCreatesWhenNoMatch(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--slug", "report", "--upsert")
	require.NoError(t, err)
	assert.Contains(t, filepath.Base(out), "_report")
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

	assert.Equal(t, second, first)
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

	assert.Equal(t, second, first)
}

func TestNewUpsertWithoutFilterErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runNew(t, root, "", "--upsert")
	require.Error(t, err)
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
	require.NoError(t, err)
	assert.Contains(t, filepath.Base(out), "sync.meeting.md")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	assert.Contains(t, string(data), "type: meeting")
}

func TestNewWithKnownTypeStillWrites(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNew(t, root, "", "--type", "todo")
	require.NoError(t, err)
	if !strings.HasSuffix(filepath.Base(out), ".todo.md") {
		t.Errorf("expected .todo.md suffix, got %q", filepath.Base(out))
	}
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "type: todo")
}
