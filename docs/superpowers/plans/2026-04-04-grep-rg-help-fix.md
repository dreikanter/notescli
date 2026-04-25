# grep/rg --help Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Intercept `--help` in `notesctl grep` and `notesctl rg` so Cobra's own help is shown instead of the subprocess help, and improve Long descriptions to document injected defaults.

**Architecture:** Both commands use `DisableFlagParsing: true` so all args pass through to the subprocess unmodified. We add a pre-subprocess scan inside `RunE` that returns `cmd.Help()` when `"--help"` is present. No structural changes — two small edits and two new tests.

**Tech Stack:** Go, cobra (spf13/cobra), standard library `os/exec`

---

### Task 1: Fix `grep.go` — intercept `--help` and improve Long description

**Files:**
- Modify: `internal/cli/grep.go`

- [ ] **Step 1: Update `grep.go`**

Replace the entire file with:

```go
package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var grepCmd = &cobra.Command{
	Use:   "grep [flags] <pattern>",
	Short: "Search note contents using grep",
	Long: `Search note contents using grep. Only .md files are searched; .git directories are excluded.

The following flags are injected automatically: -r (recursive), -i (case-insensitive), --include=*.md, --exclude-dir=.git. The notesctl path is appended as the last argument.`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if arg == "--help" {
				return cmd.Help()
			}
		}

		root := mustNotesPath()
		grepArgs := append([]string{"-r", "-i", "--include=*.md", "--exclude-dir=.git"}, args...)
		grepArgs = append(grepArgs, root)

		grep := exec.Command("grep", grepArgs...)
		grep.Stdout = os.Stdout
		grep.Stderr = os.Stderr

		return grep.Run()
	},
}

func init() {
	rootCmd.AddCommand(grepCmd)
}
```

---

### Task 2: Fix `rg.go` — intercept `--help` and improve Long description

**Files:**
- Modify: `internal/cli/rg.go`

- [ ] **Step 1: Update `rg.go`**

Replace the entire file with:

```go
package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var rgCmd = &cobra.Command{
	Use:   "rg [flags] <pattern>",
	Short: "Search note contents using ripgrep",
	Long: `Search note contents using ripgrep (rg). Only .md files are searched.

The following flags are injected automatically: --glob *.md, --sortr path, --heading, --no-line-number, --ignore-case. The notesctl path is appended as the last argument.`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if arg == "--help" {
				return cmd.Help()
			}
		}

		if _, err := exec.LookPath("rg"); err != nil {
			return fmt.Errorf("ripgrep (rg) is not installed; install it from https://github.com/BurntSushi/ripgrep")
		}

		root := mustNotesPath()
		rgArgs := append([]string{
			"--glob", "*.md",
			"--sortr", "path",
			"--heading",
			"--no-line-number",
			"--ignore-case",
		}, args...)
		rgArgs = append(rgArgs, root)

		rg := exec.Command("rg", rgArgs...)
		rg.Stdout = os.Stdout
		rg.Stderr = os.Stderr

		return rg.Run()
	},
}

func init() {
	rootCmd.AddCommand(rgCmd)
}
```

---

### Task 3: Add `TestGrepHelp` in `grep_test.go`

**Files:**
- Modify: `internal/cli/grep_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/grep_test.go`:

```go
func TestGrepHelp(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}

	origOut := os.Stdout
	os.Stdout = w

	rootCmd.SetOut(w)
	rootCmd.SetArgs([]string{"grep", "--help"})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stdout = origOut
	rootCmd.SetOut(nil)

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	out := string(buf[:n])
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "grep") {
		t.Errorf("expected help output to contain 'grep', got: %q", out)
	}
	if !strings.Contains(out, "--include=*.md") {
		t.Errorf("expected help output to mention injected flags, got: %q", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```
go test ./internal/cli/ -run TestGrepHelp -v
```

Expected: FAIL (test doesn't pass yet if the file hasn't been updated, or if running before Task 1)

- [ ] **Step 3: Verify test passes after Task 1 is complete**

```
go test ./internal/cli/ -run TestGrepHelp -v
```

Expected: PASS

---

### Task 4: Add `TestRgHelp` in `rg_test.go`

**Files:**
- Modify: `internal/cli/rg_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/rg_test.go`:

```go
func TestRgHelp(t *testing.T) {
	requireRg(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("cannot create pipe: %v", err)
	}

	origOut := os.Stdout
	os.Stdout = w

	rootCmd.SetOut(w)
	rootCmd.SetArgs([]string{"rg", "--help"})
	execErr := rootCmd.Execute()

	w.Close()
	os.Stdout = origOut
	rootCmd.SetOut(nil)

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	r.Close()

	out := string(buf[:n])
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "rg") {
		t.Errorf("expected help output to contain 'rg', got: %q", out)
	}
	if !strings.Contains(out, "--glob") {
		t.Errorf("expected help output to mention injected flags, got: %q", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```
go test ./internal/cli/ -run TestRgHelp -v
```

Expected: FAIL (before Task 2 is complete)

- [ ] **Step 3: Verify test passes after Task 2 is complete**

```
go test ./internal/cli/ -run TestRgHelp -v
```

Expected: PASS

---

### Task 5: Run full test suite and lint, then commit

**Files:** none new

- [ ] **Step 1: Run all tests**

```
make test
```

Expected: all tests pass

- [ ] **Step 2: Run lint**

```
make lint
```

Expected: no lint errors

- [ ] **Step 3: Update CHANGELOG.md**

Add at the top of `CHANGELOG.md` (after `# Changelog`):

```markdown
## [0.1.41] - 2026-04-04

### Fixed

- Fix `grep` and `rg` commands passing `--help` to the subprocess instead of showing notes-specific help ([#63])
- Improve `Long` descriptions for `grep` and `rg` to document injected default flags

[#63]: https://github.com/dreikanter/notesctl/pull/63
```

- [ ] **Step 4: Commit**

```bash
git add internal/cli/grep.go internal/cli/rg.go internal/cli/grep_test.go internal/cli/rg_test.go CHANGELOG.md
git commit -m "Fix --help passthrough in grep/rg commands (#63)"
```

---

### Task 6: Open PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin issue-63
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "Fix --help passthrough in grep/rg commands" --body "$(cat <<'EOF'
## Summary

- Intercept `--help` in `notesctl grep` and `notesctl rg` before passing args to the subprocess — shows Cobra help instead of the underlying tool's help
- Improve `Long` descriptions to document injected default flags for both commands

## References

- Closes #63
EOF
)"
```
