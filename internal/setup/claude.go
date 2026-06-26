package setup

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyriacos/go-gwt/internal/ui"
)

// worktreeIncludeName is the Claude Code file at the repo root listing gitignored
// paths to copy into new worktrees (.gitignore syntax).
const worktreeIncludeName = ".worktreeinclude"

func (r *Runner) confirmClaude(paths []string) bool {
	if !ui.HasTTY() {
		ui.Warn("%s defines paths to copy, but there is no terminal to confirm; skipping.", worktreeIncludeName)
		ui.Dim("Re-run with --claude-run-setup to copy them.")
		return false
	}
	ui.Warn("This repo's %s wants to copy these gitignored files from the main worktree:", worktreeIncludeName)
	for _, p := range paths {
		ui.Dim("  %s", p)
	}
	return ui.Confirm("Run Claude worktree setup?", false)
}

// RunClaudeSetup copies gitignored files listed in <root>/.worktreeinclude into
// newPath, matching Claude Code's worktree behavior. Only paths that match a
// pattern in .worktreeinclude and are ignored by the repo's .gitignore rules
// are copied; tracked files are never duplicated. WorktreeCreate hooks are not
// run — go-gwt already created the worktree via git.
func (r *Runner) RunClaudeSetup(ctx context.Context, newPath, root string, decision Decision) error {
	paths, err := r.listWorktreeIncludePaths(ctx, root)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}

	if !r.consent(decision, r.Cfg.ClaudeWorktreeSetup(), paths, r.confirmClaude) {
		ui.Dim("Skipped %s file copies.", worktreeIncludeName)
		return nil
	}

	ui.Dim("Copying gitignored files from %s ...", worktreeIncludeName)
	return copyWorktreeIncludeFiles(root, newPath, paths)
}

// listWorktreeIncludePaths returns repo-relative paths under root that match
// .worktreeinclude and are gitignored. A missing or empty file yields no paths.
func (r *Runner) listWorktreeIncludePaths(ctx context.Context, root string) ([]string, error) {
	includePath := filepath.Join(root, worktreeIncludeName)
	if _, err := os.Stat(includePath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(includePath)
	if err != nil {
		return nil, err
	}
	if !hasWorktreeIncludePatterns(data) {
		return nil, nil
	}

	stdout, stderr, err := r.Run.Run(ctx, root, "git", "ls-files", "-o", "-i", "--exclude-standard", "-z")
	if err != nil {
		if msg := strings.TrimSpace(string(stderr)); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, err
	}

	var paths []string
	for _, rel := range parseNullSeparated(stdout) {
		if r.matchesWorktreeInclude(ctx, root, rel) {
			paths = append(paths, rel)
		}
	}
	return paths, nil
}

func hasWorktreeIncludePatterns(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return true
	}
	return false
}

// matchesWorktreeInclude reports whether rel matches a pattern in .worktreeinclude.
func (r *Runner) matchesWorktreeInclude(ctx context.Context, root, rel string) bool {
	_, _, err := r.Run.Run(ctx, root, "git",
		"-c", "core.excludesFile="+worktreeIncludeName,
		"check-ignore", "--no-index", "-q", "--", rel)
	return err == nil
}

func copyWorktreeIncludeFiles(root, newPath string, relPaths []string) error {
	for _, rel := range relPaths {
		src := filepath.Join(root, rel)
		dst := filepath.Join(newPath, rel)

		if _, err := os.Stat(dst); err == nil {
			continue // do not clobber
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		info, err := os.Stat(src)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
			return err
		}
		ui.Dim("  copied %s", rel)
	}
	return nil
}

// parseNullSeparated splits git -z output, tolerating a trailing NUL.
func parseNullSeparated(stdout []byte) []string {
	if len(stdout) == 0 {
		return nil
	}
	parts := bytes.Split(stdout, []byte{0})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(string(p)); s != "" {
			out = append(out, s)
		}
	}
	return out
}
