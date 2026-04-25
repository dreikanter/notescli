# append --public/--private Flags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--public` and `--private` flags to the `append` command, gated on `--create`/`--today`, mirroring the same flags on `new`.

**Architecture:** Two files change: `internal/cli/append.go` gets two new flags, validation, and propagation to `createNoteParams`; `internal/cli/append_test.go` gets four new tests. No other files need to change — `createNote`, `createNoteParams`, and the frontmatter builder already handle the `Public` field.

**Tech Stack:** Go, cobra (flag registration), standard library

---

### Task 1: Add failing tests for --public/--private on append

**Files:**
- Modify: `internal/cli/append_test.go`

- [ ] **Step 1: Add four failing tests at the bottom of `append_test.go`**

Open `internal/cli/append_test.go` and append the following four test functions after the last existing test (`TestAppendDescriptionWithoutCreateOrTodayErrors`):

```go
func TestAppendCreateWithPublic(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "public content", "--type", "weekly", "--create", "--public")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read created file: %v", err)
	}
	if !strings.Contains(string(data), "public: true") {
		t.Errorf("expected public: true in frontmatter, got:\n%s", string(data))
	}
}

func TestAppendCreatePrivateOverridesPublic(t *testing.T) {
	root := copyTestdata(t)
	out, err := runAppend(t, root, "private content", "--type", "weekly", "--create", "--public", "--private")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("cannot read created file: %v", err)
	}
	if strings.Contains(string(data), "public:") {
		t.Errorf("expected public field absent when --private wins, got:\n%s", string(data))
	}
}

func TestAppendPublicWithoutCreateErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--public")
	if err == nil {
		t.Fatal("expected error when using --public without --create or --today, got nil")
	}
}

func TestAppendPrivateWithoutCreateErrors(t *testing.T) {
	root := copyTestdata(t)
	_, err := runAppend(t, root, "text", "--type", "todo", "--private")
	if err == nil {
		t.Fatal("expected error when using --private without --create or --today, got nil")
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

```bash
cd /Users/alex/src/notesctl-issue-68 && go test ./internal/cli/ -run "TestAppendCreateWithPublic|TestAppendCreatePrivateOverridesPublic|TestAppendPublicWithoutCreateErrors|TestAppendPrivateWithoutCreateErrors" -v
```

Expected: FAIL — `TestAppendCreateWithPublic` and `TestAppendCreatePrivateOverridesPublic` will fail because `--public`/`--private` flags don't exist yet (cobra will error "unknown flag"). `TestAppendPublicWithoutCreateErrors` and `TestAppendPrivateWithoutCreateErrors` will also fail for the same reason — the error they expect won't come from validation but the unknown-flag error may cause them to pass for the wrong reason. Either way, the implementation is missing.

---

### Task 2: Implement --public/--private flags in append.go

**Files:**
- Modify: `internal/cli/append.go`

- [ ] **Step 1: Register the two new flags in `registerAppendFlags()`**

In `internal/cli/append.go`, find `registerAppendFlags()` (line 165) and add two lines at the end:

```go
func registerAppendFlags() {
	appendCmd.Flags().String("type", "", "filter by note type")
	appendCmd.Flags().String("slug", "", "filter by slug")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	appendCmd.Flags().Bool("create", false, "create note if no match found")
	appendCmd.Flags().Bool("today", false, "append to today's note or create a new one")
	appendCmd.Flags().String("title", "", "title for frontmatter (requires --create or --today)")
	appendCmd.Flags().String("description", "", "description for frontmatter (requires --create or --today)")
	appendCmd.Flags().Bool("public", false, "mark note as public in frontmatter (requires --create or --today)")
	appendCmd.Flags().Bool("private", false, "mark note as private in frontmatter (requires --create or --today; overrides --public)")
}
```

- [ ] **Step 2: Read the new flags and add validation in `RunE`**

In `internal/cli/append.go`, find the block that reads flags (lines 39–45) and add two lines after `description`:

```go
		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		create, _ := cmd.Flags().GetBool("create")
		today, _ := cmd.Flags().GetBool("today")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		publicFlag, _ := cmd.Flags().GetBool("public")
		privateFlag, _ := cmd.Flags().GetBool("private")
