# gwt

[![CI](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml/badge.svg)](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kyriacos/go-gwt.svg)](https://pkg.go.dev/github.com/kyriacos/go-gwt)
[![License: MIT](https://img.shields.io/github/license/kyriacos/go-gwt)](LICENSE)

A fast git worktree helper with a Charm TUI and `gh` integration. `gwt` keeps
your worktrees tidy as siblings of the repo, gives you a live dashboard over all
of them, and checks out GitHub PRs into fresh worktrees in one command.

Worktrees default to a sibling of the repo (one level up), named
`<repo>-<branch>` — so `gwt new feature` from `~/code/backend` creates
`~/code/backend-feature`.

## Demo

![gwt demo](docs/demo/demo.gif)

> The gif is generated with [VHS](https://github.com/charmbracelet/vhs) from
> [`docs/demo/demo.tape`](docs/demo/demo.tape) — run `vhs docs/demo/demo.tape`
> to regenerate it.

## Features

- Sibling-by-default worktrees, named from a template (default `{repo}-{branch}`).
- Color-coded worktree **states** in `ls`: active, local-only, gone (upstream
  deleted), missing, detached — with a legend when anything is stale.
- A live Bubble Tea dashboard: every worktree with concurrent git status
  (branch, ahead/behind, dirty, last commit, disk size) and inline actions.
- Interactive pickers: `from`/`co` with no argument open a branch picker; `search`
  fuzzy-finds worktrees; `clean` opens a multi-select list. When [fzf](https://github.com/junegunn/fzf)
  is installed these use fzf (matching the original bash `gwt`); otherwise the
  built-in TUI pickers are used. No-arg `from`/`co` with fzf emit a
  `GWT_POPULATE:` line so the shell wrapper can park the command for review
  before you run it.
- `gh` integration: check out a PR into a new worktree; see PR and CI status.
- Safety checks: warns before removing a worktree with uncommitted or unpushed
  work, and refuses to remove the main worktree.
- `clean`: multi-select removal of worktrees (`--merged` for a non-interactive sweep).
- TOML config with a clear precedence chain and repo-local overrides.
- Post-create / pre-remove hooks, plus optional editor and tmux launch on create.
- IDE worktree setup: runs Cursor's `.cursor/worktrees.json` commands and Claude Code's `.worktreeinclude` copies (both consent-gated).

## Install

```sh
go install github.com/kyriacos/go-gwt@latest   # binary: gwt
```

Prebuilt binaries for macOS and Linux (amd64/arm64) are attached to each
[release](https://github.com/kyriacos/go-gwt/releases).

### Build from source

Requires Go 1.26+ and `git` at runtime. [`gh`](https://cli.github.com/) is optional
(for PR commands). [`fzf`](https://github.com/junegunn/fzf) is optional but
recommended for fast interactive pickers (`search`, `clean`, and no-arg `from`/`co`).

```sh
git clone https://github.com/kyriacos/go-gwt
cd go-gwt
go build -o gwt .          # produces ./gwt
# or install into $GOBIN / $GOPATH/bin:
go install .
```

Run the test suite and linters:

```sh
go test -race ./...
go vet ./...
golangci-lint run          # optional; see .golangci.yml
```

To cut a release build locally (matches CI), install
[goreleaser](https://goreleaser.com) and run `goreleaser release --snapshot --clean`.

## Quickstart

```sh
gwt new feature            # new branch + worktree (sibling), prints its path
gwt from existing-branch   # worktree for an existing branch (no arg = picker)
gwt co feature             # switch if it exists, else create from branch
gwt search                 # fzf worktree picker (falls back to dashboard without fzf)
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

The wrapper does **not** run `gwt ls` after every `cd` by default (that used to
cost a second binary startup). To restore the old auto-list behaviour:

```sh
export GWT_AUTO_LS=1
```

When fzf is installed, `gwt from` / `gwt co` with no argument print a
`GWT_POPULATE:gwt from <branch>` (or `co`) line instead of creating the
worktree immediately. The wrapper writes that into your line buffer so you can
add flags (`--cursor-no-setup`, `--claude-no-setup`, `-p <dir>`, etc.) before pressing Enter.

If you installed the binary under a different name, pass `--name` so the wrapper
function and the command it calls match it:

```sh
eval "$(gogwt shell-init zsh --name gogwt)"
```

### Environment variables (shell)

| Variable | Effect |
|----------|--------|
| `GWT_AUTO_LS` | When set (any value), the shell wrapper runs `gwt ls` after `cd` into a worktree. Off by default for faster switches. |
| `GWT_WORKTREE_DIR` | Default parent directory for new worktrees (see config below). |
| `GWT_CURSOR_RUN_SETUP` | Whether to run Cursor worktree setup from `.cursor/worktrees.json` (`1`/`0`, or `always`/`never`/`prompt`). |
| `GWT_RUN_SETUP` | Deprecated alias for `GWT_CURSOR_RUN_SETUP`. |
| `GWT_CLAUDE_RUN_SETUP` | Whether to run Claude worktree setup from `.worktreeinclude` (`1`/`0`, or `always`/`never`/`prompt`). |

## Configuration

`gwt` reads `${XDG_CONFIG_HOME:-~/.config}/gwt/config.toml`, with a repo-local
`.gwt.toml` at the main worktree root taking precedence, and environment
variables / command-line flags overriding both. All keys are optional:

```toml
worktree_dir = ""                # default: parent of the main worktree
naming       = "{repo}-{branch}" # tokens: {repo} {branch} {branch_slug}
open_editor  = false
editor       = ""                # e.g. "cursor", "code", "nvim"; falls back to $EDITOR
tmux         = false             # open a new tmux window/session in the worktree

[cursor]
worktree_setup = "prompt"        # prompt | always | never — consent for .cursor/worktrees.json

[claude]
worktree_setup = "prompt"        # prompt | always | never — consent for .worktreeinclude copies

[hooks]
post_create = []                 # your shell commands; cwd = new worktree; always run
pre_remove  = []                 # your shell commands; cwd = worktree about to go; always run

[gh]
enabled = true                   # auto-detected; set false to disable

[ui]
color = "auto"                   # auto | always | never (also honors NO_COLOR)
```

### Worktree setup (Cursor and Claude)

After creating a worktree, `gwt` can run IDE-specific preparation. These are
separate from your own `[hooks]` (which always run without prompting).

**Cursor** — reads `<main-worktree>/.cursor/worktrees.json` and runs the
`setup-worktree` shell commands (same format [Cursor uses](https://cursor.com/docs/configuration/worktrees)).
Commands run in the new worktree with `$ROOT_WORKTREE_PATH` pointing at the main
checkout. Consent is controlled by `[cursor].worktree_setup`, `GWT_CURSOR_RUN_SETUP`,
or `--cursor-run-setup` / `--cursor-no-setup` (aliases: `--run-setup` / `--no-setup`).

If the repo has no `worktrees.json`, `gwt` silently copies a top-level `.env`
from the main worktree when the new one does not already have one (not
consent-gated).

**Claude Code** — reads `<main-worktree>/.worktreeinclude` and copies matching
gitignored files into the new worktree (same rules as
[Claude Code worktrees](https://code.claude.com/docs/en/worktrees): only paths
that match a pattern in `.worktreeinclude` *and* are ignored by the repo's
`.gitignore` are copied; tracked files are never duplicated). Consent is
controlled by `[claude].worktree_setup`, `GWT_CLAUDE_RUN_SETUP`, or
`--claude-run-setup` / `--claude-no-setup`.

`WorktreeCreate` hooks from `.claude/settings.json` are not run — those replace
git worktree creation entirely, and `gwt` has already created the worktree.

The deprecated top-level `auto_setup` key still maps to `[cursor].worktree_setup`
for backward compatibility.

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
