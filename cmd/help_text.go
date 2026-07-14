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
  gwt co --fzf                  # fzf branch picker (parks command in shell)
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

With no argument, opens the built-in branch picker (git log preview). Pass --fzf
to use fzf instead; with fzf and no argument, parks "gwt from <branch>" in
your shell line for review before Enter.

Prints the new worktree path to stdout on success.`

	fromExample = `  gwt from release-2.1          # worktree for existing branch
  gwt from                        # TUI branch picker
  gwt from --fzf                  # fzf picker (parks command in shell)`

	coLong = `Switch to the worktree for a name/branch, creating one if needed.

If a matching worktree already exists, prints its path (shell wrapper cd's).
If not, creates a worktree from the branch — same as gwt from <name>.

With no argument, opens the built-in branch picker (git log preview). Pass --fzf
to use fzf instead; with fzf and no argument, parks "gwt co <branch>" for
review before Enter.

This is the usual day-to-day "jump to my feature branch" command.`

	coExample = `  gwt co feature                # switch if exists, else create
  gwt co                        # TUI branch picker
  gwt co --fzf                  # fzf picker (parks command in shell)
  gwt co my-fix -p ../trees     # custom parent when creating`

	rmLong = `Remove a worktree (and optionally its local branch).

With no name, removes the worktree you are currently standing in. Refuses to
remove the main worktree. Warns when uncommitted or unpushed work would be lost
unless -f / --force is given.

Set [remove].delete_branch = true in config (or GWT_DELETE_BRANCH=1) to delete
the branch by default without passing -d each time. Use force_delete_branch for
-D semantics. CLI flags always override config.`

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

By default opens the full dashboard. Pass --fzf to use an fzf picker with git
log preview (requires fzf on PATH).

Your shell wrapper cd's into the printed path.`

	searchExample = `  gwt search
  gwt pick                      # alias
  gwt search --fzf              # fzf worktree picker`

	cleanLong = `Remove unwanted worktrees.

Interactive mode (default): built-in multi-select TUI; stale worktrees (gone,
missing) are pre-marked and their branches are force-deleted on removal. Pass
--fzf to use fzf instead.

-d / -D delete the local branch for every selected worktree (like gwt rm).
Config [remove].delete_branch applies when neither flag is passed.

--merged: non-interactive sweep of worktrees merged into the default branch.
Use --dry-run with --merged to preview.`

	cleanExample = `  gwt clean                     # interactive multi-select (TUI)
  gwt clean --fzf               # fzf multi-select
  gwt clean -d                  # also delete each branch
  gwt clean -D                  # force-delete branches (unmerged OK)
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
the target path on stdout; this wrapper captures it and runs cd. The script
pins the absolute path of the gwt binary that generated it (_GWT_BIN), so a
stale copy elsewhere on PATH cannot be picked up silently.

Install once in your shell rc (use the binary you intend to run):
  eval "$(/path/to/gwt shell-init zsh)"     # zsh / bash
  /path/to/gwt shell-init fish | source     # fish

Re-run shell-init after every install or go install. Use --bin to override the
pinned path. Use --name when the binary is installed under a different name.`

	shellInitExample = `  eval "$(~/go/bin/gwt shell-init zsh)"
  bash scripts/install.sh
  eval "$(gwt shell-init zsh --name oldgwt)"`

	stLong = `Run git status -sb in the current worktree.

Passthrough to git; extra arguments are forwarded.`

	stExample = `  gwt st
  gwt status`

	logLong = `Show a short oneline git log graph for the current worktree.

Passthrough to git; extra arguments are forwarded.`

	logExample = `  gwt log`

	versionLong = `Print the gwt version, git commit, and build date.

Same output as gwt --version.`

	versionExample = `  gwt version
  gwt --version`

	doctorLong = `Check for common setup problems: stale gwt binaries on PATH and
how to refresh shell integration.

Run this when Cursor worktree setup appears to hang after confirming with y.`

	doctorExample = `  gwt doctor`
)
