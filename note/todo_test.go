package note

import (
	"strings"
	"testing"
)

func TestParseTask(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
		marker  string
		isDaily bool
	}{
		{"pending", "[ ] Buy milk", false, " ", false},
		{"pending with bullet", "- [ ] Buy milk", false, " ", false},
		{"pending indented", "  [ ] Buy milk", false, " ", false},
		{"pending indented bullet", "  - [ ] Buy milk", false, " ", false},
		{"in progress", "[>] Working on it", false, ">", false},
		{"completed", "[+] Done task", false, "+", false},
		{"daily", "[ ] Standup [daily]", false, " ", true},
		{"daily completed", "[+] Standup [daily]", false, "+", true},
		{"not a task", "Just a regular line", true, "", false},
		{"empty", "", true, "", false},
		{"header", "# Todo", true, "", false},
		{"frontmatter", "---", true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseTask(tt.line, 0)
			if tt.wantNil {
				if task != nil {
					t.Errorf("expected nil for %q, got %+v", tt.line, task)
				}
				return
			}
			if task == nil {
				t.Fatalf("expected non-nil for %q", tt.line)
			}
			if task.Marker != tt.marker {
				t.Errorf("marker: got %q, want %q", task.Marker, tt.marker)
			}
			if task.IsDaily != tt.isDaily {
				t.Errorf("isDaily: got %v, want %v", task.IsDaily, tt.isDaily)
			}
		})
	}
}

func TestReassembled(t *testing.T) {
	task := ParseTask("  - [ ] Some task", 0)
	if task == nil {
		t.Fatal("expected task")
	}
	got := task.Reassembled(">")
	want := "  - [>] Some task"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRolloverTasks(t *testing.T) {
	prev := strings.Split(`---
slug: todo
---

[ ] Pending task one

[ ] Pending task two

[>] In progress task

[+] Completed task

[ ] Standup [daily]`, "\n")

	result := RolloverTasks(prev, false)

	// Should carry over: pending task one, pending task two, standup daily
	if len(result.CarriedTasks) != 3 {
		t.Fatalf("carried %d tasks, want 3", len(result.CarriedTasks))
	}

	// Verify previous todo was updated: pending tasks → [>]
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Pending task one") && !strings.Contains(line, "[>]") {
			t.Errorf("pending task one should be [>] in updated lines, got: %s", line)
		}
		if strings.Contains(line, "Pending task two") && !strings.Contains(line, "[>]") {
			t.Errorf("pending task two should be [>] in updated lines, got: %s", line)
		}
	}

	// In-progress task should NOT be modified in updated lines
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "In progress task") && !strings.Contains(line, "[>]") {
			t.Errorf("in-progress task should remain [>], got: %s", line)
		}
	}
}

func TestRolloverTasksForce(t *testing.T) {
	prev := strings.Split(`[ ] Pending task

[>] In progress task

[+] Completed task`, "\n")

	result := RolloverTasks(prev, true)

	// With force: carry pending + in-progress
	if len(result.CarriedTasks) != 2 {
		t.Fatalf("carried %d tasks, want 2", len(result.CarriedTasks))
	}

	markers := make(map[string]bool)
	for _, task := range result.CarriedTasks {
		if strings.Contains(task.Suffix, "Pending") {
			markers["pending"] = true
		}
		if strings.Contains(task.Suffix, "In progress") {
			markers["in-progress"] = true
		}
	}
	if !markers["pending"] || !markers["in-progress"] {
		t.Error("expected both pending and in-progress tasks to be carried over with force")
	}
}

func TestRolloverTasksDailyAlwaysCarried(t *testing.T) {
	prev := strings.Split(`[+] Standup [daily]

[+] Completed other task`, "\n")

	result := RolloverTasks(prev, false)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1", len(result.CarriedTasks))
	}
	if !strings.Contains(result.CarriedTasks[0].Suffix, "Standup") {
		t.Error("expected daily task to be carried over")
	}
}

