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
		isMoved bool
	}{
		{"pending", "[ ] Buy milk", false, " ", false, false},
		{"pending with bullet", "- [ ] Buy milk", false, " ", false, false},
		{"pending indented", "  [ ] Buy milk", false, " ", false, false},
		{"pending indented bullet", "  - [ ] Buy milk", false, " ", false, false},
		{"completed", "[+] Done task", false, "+", false, false},
		{"daily", "[ ] Standup #daily", false, " ", true, false},
		{"daily completed", "[+] Standup #daily", false, "+", true, false},
		{"moved", "- [ ] (moved) Buy milk", false, " ", false, true},
		{"moved with other tag", "- [ ] (moved) (private) Do thing", false, " ", false, true},
		{"not a task", "Just a regular line", true, "", false, false},
		{"empty", "", true, "", false, false},
		{"header", "# Todo", true, "", false, false},
		{"frontmatter", "---", true, "", false, false},
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
			if task.IsMoved != tt.isMoved {
				t.Errorf("isMoved: got %v, want %v", task.IsMoved, tt.isMoved)
			}
		})
	}
}

func TestReassembled(t *testing.T) {
	task := ParseTask("  - [ ] Some task", 0)
	if task == nil {
		t.Fatal("expected task")
	}
	got := task.Reassembled("+")
	want := "  - [+] Some task"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithTag(t *testing.T) {
	tests := []struct {
		name string
		line string
		tag  string
		want string
	}{
		{"simple", "- [ ] Do the thing", "moved", "- [ ] (moved) Do the thing"},
		{"with existing tag", "- [ ] (private) Do the thing", "moved", "- [ ] (moved) (private) Do the thing"},
		{"no bullet", "[ ] Do the thing", "moved", "[ ] (moved) Do the thing"},
		{"indented", "  - [ ] Do the thing", "moved", "  - [ ] (moved) Do the thing"},
		{"already tagged", "- [ ] (moved) Do the thing", "moved", "- [ ] (moved) Do the thing"},
		{"already tagged with other", "- [ ] (moved) (private) Do the thing", "moved", "- [ ] (moved) (private) Do the thing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseTask(tt.line, 0)
			if task == nil {
				t.Fatalf("expected task for %q", tt.line)
			}
			got := task.WithTag(tt.tag)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRolloverTasks(t *testing.T) {
	prev := strings.Split(`---
slug: todo
---

[ ] Pending task one

[ ] Pending task two

[+] Completed task

[ ] Standup #daily`, "\n")

	result := RolloverTasks(prev)

	// Should carry over: pending task one, pending task two, standup daily
	if len(result.CarriedTasks) != 3 {
		t.Fatalf("carried %d tasks, want 3", len(result.CarriedTasks))
	}

	// Verify previous todo was updated: pending tasks tagged (moved)
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Pending task one") && !strings.Contains(line, "(moved)") {
			t.Errorf("pending task one should have (moved) tag, got: %s", line)
		}
		if strings.Contains(line, "Pending task two") && !strings.Contains(line, "(moved)") {
			t.Errorf("pending task two should have (moved) tag, got: %s", line)
		}
	}

	// Verify moved tasks still have [ ] marker
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "(moved)") && !strings.Contains(line, "[ ]") {
			t.Errorf("moved task should keep [ ] marker, got: %s", line)
		}
	}
}

func TestRolloverTasksMovedFormat(t *testing.T) {
	prev := strings.Split(`[ ] Buy milk

[ ] (private) Secret task

[+] Completed task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 2 {
		t.Fatalf("carried %d tasks, want 2", len(result.CarriedTasks))
	}

	// Verify (moved) is inserted before existing tags
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Buy milk") && strings.Contains(line, "(moved)") {
			want := "[ ] (moved) Buy milk"
			if line != want {
				t.Errorf("got %q, want %q", line, want)
			}
		}
		if strings.Contains(line, "Secret task") && strings.Contains(line, "(moved)") {
			want := "[ ] (moved) (private) Secret task"
			if line != want {
				t.Errorf("got %q, want %q", line, want)
			}
		}
	}
}

func TestRolloverTasksSkipsMoved(t *testing.T) {
	prev := strings.Split(`[ ] (moved) Already moved task

[ ] Fresh task

[x] Done task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1", len(result.CarriedTasks))
	}
	if !strings.Contains(result.CarriedTasks[0].Suffix, "Fresh task") {
		t.Errorf("expected Fresh task, got: %s", result.CarriedTasks[0].Suffix)
	}

	// Already-moved task should not be re-tagged
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Already moved") && line != "[ ] (moved) Already moved task" {
			t.Errorf("moved task should be unchanged, got: %s", line)
		}
	}
}

