package note

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTask(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
		done    bool
		isDaily bool
		isMoved bool
	}{
		{"pending", "- [ ] Buy milk", false, false, false, false},
		{"pending indented", "  - [ ] Buy milk", false, false, false, false},
		{"completed", "- [x] Done task", false, true, false, false},
		{"daily", "- [ ] Standup #daily", false, false, true, false},
		{"daily completed", "- [x] Standup #daily", false, true, true, false},
		{"moved", "- [ ] (moved) Buy milk", false, false, false, true},
		{"moved with other tag", "- [ ] (moved) (private) Do thing", false, false, false, true},
		{"no bullet", "[ ] Buy milk", true, false, false, false},
		{"unknown marker", "- [+] Done", true, false, false, false},
		{"not a task", "Just a regular line", true, false, false, false},
		{"empty", "", true, false, false, false},
		{"header", "# Todo", true, false, false, false},
		{"frontmatter", "---", true, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseTask(tt.line, 0)
			if tt.wantNil {
				assert.Nil(t, task)
				return
			}
			require.NotNil(t, task)
			assert.Equal(t, tt.done, task.Done)
			assert.Equal(t, tt.isDaily, task.IsDaily)
			assert.Equal(t, tt.isMoved, task.IsMoved)
		})
	}
}

func TestParseTaskText(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"- [ ] Buy milk", "Buy milk"},
		{"- [ ] Buy milk #daily", "Buy milk #daily"},
		{"  - [ ] (moved) Do thing", "(moved) Do thing"},
		{"- [x] Done", "Done"},
	}
	for _, tt := range tests {
		task := ParseTask(tt.line, 0)
		if task == nil {
			t.Fatalf("expected task for %q", tt.line)
		}
		if task.Text != tt.want {
			t.Errorf("Text for %q: got %q, want %q", tt.line, task.Text, tt.want)
		}
	}
}

func TestReassembled(t *testing.T) {
	task := ParseTask("  - [ ] Some task", 0)
	if task == nil {
		t.Fatal("expected task")
	}
	got := task.Reassembled("x")
	want := "  - [x] Some task"
	assert.Equal(t, want, got)
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

- [ ] Pending task one

- [ ] Pending task two

- [x] Completed task

- [ ] Standup #daily`, "\n")

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
	prev := strings.Split(`- [ ] Buy milk

- [ ] (private) Secret task

- [x] Completed task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 2 {
		t.Fatalf("carried %d tasks, want 2", len(result.CarriedTasks))
	}

	// Verify (moved) is inserted before existing tags
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Buy milk") && strings.Contains(line, "(moved)") {
			want := "- [ ] (moved) Buy milk"
			assert.Equal(t, want, line)
		}
		if strings.Contains(line, "Secret task") && strings.Contains(line, "(moved)") {
			want := "- [ ] (moved) (private) Secret task"
			assert.Equal(t, want, line)
		}
	}
}

func TestRolloverTasksSkipsMoved(t *testing.T) {
	prev := strings.Split(`- [ ] (moved) Already moved task

- [ ] Fresh task

- [x] Done task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1", len(result.CarriedTasks))
	}
	assert.Contains(t, result.CarriedTasks[0].Text, "Fresh task")

	// Already-moved task should not be re-tagged
	for _, line := range result.UpdatedLines {
		if strings.Contains(line, "Already moved") && line != "- [ ] (moved) Already moved task" {
			t.Errorf("moved task should be unchanged, got: %s", line)
		}
	}
}

func TestRolloverTasksDailyAlwaysCarried(t *testing.T) {
	prev := strings.Split(`- [x] Standup #daily

- [x] Completed other task`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1", len(result.CarriedTasks))
	}
	if !strings.Contains(result.CarriedTasks[0].Text, "Standup") {
		t.Error("expected daily task to be carried over")
	}
}

func TestRolloverTasksNoDuplicates(t *testing.T) {
	// A daily task that is also pending should appear only once
	prev := strings.Split(`- [ ] Standup #daily`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 1 {
		t.Fatalf("carried %d tasks, want 1 (no duplicates)", len(result.CarriedTasks))
	}
}

func TestRolloverTasksEmpty(t *testing.T) {
	prev := strings.Split(`---
slug: todo
---

- [x] Everything done`, "\n")

	result := RolloverTasks(prev)

	if len(result.CarriedTasks) != 0 {
		t.Errorf("carried %d tasks, want 0", len(result.CarriedTasks))
	}
}

func TestFormatTodoContent(t *testing.T) {
	task1 := ParseTask("- [ ] Task one", 0)
	task2 := ParseTask("- [ ] Task two", 1)
	if task1 == nil || task2 == nil {
		t.Fatal("expected tasks")
	}

	content := FormatTodoContent([]Task{*task1, *task2})

	if strings.HasPrefix(content, "---") {
		t.Errorf("unexpected frontmatter, got:\n%s", content)
	}
	if !strings.Contains(content, "- [ ] Task one") {
		t.Error("expected Task one with reset marker")
	}
	if !strings.Contains(content, "- [ ] Task two") {
		t.Error("expected Task two with reset marker")
	}
	// Tasks separated by blank lines
	assert.Contains(t, content, "- [ ] Task one\n\n- [ ] Task two")
}

func TestFormatTodoContentEmpty(t *testing.T) {
	content := FormatTodoContent(nil)
	if content != "" {
		t.Errorf("got:\n%q\nwant empty string", content)
	}
}
