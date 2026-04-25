# read filter flags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--type`, `--slug`, `--tag`, and `--today` filter flags to `notes read`, mutually exclusive with the positional argument, so users can read notes by filter criteria in a single command.

**Architecture:** The positional-ref path is unchanged (`note.ResolveRef`). A new filter path calls `note.Scan`, applies `FilterByDate`/`FilterByTypes`/`FilterBySlugs`/`FilterByTags` in sequence, then reads `notes[0]`. Validation guards prevent combining both paths. No new shared helpers — `read.go` is small enough to be self-contained.

**Tech Stack:** Go, cobra, `internal/cli` package, `note` package filter functions already in `note/store.go`

---

## File map

| Action | File | Purpose |
|---|---|---|
| Modify | `internal/cli/read.go` | Add filter flags, optional positional arg, validation, filter execution path |
| Create | `internal/cli/read_test.go` | Tests for filter flags, mutual exclusion, error cases |
| Modify | `CHANGELOG.md` | Add v0.1.41 entry |

---

### Task 1: Rewrite `internal/cli/read.go`

**Files:**
- Modify: `internal/cli/read.go`

- [ ] **Step 1: Replace the entire contents of `internal/cli/read.go`**

The new implementation:
- Changes `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
- Adds `--type`, `--slug`, `--tag`, `--today` flags
- Adds mutual-exclusion validation
- Adds filter path: Scan → FilterByDate → FilterByTypes → FilterBySlugs → FilterByTags → notes[0]
- Keeps existing positional path and `--no-frontmatter` behavior unchanged

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read [<id|path|basename|slug|type>]",
	Short: "Read a note by ref or filter flags",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()

		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		today, _ := cmd.Flags().GetBool("today")
		noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")

		hasFilters := noteType != "" || slug != "" || len(tags) > 0 || today

		var relPath string

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}
			n, err := note.ResolveRef(root, args[0])
			if err != nil {
				return err
			}
			relPath = n.RelPath
		} else if hasFilters {
			notes, err := note.Scan(root)
			if err != nil {
				return err
			}

			if today {
				notes = note.FilterByDate(notes, time.Now().Format("20060102"))
			}
			if noteType != "" {
				notes = note.FilterByTypes(notes, []string{noteType})
			}
			if slug != "" {
				notes = note.FilterBySlugs(notes, []string{slug})
			}
			if len(tags) > 0 {
				notes, err = note.FilterByTags(notes, root, tags)
				if err != nil {
					return err
				}
			}

			if len(notes) == 0 {
				return fmt.Errorf("no notes found matching the given criteria")
			}
			relPath = notes[0].RelPath
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		data, err := os.ReadFile(filepath.Join(root, relPath))
		if err != nil {
			return err
		}

		if noFrontmatter {
			data = note.StripFrontmatter(data)
		}

		_, err = cmd.OutOrStdout().Write(data)
		return err
	},
}

func registerReadFlags() {
	readCmd.Flags().String("type", "", "filter by note type")
	readCmd.Flags().String("slug", "", "filter by slug")
	readCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	readCmd.Flags().Bool("today", false, "only match notes created today")
	readCmd.Flags().BoolP("no-frontmatter", "F", false, "exclude YAML frontmatter from output")
}

func init() {
	registerReadFlags()
	rootCmd.AddCommand(readCmd)
}
```

- [ ] **Step 2: Build to verify it compiles**

```bash
make build
```

Expected: `./notes` binary created, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/read.go
git commit -m "Add filter flags to read command"
```

---

### Task 2: Write tests for `read` filter flags

**Files:**
- Create: `internal/cli/read_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/read_test.go` with the following content. The `runRead` helper mirrors `runLatest` in `latest_test.go`, but resets `readCmd` flags using `registerReadFlags()` (just as `runAppend` resets via `registerAppendFlags()`).

```go
package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runRead(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	readCmd.ResetFlags()
	registerReadFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"read", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestReadByID(t *testing.T) {
	out, err := runRead(t, "8823")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Plain note") {
		t.Errorf("expected note content, got: %s", out)
	}
}

