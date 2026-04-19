# Notes CLI

A command-line tool for managing a plain-text note archive you own entirely.

Every note is a markdown file in a date-stamped folder (`2026/04/20260405_9522.md`). No database, no proprietary format, no account — just files on disk, synced however you like (Dropbox, git, rsync, nothing at all). The CLI gives you fast, scriptable access to the archive from the terminal:

- **Create** notes with optional title, tags, slug, and type
- **List and filter** by date, type, tag, or slug
- **Read, append, search** without leaving the shell
- **Resolve** any reference (ID, type, substring) to a file path — so the tool composes with everything Unix already has

The tool doesn't try to be a knowledge graph, a publishing platform, or a second brain app. It covers a deliberately small scope:

1. **Capture** — get text into a file quickly (`echo "..." | notes new`)
2. **Retrieve** — find it again by ID, type, tag, or full-text search
3. **Integrate** — pipe notes into other tools, scripts, and AI assistants

Think of it as the storage layer that sits underneath your workflow, not the workflow itself. Obsidian and Logseq are rich GUIs built around their own vaults. Notes CLI is closer to a structured `~/notes` directory with a fast command-line interface on top — no plugins, no sync service, no lock-in. The files are yours; the CLI is optional convenience.

## Install

```sh
go install github.com/dreikanter/notes-cli/cmd/notes@latest
```

Make sure `~/go/bin` is on your `PATH`:

```sh
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

For development, use `make build` or `make install` from a local clone.

## Usage

```sh
# Create a new note
notes new
notes new --title "Meeting notes" --slug meeting --tag work
notes new --type todo --upsert   # create or return existing today's todo

# Create today's todo (rolls over pending tasks from the previous one)
notes new-todo

# List notes
notes ls
notes ls --limit 10
notes ls --type todo
notes ls --type todo --type backlog   # multiple --type flags are ORed
notes ls --slug meeting
notes ls --tag work
notes ls --tag work --tag meeting     # multiple --tag flags are ANDed
notes ls --name 2026
notes ls --today

# Resolve a note reference and print its absolute path
notes resolve 8823               # by ID
notes resolve todo               # by type (most recent)
notes resolve meeting            # by path substring (slug, basename, etc.)
notes resolve --type todo        # by filter flags
notes resolve --tag work --today

# Note references: any command accepting a ref resolves by ID, type, or path substring
notes read 8823
notes read meeting
notes read --type todo           # filter flags also work on read
notes read --type todo --no-frontmatter

# Open a note in $EDITOR
notes edit todo
notes edit meeting

# Fill empty frontmatter (title, description, tags) using Claude Code CLI
notes annotate 8823
notes annotate meeting --model claude-sonnet-4-6
notes annotate 8823 --max-chars 4000   # truncate body before sending

# Append stdin text to a note
echo "text" | notes append 8823
echo "text" | notes append --type todo
echo "text" | notes append --type todo --today

# Delete a note
notes rm 8823
notes rm meeting --today

# Update frontmatter and rename a note
notes update 8823 --title "New Title"
notes update 8823 --tag work --tag planning
notes update 8823 --slug meeting
notes update 8823 --no-slug
notes update 8823 --type todo
notes update 8823 --no-type
notes update 8823 --no-tags
notes update 8823 --public
notes update 8823 --private

# Search note contents
notes grep "search pattern"
notes rg "search pattern"

# List all tags (frontmatter + body hashtags)
notes tags
```

The notes store path is resolved in this order:

1. `--path` flag
2. `NOTES_PATH` environment variable
3. `~/notes` (default)

## Development

```sh
make build       # build local ./notes binary
make install     # build and install to ~/go/bin/notes
make test        # run all tests
make lint        # run golangci-lint
make clean       # remove built binary
```

Run a single test:

```sh
go test ./note/ -run TestParseFilename -v
```

## Versioning

Patch version auto-increments on each PR merge to `main` via GitHub Actions
(e.g. `v0.1.0` → `v0.1.1`). To bump minor or major, edit the version prefix in
`.github/workflows/tag.yml` and push a manual tag (e.g. `git tag v0.2.0`).

Each PR must include a `CHANGELOG.md` entry for the version it will create. Check
the next version with `git describe --tags` and increment the patch number.

After merging, pull and reinstall locally:

```sh
git pull --tags
make install
notes --version
```

## License

MIT
