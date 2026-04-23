package note

import (
	"regexp"
	"strings"
)

// taskRe matches lines like "  - [ ] some task" or "[ ] some task".
var taskRe = regexp.MustCompile(`^(\s*(?:- )?\[)(.)(\].*)$`)

// Task represents a parsed task line from a todo note.
type Task struct {
	Line       string // original full line
	Prefix     string // everything before the marker character: e.g. "  - ["
	Marker     string // single char marker: " ", "x", "+", etc.
	Suffix     string // everything after marker: e.g. "] some task"
	IsDaily    bool   // whether line contains #daily
	IsMoved    bool   // whether line contains (moved)
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
		IsDaily:    strings.Contains(line, "#daily"),
		IsMoved:    strings.Contains(line, "(moved)"),
		LineNumber: lineNumber,
	}
}

// Reassembled returns the task line with a new marker.
func (t *Task) Reassembled(marker string) string {
	return t.Prefix + marker + t.Suffix
}

// WithTag returns the task line with a tag inserted after the marker bracket.
// E.g. "- [ ] Do thing" with tag "moved" becomes "- [ ] (moved) Do thing".
// Returns the line unchanged if the tag is already present.
func (t *Task) WithTag(tag string) string {
	tagStr := "(" + tag + ")"
	if strings.Contains(t.Suffix, tagStr) {
		return t.Line
	}
	// Suffix starts with "] ", insert tag after the "] "
	if len(t.Suffix) >= 2 && t.Suffix[:2] == "] " {
		return t.Prefix + t.Marker + "] " + tagStr + " " + t.Suffix[2:]
	}
	// Suffix is just "]" with no text
	return t.Prefix + t.Marker + "] " + tagStr + t.Suffix[1:]
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
	UpdatedLines []string // modified lines of the previous todo (with (moved) tag added)
}

// RolloverTasks determines which tasks to carry over and produces the modified previous todo.
func RolloverTasks(prevLines []string) RolloverResult {
	tasks := ExtractTasks(prevLines)
	updated := make([]string, len(prevLines))
	copy(updated, prevLines)

	seen := make(map[string]bool) // normalized task text to prevent duplicates
	var carried []Task

	addTask := func(t Task) {
		// Strip (moved) from suffix so carried tasks are clean
		t.Suffix = strings.Replace(t.Suffix, "(moved) ", "", 1)
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
			if t.Marker == " " && !t.IsMoved {
				updated[t.LineNumber] = t.WithTag("moved")
			}
		case t.Marker == " " && !t.IsMoved:
			// Pending tasks: carry over and tag as moved in previous
			addTask(t)
			updated[t.LineNumber] = t.WithTag("moved")
		}
	}

	return RolloverResult{
		CarriedTasks: carried,
		UpdatedLines: updated,
	}
}

// FormatTodoContent formats carried tasks into the new todo file content.
func FormatTodoContent(tasks []Task) string {
	if len(tasks) == 0 {
		return ""
	}

	var lines []string
	for _, t := range tasks {
		// Reset marker to pending [ ]
		lines = append(lines, t.Reassembled(" "))
	}

	return strings.Join(lines, "\n\n") + "\n"
}
