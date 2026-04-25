# Update No-Flags Error Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Return an error when `notesctl update <id>` is called with no update flags, instead of silently rewriting the file unchanged.

**Architecture:** Add an early guard in the update command's `RunE` function that checks whether any update flag was explicitly set via `cmd.Flags().Changed()`. If none were set, return an error before any file I/O. Update the existing test to expect an error.

**Tech Stack:** Go, cobra (CLI framework)

---

### Task 1: Update test to expect error on no flags

**Files:**
- Modify: `internal/cli/update_test.go:196-214`

- [ ] **Step 1: Replace `TestUpdateNoFlagsUnchanged` with `TestUpdateNoFlagsErrors`**

Replace the existing test at line 196-214 with:

```go
// TestUpdateNoFlagsErrors verifies that update with no flags returns an error.
func TestUpdateNoFlagsErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runUpdate(t, root, "8823")
	if err == nil {
		t.Fatal("expected error when no update flags provided, got nil")
	}
	if !strings.Contains(err.Error(), "at least one update flag is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/cli/ -run TestUpdateNoFlagsErrors -v`
Expected: FAIL — the current code returns nil error when no flags are provided.

- [ ] **Step 3: Commit failing test**

```bash
git add internal/cli/update_test.go
git commit -m "Test that update with no flags returns error (#69)"
```

### Task 2: Add no-flags guard to update command

**Files:**
- Modify: `internal/cli/update.go:18-28`

- [ ] **Step 4: Add the early check in `update.go`**

Insert the following block after line 28 (the `updatePrivate` assignment) and before line 29 (the type validation):

```go
		// At least one update flag must be provided.
		updateFlags := []string{
			"tag", "no-tags", "title", "description",
			"slug", "no-slug", "type", "no-type",
			"public", "private",
		}
		hasFlag := false
		for _, name := range updateFlags {
			if cmd.Flags().Changed(name) {
				hasFlag = true
				break
			}
		}
		if !hasFlag {
			return fmt.Errorf("at least one update flag is required")
		}
```

- [ ] **Step 5: Run the new test to verify it passes**

Run: `go test ./internal/cli/ -run TestUpdateNoFlagsErrors -v`
Expected: PASS

- [ ] **Step 6: Run the full test suite to verify no regressions**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 7: Run linter**

Run: `make lint`
Expected: No issues.

- [ ] **Step 8: Commit implementation**

```bash
git add internal/cli/update.go
git commit -m "Error when update called with no flags (#69)"
```
