# surat

A CLI tool for interacting with a date-based notes archive.

## Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

## Setup

```sh
git clone https://github.com/dreikanter/surat.git
cd surat
go mod download
```

## Build

```sh
make build       # produces ./notes binary
```

Or directly:

```sh
go build -o notes .
```

## Install

Install to `$GOPATH/bin` (or `$HOME/go/bin`):

```sh
make install
```

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
main.go          # entrypoint
cmd/             # cobra command definitions
  root.go        # root command, --path flag, path resolution
  read.go        # notes read
  filter.go      # notes filter
  ls.go          # notes ls
note/            # domain logic (parsing, scanning, matching)
  note.go        # Note struct, ParseFilename
  archive.go     # Scan, Resolve, Filter, FilterBySlug
  note_test.go   # unit tests for parsing
  archive_test.go # tests for scanning and matching
testdata/        # fixture notes for tests
```

## License

MIT
