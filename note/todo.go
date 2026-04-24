package note

import (
	"regexp"
	"strings"
)

// taskRe matches lines like "- [ ] some task" or "  - [x] some task".
var taskRe = regexp.MustCompile(`^(\s*- \[)( |x)(\].*)$`)

// Task represents a parsed task line from a todo note.
type Task struct {
	Line       string // original full line
	Text       string // trimmed task text, e.g. "Buy milk #daily"
	Done       bool   // true when the marker is "x"
	IsDaily    bool   // whether line contains #daily
	IsMoved    bool   // whether line contains (moved)
	LineNumber int    // 0-based index in the source file lines

	// regex capture groups kept unexported; use Reassembled / WithTag to rebuild lines.
	prefix string
	marker string
	suffix string
}

// ParseTask attempts to parse a line as a task. Returns nil if not a task line.
func ParseTask(line string, lineNumber int) *Task {
	m := taskRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	suffix := m[3]
	text := ""
	if len(suffix) >= 2 && suffix[:2] == "] " {
		text = strings.TrimSpace(suffix[2:])
	} else if len(suffix) > 1 {
		text = strings.TrimSpace(suffix[1:])
	}
	return &Task{
		Line:       line,
		Text:       text,
		Done:       m[2] == "x",
		IsDaily:    strings.Contains(line, "#daily"),
		IsMoved:    strings.Contains(line, "(moved)"),
		LineNumber: lineNumber,
		prefix:     m[1],
		marker:     m[2],
		suffix:     suffix,
	}
}

// Reassembled returns the task line with a new marker.
func (t *Task) Reassembled(marker string) string {
	return t.prefix + marker + t.suffix
}

// WithTag returns the task line with a tag inserted after the marker bracket.
// E.g. "- [ ] Do thing" with tag "moved" becomes "- [ ] (moved) Do thing".
// Returns the line unchanged if the tag is already present.
func (t *Task) WithTag(tag string) string {
	tagStr := "(" + tag + ")"
	if strings.Contains(t.suffix, tagStr) {
		return t.Line
	}
	// suffix starts with "] ", insert tag after the "] "
	if len(t.suffix) >= 2 && t.suffix[:2] == "] " {
		return t.prefix + t.marker + "] " + tagStr + " " + t.suffix[2:]
	}
	// suffix is just "]" with no text
	return t.prefix + t.marker + "] " + tagStr + t.suffix[1:]
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
		t.suffix = strings.Replace(t.suffix, "(moved) ", "", 1)
		// Normalize: strip leading whitespace, bullet, and marker for dedup
		key := strings.TrimSpace(t.suffix)
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
			if t.marker == " " && !t.IsMoved {
				updated[t.LineNumber] = t.WithTag("moved")
			}
		case t.marker == " " && !t.IsMoved:
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