func TestRolloverTasksNoDuplicates(t *testing.T) {
	// A daily task that is also pending should appear only once
	prev := strings.Split(`[ ] Standup [daily]`, "\n")

	result := RolloverTasks(prev, false)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1 (no duplicates)", len(result.CarriedTasks))
	}
}

func TestRolloverTasksEmpty(t *testing.T) {
	prev := strings.Split(`---
slug: todo
---

[+] Everything done`, "\n")

	result := RolloverTasks(prev, false)

	if len(result.CarriedTasks) != 0 {
		t.Errorf("carried %d tasks, want 0", len(result.CarriedTasks))
	}
}

func TestFormatTodoContent(t *testing.T) {
	tasks := []Task{
		{Prefix: "[", Marker: " ", Suffix: "] Task one"},
		{Prefix: "[", Marker: ">", Suffix: "] Task two"},
	}

	content := FormatTodoContent(tasks)

	if strings.HasPrefix(content, "---") {
		t.Errorf("unexpected frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "[ ] Task one") {
		t.Error("expected Task one with reset marker")
	}
	if !strings.Contains(content, "[ ] Task two") {
		t.Error("expected Task two with reset marker")
	}
	// Tasks separated by blank lines
	if !strings.Contains(content, "[ ] Task one\n\n[ ] Task two") {
		t.Errorf("tasks should be separated by blank lines, got:\n%s", content)
	}
}

func TestFormatTodoContentEmpty(t *testing.T) {
	content := FormatTodoContent(nil)
	if content != "" {
		t.Errorf("got:\n%q\nwant empty string", content)
	}
}

func TestNoteFilename(t *testing.T) {
	tests := []struct {
		date string
		id   int
		slug string
		want string
	}{
		{"20260312", 9219, "", "20260312_9219.md"},
		{"20260312", 9219, "my-note", "20260312_9219_my-note.md"},
		{"20260312", 9219, "todo", "20260312_9219_todo.md"},
	}

	for _, tt := range tests {
		got := NoteFilename(tt.date, tt.id, tt.slug)
		if got != tt.want {
			t.Errorf("NoteFilename(%q, %d, %q) = %q, want %q", tt.date, tt.id, tt.slug, got, tt.want)
		}
	}
}

func TestNoteDirPath(t *testing.T) {
	got := NoteDirPath("/archive", "20260312")
	want := "/archive/2026/03"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindLatestTodo(t *testing.T) {
	notes := []Note{
		{Date: "20260312", Slug: "todo", RelPath: "2026/03/20260312_100_todo.md"},
		{Date: "20260311", Slug: "todo", RelPath: "2026/03/20260311_99_todo.md"},
		{Date: "20260310", Slug: "", RelPath: "2026/03/20260310_98.md"},
		{Date: "20260309", Slug: "todo", RelPath: "2026/03/20260309_97_todo.md"},
	}

	got := FindLatestTodo(notes, "20260312")
	if got == nil {
		t.Fatal("expected to find a todo")
	}
	if got.Date != "20260311" {
		t.Errorf("got date %s, want 20260311", got.Date)
	}
}

func TestFindLatestTodoNone(t *testing.T) {
	notes := []Note{
		{Date: "20260312", Slug: "todo"},
	}
	got := FindLatestTodo(notes, "20260312")
	if got != nil {
		t.Error("expected nil when no previous todo exists")
	}
}

func TestFindTodayTodo(t *testing.T) {
	notes := []Note{
		{Date: "20260312", Slug: "todo", RelPath: "2026/03/20260312_100_todo.md"},
		{Date: "20260311", Slug: "todo", RelPath: "2026/03/20260311_99_todo.md"},
	}

	got := FindTodayTodo("/archive", notes, "20260312")
	if got == nil {
		t.Fatal("expected to find today's todo")
	}
	if got.Date != "20260312" {
		t.Errorf("got date %s, want 20260312", got.Date)
	}

	got = FindTodayTodo("/archive", notes, "20260313")
	if got != nil {
		t.Error("expected nil for future date")
	}
}