func TestRolloverTasksDailyAlwaysCarried(t *testing.T) {
	prev := strings.Split(`[+] Standup #daily

[+] Completed other task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1", len(result.CarriedTasks))
	}
	if !strings.Contains(result.CarriedTasks[0].Suffix, "Standup") {
		t.Error("expected daily task to be carried over")
	}
}

func TestRolloverTasksNoDuplicates(t *testing.T) {
	// A daily task that is also pending should appear only once
	prev := strings.Split(`[ ] Standup #daily`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1 (no duplicates)", len(result.CarriedTasks))
	}
}

func TestRolloverTasksEmpty(t *testing.T) {
	prev := strings.Split(`---
slug: todo
---

[+] Everything done`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 0 {
		t.Errorf("carried %d tasks, want 0", len(result.CarriedTasks))
	}
}

func TestFormatTodoContent(t *testing.T) {
	tasks := []Task{
		{Prefix: "[", Marker: " ", Suffix: "] Task one"},
		{Prefix: "[", Marker: " ", Suffix: "] Task two"},
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

func TestFilename(t *testing.T) {
	tests := []struct {
		date     string
		id       int
		slug     string
		noteType string
		want     string
	}{
		{"20260312", 9219, "", "", "20260312_9219.md"},
		{"20260312", 9219, "my-note", "", "20260312_9219_my-note.md"},
		{"20260312", 9219, "", "todo", "20260312_9219.todo.md"},
		{"20260312", 9219, "standup", "todo", "20260312_9219_standup.todo.md"},
		{"20260312", 9219, "", "backlog", "20260312_9219.backlog.md"},
		// Unsafe types are omitted from the filename (frontmatter remains canonical).
		{"20260312", 9219, "", "foo.bar", "20260312_9219.md"},
		{"20260312", 9219, "", "a/b", "20260312_9219.md"},
		{"20260312", 9219, "", `a\b`, "20260312_9219.md"},
	}

	for _, tt := range tests {
		got := Filename(tt.date, tt.id, tt.slug, tt.noteType)
		if got != tt.want {
			t.Errorf("Filename(%q, %d, %q, %q) = %q, want %q", tt.date, tt.id, tt.slug, tt.noteType, got, tt.want)
		}
	}
}

func TestDirPath(t *testing.T) {
	got := DirPath("/archive", "20260312")
	want := "/archive/2026/03"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindLatestTodo(t *testing.T) {
	notes := []Note{
		{Date: "20260312", Type: "todo", RelPath: "2026/03/20260312_100.todo.md"},
		{Date: "20260311", Type: "todo", RelPath: "2026/03/20260311_99.todo.md"},
		{Date: "20260310", Type: "", RelPath: "2026/03/20260310_98.md"},
		{Date: "20260309", Type: "todo", RelPath: "2026/03/20260309_97.todo.md"},
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
		{Date: "20260312", Type: "todo"},
	}
	got := FindLatestTodo(notes, "20260312")
	if got != nil {
		t.Error("expected nil when no previous todo exists")
	}
}

func TestFindTodayTodo(t *testing.T) {
	notes := []Note{
		{Date: "20260312", Type: "todo", RelPath: "2026/03/20260312_100.todo.md"},
		{Date: "20260311", Type: "todo", RelPath: "2026/03/20260311_99.todo.md"},
	}

	got := FindTodayTodo(notes, "20260312")
	if got == nil {
		t.Fatal("expected to find today's todo")
	}
	if got.Date != "20260312" {
		t.Errorf("got date %s, want 20260312", got.Date)
	}

	got = FindTodayTodo(notes, "20260313")
	if got != nil {
		t.Error("expected nil for future date")
	}
}
