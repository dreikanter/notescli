# notescli

## Build & Install

```sh
make install     # builds and installs to ~/go/bin/notes
make build       # builds local ./notes binary
make test        # run tests
make lint        # run golangci-lint
```

## Versioning

Version is set at build time via git tags and `-ldflags`. The `Version` var in
`internal/cli/root.go` defaults to `"dev"` and is overridden by `make install`
/ `make build` using `git describe --tags`.

Patch version auto-increments on each PR merge via GitHub Actions
(`.github/workflows/tag.yml`), e.g. `v0.1.0` → `v0.1.1`.

After merging a PR, reinstall locally:

```sh
make update
```

## Workflow

Run `make lint` before committing or creating a PR to catch issues early.

## Commits

- One logical change per commit (atomic commits)
- Commit message: one short line, no body

## Changelog

Update `CHANGELOG.md` in every PR with an entry for the version that PR will create.

Each PR merge auto-increments the patch version. To find the next version:

```sh
git describe --tags   # e.g. v0.1.32 → next PR will be v0.1.33
```

Rules:
- One entry per PR — do not bundle multiple PRs into one entry
- Use the exact next patch version as the heading
- Reference the PR number (`[#N]`) in the entry and add its link at the bottom

## Pull Requests

Use `.github/pull_request_template.md` for all PR bodies. When running `gh pr create`, pass its content via `--body`.
