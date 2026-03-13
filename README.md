# Notes CLI

A CLI tool for managing a file-based of markdown notes.

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

## Versioning

Patch version auto-increments on each PR merge to `main` via GitHub Actions
(e.g. `v0.1.0` → `v0.1.1`). To bump minor or major, edit the version prefix in
`.github/workflows/tag.yml` and push a manual tag (e.g. `git tag v0.2.0`).

After merging, pull and reinstall locally:

```sh
git pull --tags
make install
notes --version
```

## License

MIT
