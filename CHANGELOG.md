# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Claude Code worktree setup: copy gitignored paths from `.worktreeinclude` (consent-gated via `[claude].worktree_setup`, `GWT_CLAUDE_RUN_SETUP`, `--claude-run-setup` / `--claude-no-setup`).

### Changed

- Rename generic `auto_setup` to `[cursor].worktree_setup` for Cursor `.cursor/worktrees.json` integration; add `[claude].worktree_setup`. Deprecated keys and env vars still work.
- `gwt co`, `gwt new`, and `gwt from` now align each branch's upstream: track `origin/<branch>` when that remote ref exists, otherwise clear a stale upstream inherited from `main`.
- Cursor worktree setup: run script paths from `<worktree>/.cursor` (relative to `worktrees.json`); run command arrays from the worktree root.

[Unreleased]: https://github.com/kyriacos/go-gwt/commits/main
