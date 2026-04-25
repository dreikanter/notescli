# new-todo: Remove rollover requirement, always create today's todo

**Issue:** [#58](https://github.com/dreikanter/notesctl/issues/58)
**Date:** 2026-04-04

## Problem

`new-todo` treats rollover from a previous todo as mandatory. When no previous
todo exists, the command fails with `"no previous todo found"`. This is wrong:
the command's purpose is to create today's todo, and rollover is an optional
convenience when a prior todo happens to exist.

The `--force` flag compounds the confusion. A user with today's todo as the
only todo runs `new-todo --force` expecting regeneration, but gets the same
error — even though a todo clearly exists.

The root cause is not the error message wording. The root cause is that the
command should never error here. There is no scenario where "no previous todo"
should prevent creating today's todo. No previous todo is equivalent to an
empty previous todo — zero tasks to carry over.

## Design

### Behavioral change

`new-todo` always succeeds in creating today's todo. The `FindLatestTodo` result
controls whether rollover happens, not whether the command proceeds:

- **Previous todo found:** roll over pending/daily tasks (current behavior,
  unchanged).
- **No previous todo found:** skip rollover, create today's todo with no
  carried tasks.

This applies to both the default and `--force` paths. The error
`"no previous todo found"` is removed entirely.

### Command description update

The current `Short` description is:

```
Create today's todo from the previous todo
```

This implies rollover is the primary purpose and reinforces the misconception
that a previous todo is required. Update to:

```
Create today's todo
```

The `--force` flag description stays as-is (`"regenerate today's todo even if
it exists"`) — it's already clear.

Update the README usage comment similarly:

```
# Create today's todo (rolls over pending tasks from the previous one)
```

This makes the actual behavior unambiguous: the command creates a todo, and
rollover is a side effect when applicable.

### Files changed

| File | Change |
|---|---|
| `internal/cli/new_todo.go` | When `FindLatestTodo` returns nil, skip rollover and proceed with empty carried tasks. Remove the error. Update `Short` description. |
| `internal/cli/new_todo_test.go` | Rename `TestNewTodoNoPreviousErrors` → `TestNewTodoNoPreviousCreatesEmpty` (expect success, verify empty todo file). Add `TestNewTodoForceOnlyTodayExists` (today's todo is the only one, `--force` succeeds). |
| `README.md` | Update `new-todo` usage comment. |
| `CHANGELOG.md` | Add entry for the fix. |

### What does NOT change

- `note/todo.go` — `RolloverTasks`, `FormatTodoContent`, `FindLatestTodo` are
  all correct. The logic change is entirely in the CLI command handler.
- `--force` semantics — it still means "regenerate even if today's todo exists."
- Rollover behavior when a previous todo exists — completely unchanged.

## Testing

1. **No previous todo (new test):** `new-todo` on an empty store creates an
   empty todo file. Verify the file exists and contains no task lines.
2. **Force with only today's todo (new test):** Create today's todo, then
   `new-todo --force`. Verify it succeeds and creates a new file.
3. **Existing tests pass unchanged:** rollover, idempotency, force-regeneration
   with a previous todo all work as before.
