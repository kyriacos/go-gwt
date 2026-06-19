# gwt

[![CI](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml/badge.svg)](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kyriacos/go-gwt.svg)](https://pkg.go.dev/github.com/kyriacos/go-gwt)
[![License: MIT](https://img.shields.io/github/license/kyriacos/go-gwt)](LICENSE)

A fast git worktree helper with a Charm TUI and `gh` integration. `gwt` keeps
your worktrees tidy as siblings of the repo, gives you a live dashboard over all
of them, and checks out GitHub PRs into fresh worktrees in one command.

> Status: under construction. The design is settled and the build is underway.
> See [PLAN.md](PLAN.md) for the architecture and roadmap.

Worktrees default to a sibling of the repo (one level up), named
`<repo>-<branch>` — so `gwt new feature` from `~/code/backend` creates
`~/code/backend-feature`.

## Features

- Sibling-by-default worktrees, named from a template (default `{repo}-{branch}`).
- A live Bubble Tea dashboard: every worktree with concurrent git status
  (branch, ahead/behind, dirty, last commit, disk size) and inline actions.
- `gh` integration: check out a PR into a new worktree; see PR and CI status.
- Safety checks: warns before removing a worktree with uncommitted or unpushed
  work, and refuses to remove the main worktree.
- `clean --merged`: bulk-remove worktrees whose branch is already merged.
- TOML config with a clear precedence chain and repo-local overrides.
- Post-create / pre-remove hooks, plus optional editor and tmux launch on create.

## Install

```sh
go install github.com/kyriacos/go-gwt@latest   # binary: gwt
```

Homebrew (available once the first release is tagged):

```sh
brew install kyriacos/tap/gwt
```

Prebuilt binaries for macOS and Linux (amd64/arm64) are attached to each
[release](https://github.com/kyriacos/go-gwt/releases).

## Quickstart

```sh
gwt new feature            # new branch + worktree (sibling), prints its path
gwt from existing-branch   # worktree for an existing branch (no arg = picker)
gwt co feature             # switch if it exists, else create from branch
gwt rm feature -d          # remove the worktree and delete its local branch
gwt ls                     # table of all worktrees (non-interactive)
gwt                        # no args + a tty = open the dashboard
gwt pr 1234                # check out PR #1234 into a fresh worktree
gwt clean --merged         # remove worktrees whose branch is merged
```

The switch verbs (`new`, `from`, `co`, `search`, dashboard select) print **only
the chosen path** to stdout; all diagnostics and the TUI render to the tty. That
is what makes the shell integration below able to `cd` for you.

## Shell integration

A process cannot change its parent shell's directory, so the switch verbs print
the target path to stdout and a tiny wrapper does the `cd`:

```sh
eval "$(gwt shell-init zsh)"   # or: bash | fish
```

Add that line to your `~/.zshrc` (or equivalent) to have `gwt new`/`co`/`from`
drop you into the new worktree automatically.

## Configuration

`gwt` reads `${XDG_CONFIG_HOME:-~/.config}/gwt/config.toml`, with a repo-local
`.gwt.toml` at the main worktree root taking precedence, and environment
variables / command-line flags overriding both. All keys are optional:

```toml
worktree_dir = ""                # default: parent of the main worktree
naming       = "{repo}-{branch}" # tokens: {repo} {branch} {branch_slug}
auto_setup   = "prompt"          # prompt | always | never
open_editor  = false
editor       = ""                # e.g. "cursor", "code", "nvim"; falls back to $EDITOR
tmux         = false             # open a new tmux window/session in the worktree

[hooks]
post_create = []                 # shell commands; cwd = new worktree
pre_remove  = []                 # shell commands; cwd = worktree about to go

[gh]
enabled = true                   # auto-detected; set false to disable

[ui]
color = "auto"                   # auto | always | never (also honors NO_COLOR)
```

Hooks and setup commands come from the repo and are treated as untrusted: they
run only with your consent (`auto_setup`, `--run-setup`/`--no-setup`, or an
interactive prompt; skipped by default when there is no tty).

## Documentation

The full design, command reference, and implementation roadmap live in
[PLAN.md](PLAN.md). Contributions are welcome — see
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
