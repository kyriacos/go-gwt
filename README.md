# gwt

A fast git worktree helper with a Charm TUI and `gh` integration.

> Status: under construction. See [PLAN.md](PLAN.md) for the design and the
> implementation roadmap.

Worktrees default to a sibling of the repo (one level up), named
`<repo>-<branch>` — so `gwt new feature` from `~/code/backend` creates
`~/code/backend-feature`.

## Install

```sh
go install github.com/kyriacos/go-gwt@latest   # binary: gwt
```

Homebrew tap and prebuilt binaries land with the first tagged release.

## Shell integration

A process cannot change its parent shell's directory, so the switch verbs print
the target path to stdout and a tiny wrapper does the `cd`:

```sh
eval "$(gwt shell-init zsh)"   # or bash | fish
```

## License

MIT — see [LICENSE](LICENSE).
