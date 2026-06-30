# gwt

[![CI](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml/badge.svg)](https://github.com/kyriacos/go-gwt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kyriacos/go-gwt.svg)](https://pkg.go.dev/github.com/kyriacos/go-gwt)
[![License: MIT](https://img.shields.io/github/license/kyriacos/go-gwt)](LICENSE)

A fast git worktree helper with a terminal UI and `gh` integration. Worktrees
land as siblings of your repo (`~/code/backend` → `~/code/backend-feature`).

![gwt demo](docs/demo/demo.gif)

## Install

```sh
go install github.com/kyriacos/go-gwt@latest
```

Prebuilt binaries: [releases](https://github.com/kyriacos/go-gwt/releases).

Optional: [`gh`](https://cli.github.com/) for PR checkout, [`fzf`](https://github.com/junegunn/fzf) if you prefer fzf pickers over the built-in TUI.

## Quickstart

```sh
gwt co feature        # switch to a worktree (create if needed)
gwt new feature       # new branch + worktree
gwt                   # dashboard (in a tty)
gwt ls                # list worktrees
gwt pr 1234           # PR into a fresh worktree
gwt clean             # remove stale worktrees
```

Use `gwt <command> --help` for full docs on any command (colored, with examples).

## Commands

| Command | What it does |
|---------|----------------|
| `co` | Switch to a worktree; create from branch if missing (`checkout`) |
| `new` | New branch + worktree |
| `from` | Worktree for an existing branch |
| `search` | Pick a worktree (`pick`) |
| `rm` | Remove a worktree (`remove`) |
| `clean` | Multi-select removal; `--merged` for a sweep |
| `ls` | List worktrees (`list`) |
| `pr` | Check out a GitHub PR |
| `dashboard` | Full-screen TUI |
| `st` | `git status -sb` (`status`) |
| `log` | Short git log graph |
| `prune` | `git worktree prune` |
| `shell-init` | Shell wrapper for auto-`cd` |
| `version` | Version info |

Interactive pickers use the built-in TUI by default (branch log preview, dashboard). Pass `--fzf` for fzf pickers.

## Shell integration

Add to `~/.zshrc` (or bash/fish equivalent):

```sh
eval "$(gwt shell-init zsh)"
```

After that, `gwt co` / `gwt new` / `gwt search` drop you into the chosen worktree.

```sh
export GWT_AUTO_LS=1          # run gwt ls after each switch
export GWT_WORKTREE_DIR=~/wt  # default parent for new worktrees
```

Different binary name:

```sh
eval "$(gogwt shell-init zsh --name gogwt)"
```

## Configuration

Global: `~/.config/gwt/config.toml`. Per-repo override: `.gwt.toml` at the main worktree root.

```toml
worktree_dir = ""                # default: parent of the repo
naming       = "{repo}-{branch}"
editor       = "cursor"          # used with --open / open_editor
tmux         = false

[cursor]
worktree_setup = "prompt"        # prompt | always | never

[claude]
worktree_setup = "prompt"

[remove]
delete_branch = false            # gwt rm: delete branch by default (like -d)
force_delete_branch = false      # gwt rm: force-delete (like -D)

[ui]
picker = "tui"                   # tui (default) | fzf
color  = "always"

[hooks]
post_create = ["npm install"]
pre_remove  = []
```

Flags like `--path`, `--fzf`, `--cursor-no-setup`, and `--open` override per invocation.
See `gwt co --help` for the full flag list.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — [LICENSE](LICENSE).