func TestReadByTagFilter(t *testing.T) {
	out, err := runRead(t, "--tag", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 20260104_8818_meeting.md contains "Standup notes"
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected meeting note content, got: %s", out)
	}
}

func TestReadByTypeFilter(t *testing.T) {
	out, err := runRead(t, "--type", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 20260102_8814.todo.md contains "Todo"
	if !strings.Contains(out, "Todo") {
		t.Errorf("expected todo note content, got: %s", out)
	}
}

func TestReadBySlugFilter(t *testing.T) {
	out, err := runRead(t, "--slug", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected meeting note content, got: %s", out)
	}
}

func TestReadByTodayFilter(t *testing.T) {
	// No notes in testdata match today's date, so this should error.
	today := time.Now().Format("20060102")
	_, err := runRead(t, "--today")
	if err == nil {
		t.Fatalf("expected error for --today with no matching notesctl (today=%s), got nil", today)
	}
}

func TestReadPositionalArgWithFilterErrors(t *testing.T) {
	_, err := runRead(t, "8823", "--type", "todo")
	if err == nil {
		t.Fatal("expected error when combining positional arg with filter flags, got nil")
	}
}

func TestReadNoTargetErrors(t *testing.T) {
	_, err := runRead(t)
	if err == nil {
		t.Fatal("expected error when no positional arg and no filter flags, got nil")
	}
}

func TestReadNoMatchErrors(t *testing.T) {
	_, err := runRead(t, "--slug", "nonexistent-slug-xyz")
	if err == nil {
		t.Fatal("expected error when filters match nothing, got nil")
	}
}

func TestReadNoFrontmatterWithFilter(t *testing.T) {
	out, err := runRead(t, "--tag", "meeting", "--no-frontmatter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Frontmatter should be stripped; "tags:" should not appear
	if strings.Contains(out, "tags:") {
		t.Errorf("expected frontmatter stripped, got: %s", out)
	}
	if !strings.Contains(out, "Standup notes") {
		t.Errorf("expected note body, got: %s", out)
	}
}

func TestReadPositionalArgWithTodayErrors(t *testing.T) {
	_, err := runRead(t, "8823", "--today")
	if err == nil {
		t.Fatal("expected error when combining positional arg with --today, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (or pass where expected)**

```bash
make test
```

Expected: `TestReadByID` may pass (it used `readCmd` before). The new filter tests should all fail with compilation errors or wrong behavior until the implementation is in place. Since we already wrote the implementation in Task 1, all tests should pass — if any fail, diagnose before proceeding.

- [ ] **Step 3: Run tests to confirm all pass**

```bash
make test
```

Expected output includes lines like:
```
ok  	github.com/dreikanter/notesctl/internal/cli
```
No `FAIL` lines.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/read_test.go
git commit -m "Add tests for read filter flags"
```

---

### Task 3: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add v0.1.41 entry at the top of `CHANGELOG.md`**

Insert after the first `# Changelog` line, before the existing `## [0.1.40]` entry:

```markdown
## [0.1.41] - 2026-04-04

### Added

- Add `--type`, `--slug`, `--tag`, and `--today` filter flags to `read`; mutually exclusive with the positional ref argument ([#62])

[#62]: https://github.com/dreikanter/notesctl/pull/62
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "Update changelog for v0.1.41"
```

---

### Task 4: Lint and final checks

- [ ] **Step 1: Run linter**

```bash
make lint
```

Expected: no errors or warnings. If any are reported, fix them before proceeding.

- [ ] **Step 2: Run full test suite one final time**

```bash
make test
```

Expected: all tests pass, `ok` for every package.

- [ ] **Step 3: Open PR**

```bash
gh pr create \
  --title "Add filter flags to read command (#62)" \
  --body "$(cat .github/pull_request_template.md)"
```

Fill in the PR template body with:
- **What**: Add `--type`, `--slug`, `--tag`, `--today` filter flags to `notes read`
- **Why**: Fixes #62 — reading by tag previously required two commands
- **How**: Same pattern as `append` — positional arg becomes optional, filters are mutually exclusive with it, most recent matching note is read