```

Then find the validation block that checks `!canCreate` (lines 50–57) and extend it to cover public/private:

```go
		if !canCreate {
			if title != "" {
				return fmt.Errorf("--title requires --create or --today")
			}
			if description != "" {
				return fmt.Errorf("--description requires --create or --today")
			}
			if publicFlag {
				return fmt.Errorf("--public requires --create or --today")
			}
			if privateFlag {
				return fmt.Errorf("--private requires --create or --today")
			}
		}
```

- [ ] **Step 3: Update the "ignored when appending to existing" warning**

Find line 120–122:

```go
			if !needsCreate && (title != "" || description != "") {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: --title and --description are ignored when appending to an existing note")
			}
```

Replace with:

```go
			if !needsCreate && (title != "" || description != "" || publicFlag || privateFlag) {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: --title, --description, --public, and --private are ignored when appending to an existing note")
			}
```

- [ ] **Step 4: Pass Public to createNoteParams**

Find the `createNote(createNoteParams{...})` call inside the `if needsCreate {` block (lines 124–135) and add `Public`:

```go
				targetPath, err = createNote(createNoteParams{
					Root:        root,
					Slug:        slug,
					Type:        noteType,
					Tags:        tags,
					Title:       title,
					Description: description,
					Public:      publicFlag && !privateFlag,
				})
```

---

### Task 3: Verify tests pass and run full suite

**Files:** (read-only, run only)

- [ ] **Step 1: Run the four new tests**

```bash
cd /Users/alex/src/notesctl-issue-68 && go test ./internal/cli/ -run "TestAppendCreateWithPublic|TestAppendCreatePrivateOverridesPublic|TestAppendPublicWithoutCreateErrors|TestAppendPrivateWithoutCreateErrors" -v
```

Expected: all four PASS.

- [ ] **Step 2: Run the full test suite**

```bash
cd /Users/alex/src/notesctl-issue-68 && make test
```

Expected: all tests pass, no failures.

- [ ] **Step 3: Run lint**

```bash
cd /Users/alex/src/notesctl-issue-68 && make lint
```

Expected: no lint errors.

- [ ] **Step 4: Commit**

```bash
cd /Users/alex/src/notesctl-issue-68 && git add internal/cli/append.go internal/cli/append_test.go && git commit -m "Add --public/--private flags to append command"
```

---

### Task 4: Update CHANGELOG and open PR

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Find next version**

```bash
cd /Users/alex/src/notesctl-issue-68 && git describe --tags
```

Current tag base is `v0.1.40`, so next PR version is `v0.1.41`.

- [ ] **Step 2: Add changelog entry**

Open `CHANGELOG.md` and insert a new entry at the top (after the `# Changelog` heading and before the first existing version entry):

```markdown
## v0.1.41

- Add `--public` and `--private` flags to `append` command, gated on `--create`/`--today` [#68]

[#68]: https://github.com/dreikanter/notesctl/pull/68
```

- [ ] **Step 3: Commit changelog**

```bash
cd /Users/alex/src/notesctl-issue-68 && git add CHANGELOG.md && git commit -m "Update CHANGELOG for v0.1.41"
```

- [ ] **Step 4: Push branch**

```bash
cd /Users/alex/src/notesctl-issue-68 && git push -u origin issue-68
```

- [ ] **Step 5: Open PR using the project template**

Read `.github/pull_request_template.md` first, then run:

```bash
gh pr create --title "Add --public/--private flags to append command" --body "$(cat <<'EOF'
## Changes

- Add `--public` and `--private` flags to `append` command
- Flags are gated on `--create` or `--today` (same as `--title` and `--description`)
- `--private` overrides `--public` when both are set
- Warning emitted when flags are provided but append targets an existing note
- Four new tests covering the new flags and error cases

## Testing

- `make test` passes
- `make lint` passes

Closes #68
EOF
)"
```
