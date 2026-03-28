# Notes CLI

A CLI tool for managing a file-based store of markdown notes.

## Install

```sh
go install github.com/dreikanter/notescli/cmd/notes@latest
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

# Create today's todo from the previous todo
notes new-todo

# List recent notes
notes ls
notes ls --limit 10
notes ls --type todo
notes ls --slug meeting
notes ls --tag work
notes ls --tag work --tag meeting  # multiple --tag flags are ANDed; all flags compose
notes ls --tag work --type todo
notes ls --name 2026
notes ls --today

# Note references: any command accepting a note ref resolves by ID, slug, basename, or full path
notes read 8823
notes read meeting
notes read 20260106_8823.md

# Append stdin text to a note
echo "text" | notes append 8823
echo "text" | notes append --type todo
echo "text" | notes append --type weekly --create

# Update frontmatter and rename a note
notes update 8823 --title "New Title"
notes update 8823 --tag work --tag planning
notes update 8823 --slug meeting
notes update 8823 --no-slug
notes update 8823 --type todo
notes update 8823 --no-type
notes update 8823 --no-tags

# Print path to most recent note
notes latest
notes latest --type todo
notes latest --slug meeting
notes latest --tag work

# Search note contents
notes grep "search pattern"
notes rg "search pattern"

# Print the notes store path
notes path
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
