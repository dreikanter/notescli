# latest --today Flag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--today` flag to the `latest` command for symmetry with `resolve` and `append`, and clarify the semantic distinction between `latest` and `resolve` in their descriptions.

**Architecture:** `latest` already uses `note.FilterByDate` (exposed in `note/store.go`). We add a `--today` bool flag, call `FilterByDate` early in `scanAndFilter` when set, and update the error message. No new functions needed.

**Tech Stack:** Go, cobra, `note` package (`note.FilterByDate`, `note.Scan`)

---

## File Map

- Modify: `internal/cli/latest.go` — add `--today` flag, call `FilterByDate` in `scanAndFilter`
- Modify: `internal/cli/latest_test.go` — reset new flag in `runLatest`, add `TestLatestTodayNoMatch`
- Modify: `CHANGELOG.md` — add v0.1.41 entry

---

### Task 1: Add --today flag and wire it into scanAndFilter

**Files:**
- Modify: `internal/cli/latest.go`

- [ ] **Step 1: Write the failing test in `internal/cli/latest_test.go`**

Add `--today` to the `ResetFlags` block and add a new test after `TestLatestTypeNotFound`:

```go
func runLatest(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := testdataPath(t)

	// Reset flags to avoid state leaking between tests.
	latestCmd.ResetFlags()
	latestCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	latestCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	latestCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	latestCmd.Flags().Bool("today", false, "only match notes created today")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"latest", "--path", root}, args...))

	err := rootCmd.Execute()
	return strings.TrimSpace(buf.String()), err
}

func TestLatestTodayNoMatch(t *testing.T) {
	// testdata fixtures are from January 2026; running in April 2026 means
	// --today should find nothing.
	_, err := runLatest(t, "--today")
	if err == nil {
		t.Fatal("expected error when no notes match today, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/alex/src/notescli-issue-60
go test ./internal/cli/ -run TestLatestTodayNoMatch -v
```

Expected: FAIL — `unknown flag: --today`

- [ ] **Step 3: Implement the --today flag in `internal/cli/latest.go`**

Replace the entire file with:

```go
package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Print absolute path to the most recent note matching the given filters",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		n, err := scanAndFilter(cmd, root)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
		return nil
	},
}

// scanAndFilter scans notes and applies --today, --type, --slug, --tag filter flags,
// returning the most recent match.
func scanAndFilter(cmd *cobra.Command, root string) (*note.Note, error) {
	notes, err := note.Scan(root)
	if err != nil {
		return nil, err
	}

	today, _ := cmd.Flags().GetBool("today")
	types, _ := cmd.Flags().GetStringSlice("type")
	slugs, _ := cmd.Flags().GetStringSlice("slug")
	tags, _ := cmd.Flags().GetStringSlice("tag")

	if today {
		notes = note.FilterByDate(notes, time.Now().Format("20060102"))
	}

	if len(types) > 0 {
		notes = note.FilterByTypes(notes, types)
	}

	if len(slugs) > 0 {
		notes = note.FilterBySlugs(notes, slugs)
	}

	if len(tags) > 0 {
		notes, err = note.FilterByTags(notes, root, tags)
		if err != nil {
			return nil, err
		}
	}

	if len(notes) == 0 {
		if today {
			return nil, fmt.Errorf("no notes found for today")
		}
		if len(types) > 0 || len(slugs) > 0 || len(tags) > 0 {
			return nil, fmt.Errorf("no notes found matching the given criteria")
		}
		return nil, fmt.Errorf("no notes found")
	}

	return &notes[0], nil
}

func init() {
	latestCmd.Flags().Bool("today", false, "only match notes created today")
	latestCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	latestCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	latestCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	rootCmd.AddCommand(latestCmd)
}
```

- [ ] **Step 4: Run the new test to verify it passes**

```bash
go test ./internal/cli/ -run TestLatestTodayNoMatch -v
```

Expected: PASS

- [ ] **Step 5: Run the full test suite and lint**

```bash
make test && make lint
```

Expected: all tests pass, no lint errors

- [ ] **Step 6: Commit**

```bash
git add internal/cli/latest.go internal/cli/latest_test.go
git commit -m "Add --today flag to latest command"
```

---

### Task 2: Update CHANGELOG.md

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Insert new entry at top of CHANGELOG.md**

Insert immediately after `# Changelog` (before `## [0.1.40]`):

```markdown
## [0.1.41] - 2026-04-04

### Added

- Add `--today` flag to `latest` command for symmetry with `resolve --today`; clarify semantic distinction between `latest` and `resolve` ([#NN])
```

And at the bottom of the file, add:

```markdown
[#NN]: https://github.com/dreikanter/notescli/pull/NN
```

(Replace `NN` with the actual PR number once known.)

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "Update changelog for v0.1.41"
```

---

### Task 3: Open PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin issue-60
```

- [ ] **Step 2: Create PR using template**

```bash
gh pr create \
  --title "Add --today flag to latest and document resolve vs latest distinction" \
  --body "$(cat <<'EOF'
## Summary

- Add `--today` flag to `latest` command, consistent with `resolve --today` and `append --today`
- Clarify `latest` short description to distinguish it from `resolve`
- Add `TestLatestTodayNoMatch` covering the new flag

## References

- Closes #60
EOF
)"
```

- [ ] **Step 3: Note the PR number and update CHANGELOG.md if needed**

If the PR number wasn't known in Task 2, update `CHANGELOG.md` now:

```bash
# Replace NN with actual PR number
sed -i '' 's/#NN/#<actual-number>/g' CHANGELOG.md
git add CHANGELOG.md
git commit -m "Update changelog with PR number"
git push
```
