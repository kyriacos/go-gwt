#!/usr/bin/env bash
# Builds a sample repo at /tmp/gwt-demo with worktrees in every state
# (active / local-only / gone) so docs/demo/demo.tape can show them off.
set -e
ROOT=/tmp/gwt-demo
rm -rf "$ROOT"; mkdir -p "$ROOT"
git init -q --bare "$ROOT/origin.git"
git init -q -b main "$ROOT/go-gwt"
cd "$ROOT/go-gwt"
git config user.email demo@example.com; git config user.name "gwt demo"
for m in "initial commit" "add cobra CLI" "add bubbletea dashboard" "gh PR integration" "fix worktree naming"; do
  git commit -q --allow-empty -m "$m"
done
git remote add origin "$ROOT/origin.git"
git push -q -u origin main
git branch feature-auth && git push -q -u origin feature-auth   # active (live upstream)
git branch spike-ideas                                          # local-only (never pushed)
git checkout -q -b old-login-pr                                 # gone (merged-PR style)
git commit -q --allow-empty -m "old login work"
git push -q -u origin old-login-pr
git checkout -q main
git push -q origin --delete old-login-pr
git fetch -q --prune
git worktree add -q "$ROOT/go-gwt-feature-auth" feature-auth
git worktree add -q "$ROOT/go-gwt-spike-ideas" spike-ideas
git worktree add -q "$ROOT/go-gwt-old-login-pr" old-login-pr
echo "demo repo ready at $ROOT/go-gwt"
