# new-todo Always-Succeed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `new-todo` always create today's todo, treating rollover as optional when a previous todo exists.

**Architecture:** Single behavioral change in the CLI command handler — when no previous todo is found, skip rollover and create an empty todo instead of erroring. Update command description to clarify that rollover is not the primary purpose.

**Tech Stack:** Go, cobra CLI framework, existing `note` package

---

### Task 1: Fix `new-todo` to succeed without a previous todo (TDD)

**Files:**
- Modify: `internal/cli/new_todo_test.go:101-107` (rename + rewrite existing test)
- Modify: `internal/cli/new_todo.go:37-41` (remove error, add skip-rollover path)

- [ ] **Step 1: Rewrite `TestNewTodoNoPreviousErrors` as `TestNewTodoNoPreviousCreatesEmpty`**

In `internal/cli/new_todo_test.go`, replace the existing test (lines 101–107) with:

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /Users/alex/src/notesctl-issue-58 && go test ./internal/cli/ -run TestNewTodoNoPreviousCreatesEmpty -v`

Expected: FAIL — the current code returns an error `"no previous todo found"`.

- [ ] **Step 3: Implement the fix in `new_todo.go`**

In `internal/cli/new_todo.go`, replace lines 37–57 (the `FindLatestTodo` call through the `WriteFile` of the previous todo) with:

```go
		// Find the most recent previous todo and roll over tasks
		var carriedTasks []note.Task
		prev := note.FindLatestTodo(notes, today)
		if prev != nil {
			prevPath := filepath.Join(root, prev.RelPath)
			prevData, err := os.ReadFile(prevPath)
			if err != nil {
				return fmt.Errorf("cannot read previous todo: %w", err)
			}
			prevLines := strings.Split(string(prevData), "\n")

			result := note.RolloverTasks(prevLines)
			carriedTasks = result.CarriedTasks

			if err := os.WriteFile(prevPath, []byte(strings.Join(result.UpdatedLines, "\n")), 0o644); err != nil {
				return fmt.Errorf("cannot update previous todo: %w", err)
			}
		}
```

Then update line 72 (the `FormatTodoContent` call) to use `carriedTasks` instead of `result.CarriedTasks`:

```go
		content := note.FormatTodoContent(carriedTasks)
```

The full `RunE` function after the change:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		root := mustNotesPath()
		today := time.Now().Format("20060102")

		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		// Check if today's todo already exists
		if !force {
			if existing := note.FindTodayTodo(notes, today); existing != nil {
				fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, existing.RelPath))
				return nil
			}
		}

		// Find the most recent previous todo and roll over tasks
		var carriedTasks []note.Task
		prev := note.FindLatestTodo(notes, today)
		if prev != nil {
			prevPath := filepath.Join(root, prev.RelPath)
			prevData, err := os.ReadFile(prevPath)
			if err != nil {
				return fmt.Errorf("cannot read previous todo: %w", err)
			}
			prevLines := strings.Split(string(prevData), "\n")

			result := note.RolloverTasks(prevLines)
			carriedTasks = result.CarriedTasks

			if err := os.WriteFile(prevPath, []byte(strings.Join(result.UpdatedLines, "\n")), 0o644); err != nil {
				return fmt.Errorf("cannot update previous todo: %w", err)
			}
		}

		// Allocate new ID and create new todo
		id, err := note.NextID(root)
		if err != nil {
			return err
		}

		filename := note.NoteFilename(today, id, "", "todo")
		dir := note.NoteDirPath(root, today)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}

		fullPath := filepath.Join(dir, filename)
		content := note.FormatTodoContent(carriedTasks)

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("cannot write todo: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	},
```

- [ ] **Step 4: Run the new test to verify it passes**

Run: `cd /Users/alex/src/notesctl-issue-58 && go test ./internal/cli/ -run TestNewTodoNoPreviousCreatesEmpty -v`

Expected: PASS

- [ ] **Step 5: Run all existing tests to verify no regressions**

Run: `cd /Users/alex/src/notesctl-issue-58 && go test ./internal/cli/ -run TestNewTodo -v`

Expected: All `TestNewTodo*` tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/new_todo.go internal/cli/new_todo_test.go
git commit -m "Make new-todo succeed without a previous todo (#58)"
```

---

### Task 2: Add test for `--force` when today's todo is the only one

**Files:**
- Modify: `internal/cli/new_todo_test.go` (add new test)

- [ ] **Step 1: Write the test**

Add to `internal/cli/new_todo_test.go`:

```go
func TestNewTodoForceOnlyTodayExists(t *testing.T) {
	root := emptyNotesRoot(t)

	// Create today's todo (no previous todo to roll from).
	first, err := runNewTodo(t, root)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}

	// --force should regenerate even with no previous todo.
	second, err := runNewTodo(t, root, "--force")
	if err != nil {
		t.Fatalf("force call unexpected error: %v", err)
	}

	if first == second {
		t.Errorf("expected a different path with --force, got same path %q", first)
	}
	if _, err := os.Stat(second); err != nil {
		t.Errorf("forced file does not exist: %v", err)
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd /Users/alex/src/notesctl-issue-58 && go test ./internal/cli/ -run TestNewTodoForceOnlyTodayExists -v`

Expected: PASS (the fix from Task 1 already handles this).

- [ ] **Step 3: Commit**

```bash
git add internal/cli/new_todo_test.go
git commit -m "Add test for --force when only today's todo exists (#58)"
```

---

### Task 3: Update command description and README

**Files:**
- Modify: `internal/cli/new_todo.go:16` (command `Short` field)
- Modify: `README.md:27-28` (usage comment)

- [ ] **Step 1: Update the cobra command `Short` description**

In `internal/cli/new_todo.go`, change line 16 from:

```go
	Short: "Create today's todo from the previous todo",
```

to:

```go
	Short: "Create today's todo",
```

- [ ] **Step 2: Update the README usage comment**

In `README.md`, change line 27-28 from:

```markdown
# Create today's todo from the previous todo
notes new-todo
```

to:

```markdown
# Create today's todo (rolls over pending tasks from the previous one)
notes new-todo
```

- [ ] **Step 3: Run lint**

Run: `cd /Users/alex/src/notesctl-issue-58 && make lint`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/new_todo.go README.md
git commit -m "Update new-todo description to clarify rollover is optional (#58)"
```

---

### Task 4: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md` (add entry at top)

- [ ] **Step 1: Add changelog entry**

Add at the top of `CHANGELOG.md`, after the `# Changelog` heading and before the `## [0.1.40]` entry:

```markdown
## [0.1.41] - 2026-04-04

### Fixed

- `new-todo` no longer fails when no previous todo exists; creates an empty todo instead. `--force` works correctly when today's todo is the only one ([#58])

[#58]: https://github.com/dreikanter/notesctl/pull/58
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "Add changelog entry for v0.1.41 (#58)"
```
