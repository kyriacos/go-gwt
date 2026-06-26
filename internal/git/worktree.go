package git

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// Root returns the toplevel directory of the worktree containing the current
// directory.
func (r *CmdRepo) Root() (string, error) {
	return r.git("", "git rev-parse --show-toplevel", "rev-parse", "--show-toplevel")
}

// MainWorktree returns the path of the main worktree: the first `worktree `
// record reported by `git worktree list --porcelain`.
func (r *CmdRepo) MainWorktree() (string, error) {
	wts, err := r.fetchList()
	if err != nil {
		return "", err
	}
	if len(wts) == 0 {
		return "", fmt.Errorf("git worktree list: no worktrees found")
	}
	return wts[0].Path, nil
}

// List returns every worktree of the repository. The first entry is the main
// worktree (IsMain=true).
func (r *CmdRepo) List() ([]Worktree, error) {
	return r.fetchList()
}

// Add creates a worktree. With NewBranch it creates a branch via
// `git worktree add -b <Branch> <Path> <Base|HEAD>`; otherwise it checks out an
// existing branch via `git worktree add <Path> <Branch>`.
func (r *CmdRepo) Add(opts AddOpts) error {
	var args []string
	if opts.NewBranch {
		base := opts.Base
		if base == "" {
			base = "HEAD"
		}
		args = []string{"worktree", "add", "-b", opts.Branch, opts.Path, base}
	} else {
		args = []string{"worktree", "add", opts.Path, opts.Branch}
	}
	_, err := r.git("", "git worktree add", args...)
	if err == nil {
		r.invalidateList()
	}
	return err
}

// Remove removes the worktree at path. With force it passes --force.
func (r *CmdRepo) Remove(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := r.git("", "git worktree remove", args...)
	if err == nil {
		r.invalidateList()
	}
	return err
}

// Prune prunes worktree administrative files for deleted worktrees.
func (r *CmdRepo) Prune() error {
	_, err := r.git("", "git worktree prune", "worktree", "prune")
	if err == nil {
		r.invalidateList()
	}
	return err
}

// DiskUsage returns the total size in bytes of all regular files under path.
// It walks the tree in pure Go and does not invoke git.
func (r *CmdRepo) DiskUsage(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("disk usage %s: %w", path, err)
	}
	return total, nil
}
