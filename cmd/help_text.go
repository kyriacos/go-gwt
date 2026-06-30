package cmd

// Long help text and examples for subcommands. Shown by `gwt <cmd> --help`.

const (
	rootLong = `gwt manages git worktrees as siblings of the main checkout, with a live
dashboard, interactive pickers, and GitHub PR integration.

Worktrees default to <parent>/<repo>-<branch> (parent = repo's parent dir).
Switch verbs (new, from, co, search) print only the chosen path on stdout so
a shell wrapper can cd for you — see gwt shell-init --help.

Run gwt <command> --help for detailed help on any subcommand.`

	rootExample = `  gwt                           # open the dashboard (tty)
  gwt ls                        # list worktrees
  gwt co feature                # switch to or create a worktree
  gwt co --tui                  # branch picker with built-in TUI + log preview
  gwt pr 42                     # check out PR #42 into a new worktree
  eval "$(gwt shell-init zsh)"    # auto-cd after create/switch`

	newLong = `Create a new branch and a worktree for it.

The worktree is placed under the resolved parent directory:
  --path / -p  >  $GWT_WORKTREE_DIR  >  parent of the main worktree

On success, prints the new worktree path to stdout (one line).`

	newExample = `  gwt new feature               # branch "feature" from current HEAD
  gwt new feature main            # branch "feature" from "main"
  gwt new hotfix -p ~/trees       # custom parent directory
  gwt new wip --open              # open in $EDITOR / configured editor after create`

	fromLong = `Create a worktree for an existing local branch.

Does not create a new branch — the branch must already exist. For a fresh
branch + worktree, use gwt new instead.

With no argument, opens an interactive branch picker:
  • fzf (when installed): parks "gwt from <branch>" in your shell line
  • --tui: built-in picker with a git log preview pane; creates on Enter

Prints the new worktree path to stdout on success.`

	fromExample = `  gwt from release-2.1          # worktree for existing branch
  gwt from                        # interactive branch picker
  gwt from --tui                  # TUI picker with log preview
  gwt from bugfix --claude-no-setup`

	coLong = `Switch to the worktree for a name/branch, creating one if needed.

If a matching worktree already exists, prints its path (shell wrapper cd's).
If not, creates a worktree from the branch — same as gwt from <name>.

With no argument, opens an interactive branch picker:
  • fzf (when installed): parks "gwt co <branch>" for review before Enter
  • --tui: built-in split-pane picker with git log preview; switches on Enter

This is the usual day-to-day "jump to my feature branch" command.`

	coExample = `  gwt co feature                # switch if exists, else create
  gwt co                        # interactive picker (fzf when available)
  gwt co --tui                  # TUI picker with branch log preview
  gwt co my-fix -p ../trees     # custom parent when creating
  gwt co hotfix --cursor-no-setup`

	rmLong = `Remove a worktree (and optionally its local branch).

With no name, removes the worktree you are currently standing in. Refuses to
remove the main worktree. Warns when uncommitted or unpushed work would be lost
unless -f / --force is given.`

	rmExample = `  gwt rm feature                # remove worktree for "feature"
  gwt rm                        # remove current worktree
  gwt rm old -d                 # remove worktree and delete local branch
  gwt rm stale -f -D            # force discard + force-delete branch`

	prLong = `Check out a GitHub pull request into a fresh worktree.

Requires the gh CLI (gh auth login). Fetches the PR branch via gh, then creates
a sibling worktree and prints its path to stdout.`

	prExample = `  gwt pr 1234
  gwt pr 1234 --open
  gwt pr 1234 --cursor-run-setup`

	lsLong = `List all worktrees for the current repository.

Rows are color-coded by state: active, local-only, gone, missing, detached.
The current worktree is marked with *. A legend is shown when stale entries exist.`

	lsExample = `  gwt ls
  gwt list                      # alias`

	searchLong = `Fuzzy-find a worktree and print its path to stdout.

When fzf is installed, opens an fzf picker with git log preview (default).
With --tui, opens the full dashboard instead. Without fzf and without --tui,
also falls back to the dashboard.

Your shell wrapper cd's into the printed path.`

	searchExample = `  gwt search
  gwt pick                      # alias
  gwt search --tui              # use the dashboard picker`

	cleanLong = `Remove unwanted worktrees.

Interactive mode (default): multi-select picker; stale worktrees (gone,
missing) are pre-marked. Uses fzf when installed, or the built-in TUI with
--tui / when fzf is absent.

--merged: non-interactive sweep of worktrees whose branch is merged into the
default branch. Use --dry-run with --merged to preview.`

	cleanExample = `  gwt clean                     # interactive multi-select
  gwt clean --tui               # built-in TUI picker
  gwt clean --merged            # remove all merged worktrees
  gwt clean --merged --dry-run  # preview only`

	pruneLong = `Prune stale worktree metadata from git.

Run after deleting worktree directories by hand. Same as git worktree prune.`

	pruneExample = `  gwt prune`

	dashboardLong = `Open the interactive worktree dashboard.

Full-screen TUI: all worktrees with live status (ahead/behind, dirty, size),
git log preview, create/remove actions, and optional gh PR checkout (p).

Selecting a worktree prints its path to stdout for shell cd integration.`

	dashboardExample = `  gwt dashboard
  gwt                           # same when run from a tty with no args`

	shellInitLong = `Print a shell function wrapper for auto-cd integration.

A child process cannot change the parent shell's directory. Switch verbs print
the target path on stdout; this wrapper captures it and runs cd.

Install once in your shell rc:
  eval "$(gwt shell-init zsh)"     # zsh / bash
  gwt shell-init fish | source     # fish

Use --name when the binary is installed under a different name (e.g. gogwt).`

	shellInitExample = `  eval "$(gwt shell-init zsh)"
  eval "$(gogwt shell-init zsh --name gogwt)"`

	stLong = `Run git status -sb in the current worktree.

Passthrough to git; extra arguments are forwarded.`

	stExample = `  gwt st
  gwt status`

	logLong = `Show a short oneline git log graph for the current worktree.

Passthrough to git; extra arguments are forwarded.`

	logExample = `  gwt log`

	versionLong = `Print the gwt version, git commit, and build date.`

	versionExample = `  gwt version`
)
