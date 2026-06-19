# go-gwt — implementation plan

A Go rewrite of the `gwt` git-worktree helper, with a Charm/Bubble Tea TUI and
`gh` integration. Open source, single binary, cross-platform.

- Module: `github.com/kyriacos/go-gwt`
- Binary: `gwt`
- License: MIT
- Toolchain: Go 1.26+, optional `gh` and `git` (git required at runtime)

This document is the source of truth for the build. It is written so the work
can be split across parallel subagents. Read "Work breakdown" (bottom) for the
agent split and the contracts that let agents work without colliding.

---

## 1. Goals and non-goals

### Goals
- Faithful parity with the existing bash `gwt`: `new`, `from`, `co/checkout`,
  `rm/remove`, `ls/list`, `search/pick`, `prune`, `st/status`, `log`.
- Default worktree location = parent of the main worktree (sibling of the repo).
- Worktree directory naming via template, default `{repo}-{branch}`.
- `rm` can delete the local branch (`-d`, `-D`), with a reminder when it doesn't.
- A live Bubble Tea dashboard showing every worktree with concurrent git status
  (branch, ahead/behind, dirty, last commit, disk size), plus inline actions.
- `gh` integration: check out a PR into a new worktree; show PR/CI status.
- Safety: warn before removing a worktree with uncommitted or unpushed work.
- Bulk cleanup: remove worktrees whose branch is merged.
- Config file (TOML) with a clear precedence chain; repo-local overrides.
- Post-create / pre-remove hooks; optional editor and tmux launch on create.
- Cross-platform release: goreleaser, Homebrew tap, generated shell completions.

### Non-goals (v1)
- Reimplementing git itself. We shell out to `git` and parse porcelain.
- A daemon or background watcher. The dashboard refreshes on demand/interval.
- Windows-first support. Target macOS + Linux; keep code Windows-friendly but
  don't invest in Windows CI for v1.

