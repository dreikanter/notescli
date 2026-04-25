# notesctl

A command-line tool for managing a plain-text note archive you own entirely.

Every note is a markdown file in a date-stamped folder (`2026/04/20260405_9522.md`). No database, no proprietary format, no account — just files on disk, synced however you like (Dropbox, git, rsync, nothing at all). The CLI gives you fast, scriptable access to the archive from the terminal:

- **Create** notes with optional title, tags, slug, and type
- **List and filter** by type, tag, slug, or date
- **Read, append, annotate** without leaving the shell
- **Resolve** a note (by ID, type, slug, or tag) to an absolute path — so the tool composes with everything Unix already has

The tool doesn't try to be a knowledge graph, a publishing platform, or a second brain app. It covers a deliberately small scope:

1. **Capture** — get text into a file quickly (`echo "..." | notesctl new`)
2. **Retrieve** — find it again by ID, type, tag, or slug
3. **Integrate** — pipe notes into other tools, scripts, and AI assistants

Think of it as the storage layer that sits underneath your workflow, not the workflow itself. Obsidian and Logseq are rich GUIs built around their own vaults. notesctl is closer to a structured `~/notes` directory with a fast command-line interface on top — no plugins, no sync service, no lock-in. The files are yours; the CLI is optional convenience.

## Install

```sh
go install github.com/dreikanter/notesctl/cmd/notesctl@latest
```

Make sure `~/go/bin` is on your `PATH`:

```sh
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

For development, use `make build` or `make install` from a local clone.

## Usage

Commands take a note's numeric ID (printed by `new`, `ls`, `resolve`, etc.).
To act on "the most recent note of type X" or "the most recent note with slug
Y", use `notesctl resolve` or `notesctl ls --limit 1` to turn a filter into an ID
and compose with shell substitution.

```sh
# Create a new note
notesctl new
notesctl new --title "Meeting notes" --slug meeting --tag work
notesctl new --type todo --upsert   # create or return existing today's todo

# Create today's todo (rolls over pending tasks from the previous one)
notesctl new-todo

# List note IDs, newest first
notesctl ls
notesctl ls --limit 10
notesctl ls --type todo
notesctl ls --slug meeting
notesctl ls --tag work
notesctl ls --tag work --tag meeting     # multiple --tag flags are ANDed
notesctl ls --today

# Resolve a note and print its absolute path (exactly one lookup flag, or none)
notesctl resolve                    # most recent note overall
notesctl resolve --id 8823          # by exact numeric ID
notesctl resolve --type todo        # most recent note of that type
notesctl resolve --slug meeting     # most recent note with that slug
notesctl resolve --tag work         # most recent note with that tag

# Read a note by numeric ID
notesctl read 8823
notesctl read 8823 --no-frontmatter

# Fill empty frontmatter (title, description, tags) using Claude Code CLI
notesctl annotate 8823
notesctl annotate 8823 --model claude-sonnet-4-6
notesctl annotate 8823 --max-chars 4000   # truncate body before sending
notesctl annotate 8823 --timeout 2m       # override the 60s default

# Append stdin text to a note
echo "text" | notesctl append 8823

# Delete a note
notesctl rm 8823

# Update frontmatter (file is renamed automatically when slug, type, or date changes)
notesctl update 8823 --title "New Title"
notesctl update 8823 --description "One-line summary"
notesctl update 8823 --tag work --tag planning
notesctl update 8823 --no-tags
notesctl update 8823 --slug meeting
notesctl update 8823 --no-slug
notesctl update 8823 --type todo
notesctl update 8823 --no-type
notesctl update 8823 --date 20260420        # move to a different day (YYYYMMDD)
notesctl update 8823 --public
notesctl update 8823 --private

# List all tags (frontmatter + body hashtags)
notesctl tags
```

Composing with the shell — since most commands take an ID, use `ls` or
`resolve` to turn a filter into one:

```sh
# Append to the most recent note with a given slug
echo "text" | notesctl append "$(notesctl ls --slug claude-sessions --limit 1)"

# Read the most recent todo
notesctl read "$(notesctl ls --type todo --limit 1)"

# Open the most recent meeting note in $EDITOR
$EDITOR "$(notesctl resolve --slug meeting)"
```

The notes store path is resolved in this order:

1. `--path` flag
2. `NOTESCTL_PATH` environment variable

If neither is set, `notesctl` exits with an error. There is no implicit default —
set `NOTESCTL_PATH` (e.g. `export NOTESCTL_PATH=~/notes`) or pass `--path`.

## Development

```sh
make build       # build local ./notesctl binary
make install     # build and install to ~/go/bin/notesctl
make test        # run all tests
make lint        # run golangci-lint
make clean       # remove built binary
```

Run a single test:

```sh
go test ./note/ -run TestParseFilename -v
```

## Versioning

`CHANGELOG.md` is the source of truth for the version. On PR merge, GitHub
Actions (`.github/workflows/tag.yml`) reads the topmost `## [X.Y.Z]` heading
from `CHANGELOG.md` and pushes `vX.Y.Z` as a git tag. Bump major/minor/patch
by writing the desired heading in the PR.

After merging, pull and reinstall locally:

```sh
make update
```

## License

MIT
