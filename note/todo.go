package note

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// taskRe matches lines like "  - [ ] some task" or "[ ] some task" or "  [>] task".
var taskRe = regexp.MustCompile(`^(\s*(?:- )?\[)(.?)(\].*)$`)

// Task represents a parsed task line from a todo note.
type Task struct {
	Line       string // original full line
	Prefix     string // everything before the marker character: e.g. "  - ["
	Marker     string // single char marker: " ", ">", "+", etc.
	Suffix     string // everything after marker: e.g. "] some task"
	IsDaily    bool   // whether line contains [daily]
	LineNumber int    // 0-based index in the source file lines
}

// ParseTask attempts to parse a line as a task. Returns nil if not a task line.
func ParseTask(line string, lineNumber int) *Task {
	m := taskRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	return &Task{
		Line:       line,
		Prefix:     m[1],
		Marker:     m[2],
		Suffix:     m[3],
		IsDaily:    strings.Contains(line, "[daily]"),
		LineNumber: lineNumber,
	}
}

// Reassembled returns the task line with a new marker.
func (t *Task) Reassembled(marker string) string {
	return t.Prefix + marker + t.Suffix
}

// ExtractTasks parses all task lines from a todo file's content lines.
func ExtractTasks(lines []string) []Task {
	var tasks []Task
	for i, line := range lines {
		if t := ParseTask(line, i); t != nil {
			tasks = append(tasks, *t)
		}
	}
	return tasks
}

// RolloverResult holds the output of a todo rollover operation.
type RolloverResult struct {
	CarriedTasks []Task   // tasks to include in the new todo
	UpdatedLines []string // modified lines of the previous todo (with [ ] → [>])
}

// RolloverTasks determines which tasks to carry over and produces the modified previous todo.
// If force is true, also carry over in-progress [>] tasks.
func RolloverTasks(prevLines []string, force bool) RolloverResult {
	tasks := ExtractTasks(prevLines)
	updated := make([]string, len(prevLines))
	copy(updated, prevLines)

	seen := make(map[string]bool) // normalized task text to prevent duplicates
	var carried []Task

	addTask := func(t Task) {
		// Normalize: strip leading whitespace, bullet, and marker for dedup
		key := strings.TrimSpace(t.Suffix)
		if seen[key] {
			return
		}
		seen[key] = true
		carried = append(carried, t)
	}

	for _, t := range tasks {
		switch {
		case t.IsDaily:
			// Daily tasks are always carried over regardless of marker
			addTask(t)
			// If the task was pending, mark as forwarded in previous
			if t.Marker == " " {
				updated[t.LineNumber] = t.Reassembled(">")
			}
		case t.Marker == " ":
			// Pending tasks: carry over and mark as forwarded
			addTask(t)
			updated[t.LineNumber] = t.Reassembled(">")
		case t.Marker == ">" && force:
			// In-progress tasks: only carry over with --force
			addTask(t)
		}
	}

	return RolloverResult{
		CarriedTasks: carried,
		UpdatedLines: updated,
	}
}

// FormatTodoContent formats carried tasks into the new todo file content.
func FormatTodoContent(tasks []Task) string {
	fm := BuildFrontmatter("todo", nil, "", "")

	if len(tasks) == 0 {
		return fm
	}

	var lines []string
	for _, t := range tasks {
		// Reset marker to pending [ ]
		lines = append(lines, t.Reassembled(" "))
	}

	return fm + strings.Join(lines, "\n\n") + "\n"
}

// FindLatestTodo finds the most recent todo note strictly before the given date.
func FindLatestTodo(notes []Note, beforeDate string) *Note {
	// notes are sorted newest-first
	for i := range notes {
		if notes[i].Slug == "todo" && notes[i].Date < beforeDate {
			return &notes[i]
		}
	}
	return nil
}

// FindTodayTodo finds a todo note matching today's date.
func FindTodayTodo(root string, notes []Note, today string) *Note {
	for i := range notes {
		if notes[i].Slug == "todo" && notes[i].Date == today {
			return &notes[i]
		}
	}
	return nil
}

// NoteFilename generates a note filename from date, id, and optional slug.
func NoteFilename(date string, id int, slug string) string {
	if slug != "" {
		return fmt.Sprintf("%s_%d_%s.md", date, id, slug)
	}
	return fmt.Sprintf("%s_%d.md", date, id)
}

// NoteDirPath returns the YYYY/MM directory path for a given date string (YYYYMMDD).
func NoteDirPath(root, date string) string {
	year := date[:4]
	month := date[4:6]
	return filepath.Join(root, year, month)
}
