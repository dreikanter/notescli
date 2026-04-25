package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dreikanter/notesctl/note"
)

func runNewTodo(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"new-todo", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

// emptyNotesRoot creates a temp dir with only id.json, no notes.
func emptyNotesRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	data, _ := json.Marshal(map[string]int{"last_id": 9000})
	if err := os.WriteFile(filepath.Join(dir, "id.json"), data, 0o644); err != nil {
		t.Fatalf("cannot write id.json: %v", err)
	}
	return dir
}

func TestNewTodoCreatesFromPrevious(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	today := time.Now().Format(note.DateFormat)
	if !strings.Contains(filepath.Base(out), today) {
		t.Errorf("expected today's date %s in filename, got %q", today, filepath.Base(out))
	}
	if !strings.Contains(filepath.Base(out), ".todo.") {
		t.Errorf("expected .todo. in filename, got %q", filepath.Base(out))
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("created file does not exist: %v", err)
	}
}

func TestNewTodoReturnsExistingToday(t *testing.T) {
	root := copyTestdata(t)

	first, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}

	second, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}

	if first != second {
		t.Errorf("expected same path on second call, got %q then %q", first, second)
	}
}

func TestNewTodoNoPreviousCreatesEmpty(t *testing.T) {
	root := emptyNotesRoot(t)
	out, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("expected success when no previous todo, got error: %v", err)
	}
	if out == "" {
		t.Fatal("expected output path, got empty string")
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("created file does not exist: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read created file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "[ ]") {
		t.Errorf("expected no tasks in empty todo, got:\n%s", content)
	}
}

func TestNewTodoWritesTypeFrontmatter(t *testing.T) {
	root := copyTestdata(t)
	out, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "type: todo") {
		t.Errorf("expected type: todo in frontmatter, got:\n%s", string(data))
	}
}

func TestNewTodoRollsOverIncompleteTasks(t *testing.T) {
	root := emptyNotesRoot(t)

	// Seed a previous todo dated one day ago with a pending and a done task.
	prev := time.Now().AddDate(0, 0, -1)
	dir := filepath.Join(root, prev.Format("2006"), prev.Format("01"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	prevPath := filepath.Join(dir, prev.Format("20060102")+"_9001.todo.md")
	prevBody := "---\ntype: todo\n---\n\n- [ ] pending task\n\n- [x] finished task\n"
	if err := os.WriteFile(prevPath, []byte(prevBody), 0o644); err != nil {
		t.Fatal(err)
	}
	// Keep id.json in sync with the seeded ID so NextID doesn't collide.
	idData, _ := json.Marshal(map[string]int{"last_id": 9001})
	if err := os.WriteFile(filepath.Join(root, "id.json"), idData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newData, _ := os.ReadFile(out)
	newContent := string(newData)
	if !strings.Contains(newContent, "pending task") {
		t.Errorf("expected pending task carried over, got:\n%s", newContent)
	}
	if strings.Contains(newContent, "finished task") {
		t.Errorf("completed task should not be carried over, got:\n%s", newContent)
	}

	prevData, _ := os.ReadFile(prevPath)
	if !strings.Contains(string(prevData), "(moved)") {
		t.Errorf("previous todo should have (moved) markers on its pending tasks, got:\n%s", string(prevData))
	}
}
