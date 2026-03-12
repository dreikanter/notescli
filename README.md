# notescli

A CLI tool for interacting with a date-based notes archive.

## Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

## Setup

```sh
git clone https://github.com/dreikanter/notescli.git
cd notescli
go mod download
```

## Build

```sh
make build       # produces ./notes binary
```

Or directly:

```sh
go build -o notes ./cmd/notes
```

## Install

Install from the repo (no local clone needed):

```sh
go install github.com/dreikanter/notescli/cmd/notes@latest
```

This places the `notes` binary in `$GOPATH/bin` (default `~/go/bin`).

Make sure `~/go/bin` is on your `PATH`:

```sh
# Add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

To update, re-run the `go install` command.

## Usage

```sh
# Show help
notes --help

# Read a note by ID
notes read 8823

# Read a note by slug
notes read todo

# Read a note by filename
notes read 20260106_8823.md

# Filter notes by fragment
notes filter todo
notes filter 2026

# List recent notes
notes ls
notes ls --limit 10
notes ls --type todo

# Override notes archive path
notes --path /path/to/notes read 8823
```

The notes archive path is resolved in this order:

1. `--path` flag
2. `NOTES_PATH` environment variable
3. `~/Dropbox/Notes` (default)

## Development

```sh
make test        # run all tests
make lint        # run golangci-lint
make clean       # remove built binary
```

Run a single test:

```sh
go test ./note/ -run TestParseFilename -v
```

## Project structure

```
cmd/notes/main.go    # binary entrypoint → produces "notes"
internal/cli/        # cobra command definitions
  root.go            # root command, --path flag, path resolution
  read.go            # notes read
  filter.go          # notes filter
  ls.go              # notes ls
note/                # domain logic (parsing, scanning, matching)
  note.go            # Note struct, ParseFilename
  archive.go         # Scan, Resolve, Filter, FilterBySlug
  note_test.go       # unit tests for parsing
  archive_test.go    # tests for scanning and matching
testdata/            # fixture notes for tests
```

## License

MIT
