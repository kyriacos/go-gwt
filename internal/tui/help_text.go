package tui

// dashboardHelp is the full-screen help text shown from the dashboard (?).
const dashboardHelp = `gwt worktree dashboard

Browse local git worktrees, preview recent commits, and jump into one.
The shell wrapper cd's you when you press enter on a row.

Navigation
  ↑/k j/↓     move selection
  enter       jump into highlighted worktree
  /           filter worktrees by path or branch
  r           refresh the list
  q esc       quit

Worktrees
  n           create a new worktree (new branch)
  d           remove highlighted worktree
  D           remove worktree and delete its local branch
  o           open highlighted worktree in $EDITOR
  p           browse and check out GitHub PRs (needs gh)

Info
  ?           show this help
  c           show changelog
  esc         close help or changelog

List markers
  *           worktree you are standing in now
  colors      branch state (active, local-only, gone, detached)

Outside the dashboard, run gwt <command> --help for CLI docs.`
