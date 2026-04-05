# ls Absolute Paths Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `ls` output absolute paths so its output is consumable by other commands via Unix pipelines.

**Architecture:** One-line change in `ls` RunE to prepend the store root to each `RelPath`. Update tests to assert absolute paths. Update changelog.

**Tech Stack:** Go, cobra

---

### Task 1: Update `ls` to output absolute paths

**Files:**
- Modify: `internal/cli/ls.go:1-7` (import block) and `internal/cli/ls.go:57` (output line)

- [ ] **Step 1: Add `path/filepath` to the import block**

In `internal/cli/ls.go`, change the import block from:

```go
import (
	"fmt"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)
```

to:

```go
import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 2: Change the output line to use absolute paths**

In `internal/cli/ls.go`, line 57, change:

```go
fmt.Fprintln(cmd.OutOrStdout(), n.RelPath)
```

to:

```go
fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
```

`root` is already available in scope (line 23: `root := mustNotesPath()`).

- [ ] **Step 3: Run existing tests to see which ones break**

Run: `cd /Users/alex/src/notescli-issue-55 && go test ./internal/cli/ -run TestLs -v`

Expected: Tests that check `strings.Contains` on filenames like `"todo"`, `"meeting"`, `"8814"` should still pass because the absolute path still contains those substrings. Tests that only check line counts should still pass. Observe which (if any) fail.

### Task 2: Update tests to verify absolute paths

**Files:**
- Modify: `internal/cli/ls_test.go`

- [ ] **Step 1: Add an assertion that `ls` output contains absolute paths**

Add a new test to `internal/cli/ls_test.go` that verifies output lines are absolute paths:

```go
func TestLsOutputsAbsolutePaths(t *testing.T) {
	out, err := runLs(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	root := testdataPath(t)
	for _, line := range strings.Split(out, "\n") {
		if !filepath.IsAbs(line) {
			t.Errorf("expected absolute path, got %q", line)
		}
		if !strings.HasPrefix(line, root) {
			t.Errorf("expected path under %s, got %q", root, line)
		}
	}
}
```

- [ ] **Step 2: Add `path/filepath` to the test file imports**

In `internal/cli/ls_test.go`, change the import block from:

```go
import (
	"bytes"
	"strings"
	"testing"
)
```

to:

```go
import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)
```

- [ ] **Step 3: Run all ls tests**

Run: `cd /Users/alex/src/notescli-issue-55 && go test ./internal/cli/ -run TestLs -v`

Expected: All tests pass, including the new `TestLsOutputsAbsolutePaths`.

- [ ] **Step 4: Run full test suite**

Run: `cd /Users/alex/src/notescli-issue-55 && make test`

Expected: All tests pass.

- [ ] **Step 5: Run linter**

Run: `cd /Users/alex/src/notescli-issue-55 && make lint`

Expected: No lint errors.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/ls.go internal/cli/ls_test.go
git commit -m "Output absolute paths from ls command (#55)"
```

### Task 3: Update changelog

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add changelog entry for v0.1.41**

At the top of `CHANGELOG.md`, after the `# Changelog` heading and before the `## [0.1.40]` entry, add:

```markdown
## [0.1.41] - 2026-04-04

### Fixed

- Output absolute paths from `ls` to enable Unix pipelines like `notes ls | xargs notes read` ([#55])

[#55]: https://github.com/dreikanter/notescli/pull/55
```

Note: The `[#55]` link reference goes at the end of the new entry block, before the blank line separating it from the `## [0.1.40]` entry. Follow the same pattern as existing entries (each entry has its own link reference immediately after it).

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "Add changelog entry for v0.1.41"
```
