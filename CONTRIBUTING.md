# Contributing to gwt

Thanks for your interest. This document covers how to build, test, lint, and
ship the project. The full architecture is in [PLAN.md](PLAN.md).

## Prerequisites

- Go 1.26+ (the toolchain version is pinned in `go.mod`).
- `git` on `PATH` (required at runtime and by the integration tests).
- `gh` on `PATH` is optional; only the GitHub-integration paths need it.

## Build and test

```sh
go build ./...        # build everything
go vet ./...          # static checks
go test -race ./...   # full suite with the race detector
```

Integration tests build real git repositories in `t.TempDir()` and exercise the
real `git` wrapper, so they need `git` available. They are parallel-safe.

## Lint

We use [golangci-lint](https://golangci-lint.run/); the configuration is in
[.golangci.yml](.golangci.yml).

```sh
golangci-lint run
```

CI runs build, vet, `go test -race`, and the linter on Linux and macOS for every
push to `main` and every pull request.

## Architecture and layering rule

Dependencies flow strictly downward and there are no upward imports:

```
cmd -> worktree / tui -> git / gh / config / setup -> exec / ui
```

- No package imports `cmd`.
- `git` and `gh` import neither each other nor `tui`.
- The two side-effecting layers (`git`, `gh`) are interfaces so the service and
  TUI can be tested against fakes.

### The stdout contract

The switch verbs (`new`, `from`, `co`, `search`, and the dashboard's select)
print **only the chosen path** to stdout. Every diagnostic, prompt, and the TUI
render to stderr / the tty. This is what lets the shell wrapper `cd` for the
user, so do not write anything else to stdout from those code paths.

## Commits and pull requests

- Keep changes focused; one logical change per PR.
- Conventional-style commit subjects are encouraged (`feat:`, `fix:`, `docs:`,
  `test:`, `chore:`), since the release changelog groups commits by prefix.
- Update [CHANGELOG.md](CHANGELOG.md) under `Unreleased` for user-facing changes.
- Ensure `go build ./...`, `go test -race ./...`, and `golangci-lint run` pass
  before opening a PR.

## Releases

Releases are tag-driven. Pushing a tag matching `v*` triggers the release
workflow, which runs [goreleaser](https://goreleaser.com/) to build the
cross-platform binaries, publish the GitHub Release, and update the Homebrew
tap. Maintainers: move the `Unreleased` notes into a versioned section in
`CHANGELOG.md` before tagging.