### The one hard constraint
A process cannot change its parent shell's working directory. The switch verbs
(`new`, `from`, `co`, `search`, and the dashboard's "enter") therefore print the
chosen path to **stdout** and rely on a tiny shell wrapper to `cd`. All
diagnostics, prompts, and the TUI render to **stderr / the tty**, never stdout.
This mirrors the existing bash design. `gwt shell-init <shell>` emits the wrapper.

---

## 2. Architecture

### Layering
```
cmd/         cobra commands — flag parsing, wiring, no business logic
internal/
  git/       git porcelain wrapper (the only place that execs git)
  gh/        gh CLI wrapper (the only place that execs gh)
  config/    TOML load + precedence resolution
  worktree/  domain service: resolve destination, naming, create/remove flow
  setup/     setup-worktree runner + lifecycle hooks
  tui/       Bubble Tea dashboard (model/update/view/keys/styles)
  shell/     embedded shell wrappers + cd protocol + completions glue
  ui/        non-TUI output: lipgloss styling, color policy, err/die, prompts
  exec/      thin exec.Command wrapper (injectable, fakeable in tests)
main.go      builds the root command and Execute()s it
```

Dependency direction is strictly downward: `cmd` → `worktree`/`tui` → `git`/
`gh`/`config`/`setup` → `exec`/`ui`. No upward imports. `git` and `gh` never
import `cmd`, `tui`, or each other.

### Testability via interfaces
The two side-effecting layers are defined as interfaces so the service and TUI
can be tested against fakes:

```go
// internal/git
type Repo interface {
    Root() (string, error)            // toplevel of current dir
    MainWorktree() (string, error)    // first entry of `worktree list`
    List() ([]Worktree, error)
    Add(opts AddOpts) error           // new branch or existing branch
    Remove(path string, force bool) error
    Prune() error
    Status(path string) (Status, error)
    BranchExists(name string) (bool, error)
    DeleteBranch(name string, force bool) error
    IsMerged(branch, into string) (bool, error)
    DiskUsage(path string) (int64, error)
}

// internal/gh
type Client interface {
    Available() bool
    ListPRs() ([]PR, error)
    Checkout(pr int) (branch string, err error)  // does not create the worktree
    Checks(branch string) (CIStatus, error)
}
```

`exec.Runner` underlies both: `Run(ctx, dir, name string, args ...string) (stdout, stderr []byte, err error)`.
A `fakeRunner` keyed by command line backs unit tests; integration tests use the
real runner against a temp git repo.

### Core data types (in `internal/git`, imported widely — define these first)
```go
type Worktree struct {
    Path     string
    Branch   string // "" when detached
    Head     string // short sha
    Bare     bool
    Detached bool
    IsMain   bool
}

type Status struct {
    Dirty     bool
    Staged    int
    Unstaged  int
    Untracked int
    Upstream  string // "" when no upstream
    Ahead     int
    Behind    int
}

type AddOpts struct {
    Path      string
    Branch    string
    NewBranch bool   // git worktree add -b
    Base      string // for NewBranch; default HEAD
}
```

---

## 3. Configuration

Loaded by `internal/config`. Precedence, highest first:
1. Command-line flag (e.g. `-p/--path`, `--naming`).
2. Environment variable (`GWT_WORKTREE_DIR`, `GWT_RUN_SETUP`, `GWT_NAMING`, …)
   — kept for parity with the bash tool.
3. Repo-local `.gwt.toml` at the main worktree root.
4. User config `${XDG_CONFIG_HOME:-~/.config}/gwt/config.toml`.
5. Built-in defaults.

```toml
# ~/.config/gwt/config.toml  (all keys optional)
worktree_dir = ""              # default: parent of the main worktree
naming       = "{repo}-{branch}"   # tokens: {repo} {branch} {branch_slug}
auto_setup   = "prompt"        # prompt | always | never
open_editor  = false
editor       = ""              # e.g. "cursor", "code", "nvim"; falls back to $EDITOR
tmux         = false           # open new tmux window/session in the worktree

[hooks]
post_create = []               # shell commands; cwd = new worktree
pre_remove  = []               # shell commands; cwd = worktree about to go

[gh]
enabled = true                 # auto-detected; set false to disable

[ui]
color = "auto"                 # auto | always | never  (also honors NO_COLOR)
```

`naming` always slugifies: slashes → dashes, plus `{branch_slug}` for an
aggressively sanitized variant. Hooks and setup commands come from the repo, so
they are **untrusted**: same consent model as today (`auto_setup`, `--run-setup`
/`--no-setup`, prompt when a tty exists, default to skip when none).

---

## 4. Commands (cobra)

`gwt` with no args and a tty → opens the **dashboard**. With no tty → prints help.

| Command | Notes |
|---|---|
| `new <name> [base] [-p dir] [--no-setup\|--run-setup] [--open]` | New branch + worktree. Prints path. |
| `from <branch> [-p dir] [...]` | Existing branch → worktree. No arg → fuzzy branch picker. Prints path. |
| `co\|checkout <name> [...]` | Switch if exists else create from branch. No arg → picker. Prints path. |
| `rm\|remove [name] [-f] [-d\|-D]` | Remove worktree; `-d`/`-D` delete branch; safety checks; reminder when kept. |
| `pr <number> [-p dir] [...]` | `gh pr checkout` into a fresh worktree. Prints path. |
| `ls\|list` | Table: marker, path, head, branch, ahead/behind, dirty. Non-interactive. |
| `search\|pick` | Fuzzy worktree picker (built-in, no fzf). Prints path. |
| `clean [--merged] [--dry-run]` | Bulk-remove worktrees whose branch is merged into the default branch. |
| `prune` | `git worktree prune`. |
| `st\|status [git args]` | `git status -sb` passthrough. |
| `log [git args]` | `git log --oneline --graph` passthrough. |
| `dashboard` | Explicit TUI launch (same as bare `gwt`). |
| `shell-init <zsh\|bash\|fish>` | Emit the cd wrapper for the user's rc. |
| `completion <shell>` | cobra-generated completions. |
| `version` | Version, commit, build date (ldflags). |

Switch verbs print **only the path** on stdout. `search`/picker results and
dashboard selection use the same stdout-path contract. Keep the bash wrapper's
`GWT_POPULATE:` behavior available as an option for `from`/`co` line-editing, but
the primary path is plain stdout + `cd`.

---

## 5. The dashboard (Bubble Tea)

Single-screen manager. Layout: a filterable list (left) + preview pane (right).

- **List rows**: `* repo-branch   ↑2 ↓0   ●dirty   abc1234   2d ago   12MB`
  - current worktree marked and highlighted.
- **Status is fetched concurrently** on load and on refresh: fan out
  `git status`/ahead-behind/disk across all worktrees with `errgroup`, bounded to
  ~min(8, NumCPU). Rows render immediately with a spinner, fill in as results
  land (Bubble Tea messages per worktree).
- **Preview pane**: `git log --oneline --graph -n 15` for the highlighted
  worktree; toggle to changed-files / diffstat.
- **Keybindings** (Bubbles `key` + help bar):
  - `enter` select → print path to stdout, quit (wrapper cd's there)
  - `/` filter, `r` refresh, `n` new (prompt for name), `d` remove (confirm),
    `D` remove + delete branch, `p` open PR list, `o` open in editor, `q` quit.
  - Destructive actions show a confirm modal; respect safety checks.
- **PR view** (`p`): lists open PRs via `gh`; `enter` checks one out into a new
  worktree. Hidden/disabled when `gh` is unavailable.
- **Styling**: lipgloss; honor `NO_COLOR` and `[ui].color`. Renders to the tty;
  on quit-with-selection, the chosen path is the only thing on stdout.
- **Tests**: `teatest` (charmbracelet/x/exp/teatest) golden + transition tests
  with a fake `git.Repo`.

---

## 6. Safety and behavior details

- `rm` resolves the branch before removal (worktree disappears after). Then:
  - blocks on dirty working tree unless `-f`;
  - warns on unpushed commits (`rev-list @{upstream}..HEAD`) unless `-f`;
  - refuses to remove the main worktree;
  - if standing inside the target, steps out to root first;
  - `-d` → `git branch -d` (safe, refuses unmerged); `-D` → force; on `-d`
    failure, suggests `-D`; when neither given, prints the
    "Add -d (or -D) to delete it too next time" reminder.
- `clean --merged` computes the default branch (`git symbolic-ref
  refs/remotes/origin/HEAD`, fallback `main`/`master`), lists worktrees whose
  branch `IsMerged` into it, shows them, confirms (or `--dry-run`), removes, and
  offers branch deletion.
- Naming collisions: if the resolved dir exists, error like today.
- `co` matches existing worktrees by branch name (not the new dir name), so
  `gwt co feature` still works with `repo-feature` directories.

---

## 7. Testing strategy

- **Unit**: porcelain parsing (table-driven, real `git worktree list
  --porcelain` fixtures), naming/slug templates, config precedence, destination
  resolution, safety decision logic. Back side effects with `fakeRunner`.
- **Integration**: a `testutil` helper builds a real repo in `t.TempDir()`
  (`git init`, commits, branches, worktrees) and exercises the real `git.Repo`.
  Gated by presence of `git` (always present in CI). Parallel-safe.
- **TUI**: `teatest` against a fake repo for model transitions and key handling.
- **gh**: interface + fake; one optional integration test gated on `gh` + a env
  flag, skipped by default.
- Target: race detector on (`go test -race ./...`), ~80%+ on `internal/git`,
  `internal/config`, `internal/worktree`.

---

## 8. CI / release / docs

- **CI** (GitHub Actions): `golangci-lint`, `go vet`, `go test -race` on
  macOS + Linux, `go build`. Run on PR + push to main.
- **Release** (goreleaser on tag `v*`): builds darwin/linux × amd64/arm64,
  archives include the binary, LICENSE, README, generated man page, and shell
  completions; checksums + (optional) cosign; publishes a formula to
  `kyriacos/homebrew-tap`. `go install github.com/kyriacos/go-gwt@latest` also
  works. Version via ldflags (`-X main.version=…`).
- **Docs**: `README.md` (install, quickstart, shell-init, config, dashboard
  GIF placeholder), `CONTRIBUTING.md`, `LICENSE` (MIT), `CHANGELOG.md`
  (keep-a-changelog), `.github/` issue/PR templates, `docs/` for the config
  reference and the shell-integration explainer.

---

## 9. Repository layout (target)
```
go-gwt/
  go.mod  go.sum
  main.go
  cmd/            root.go new.go from.go co.go rm.go pr.go ls.go search.go
                  clean.go prune.go status.go log.go dashboard.go shell.go
                  completion.go version.go
  internal/
    exec/         runner.go runner_fake.go runner_test.go
    git/          types.go repo.go worktree.go branch.go status.go parse.go *_test.go
    gh/           gh.go types.go gh_test.go
    config/       config.go precedence.go config_test.go
    worktree/     service.go naming.go service_test.go naming_test.go
    setup/        runner.go hooks.go runner_test.go
    tui/          model.go update.go view.go keys.go list.go preview.go
                  pr.go styles.go tui_test.go
    shell/        init.go wrapper_zsh.sh wrapper_bash.sh wrapper_fish.fish init_test.go
    ui/           output.go color.go prompt.go output_test.go
    testutil/     repo.go     // builds temp git repos for integration tests
  .github/workflows/  ci.yml release.yml
  .goreleaser.yaml
  .golangci.yml
  README.md CONTRIBUTING.md LICENSE CHANGELOG.md
  PLAN.md
```

---

## 10. Work breakdown for subagents

Strategy: one **foundation** agent lands the scaffold + shared contracts and
commits. Everything else then runs in parallel, each agent owning disjoint
directories so there are no file collisions. A final **integration** agent wires
`cmd` to the services, fixes cross-package gaps, and gets the suite green.

Each parallel agent works in its own git worktree (isolation) off the foundation
commit to avoid stepping on each other, then we merge.

### Phase 0 — Foundation (must finish and commit first; 1 agent)
- `go mod init github.com/kyriacos/go-gwt`; pick deps: cobra, bubbletea,
  bubbles, lipgloss, `BurntSushi/toml` (or `pelletier/go-toml/v2`), errgroup.
- Implement `internal/exec` (real `Runner` + `fakeRunner`).
- Define `internal/git/types.go` (Worktree, Status, AddOpts) and the `Repo`
  interface signature (stub methods returning `errNotImplemented`).
- Define `internal/gh` types + `Client` interface (stubbed).
- Define `internal/config` struct + `Load()` signature (stub).
- Define `internal/worktree` `Service` struct with constructor `New(Repo, gh.Client, *config.Config)` and method stubs.
- `internal/ui` output/color/err/die/prompt (full, it's small and shared).
- `main.go` + `cmd/root.go` skeleton that compiles and `--help`s.
- **Deliverable**: `go build ./...` passes; interfaces frozen. Commit.

### Phase 1 — Parallel implementation (after Phase 0 commit)
1. **git layer** — implement every `Repo` method: porcelain parsing, add/
   remove/prune, status (dirty + ahead/behind via `rev-list --count`),
   branch existence/delete, `IsMerged`, disk usage. Full unit + integration
   tests with `internal/testutil`. Owns `internal/git`, `internal/testutil`.
2. **config** — TOML load, env + repo-local + user precedence, validation.
   Owns `internal/config`. Tests for the full precedence matrix.
3. **worktree service** — destination resolution (sibling default), naming
   templates/slugify, create/remove orchestration, safety checks, `clean
   --merged` logic. Depends only on the `Repo`/`Client`/`Config` interfaces, so
   it tests against fakes. Owns `internal/worktree`.
4. **setup + hooks** — port `.cursor/worktrees.json` setup runner; add
   post_create/pre_remove hooks with the consent model. Owns `internal/setup`.
5. **gh** — implement `Client` over the `gh` CLI: availability, PR list,
   checkout, checks. Owns `internal/gh`.
6. **tui** — dashboard model/update/view/keys/preview/PR view against a fake
   `Repo`/`Client`; concurrent status loading. Owns `internal/tui`. teatest.
7. **shell + completions** — embedded zsh/bash/fish wrappers, `shell-init`,
   cobra completion command. Owns `internal/shell`, `cmd/shell.go`,
   `cmd/completion.go`.
8. **CI/release/docs** — `.github/workflows`, `.goreleaser.yaml`,
   `.golangci.yml`, README/CONTRIBUTING/LICENSE/CHANGELOG. Owns repo root +
   `.github`. Independent of code; can start immediately.

### Phase 2 — Integration (1 agent, after Phase 1 merges)
- Implement all `cmd/*.go` command bodies wiring flags → `worktree.Service` /
  `tui`. Resolve any interface mismatches surfaced during parallel work.
- End-to-end smoke: `gwt new`, `co`, `rm -d`, `ls`, `dashboard` (teatest),
  `pr`, `clean --dry-run` against a temp repo.
- `go test -race ./...` green; `golangci-lint` clean. Tag-ready.

### Contracts that must not drift
- The `git.Repo`, `gh.Client`, `config.Config`, and `worktree.Service`
  signatures from Phase 0 are frozen. If an agent needs a change, it adds a
  method rather than altering an existing signature, and notes it for the
  integration agent.
- stdout = path only for switch verbs; everything else to stderr/tty.
- No package imports `cmd`; `git` and `gh` import neither each other nor `tui`.
```

