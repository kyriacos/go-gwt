package git

import (
	"fmt"
	"strings"
)

// BranchExists reports whether a local branch named name exists. It uses
// `git show-ref --verify --quiet refs/heads/<name>`: exit 0 means the branch
// exists, a clean non-zero exit (no stderr) means it does not, and a non-zero
// exit accompanied by stderr is treated as a real error.
func (r *CmdRepo) BranchExists(name string) (bool, error) {
	_, stderr, err := r.run.Run(r.ctx, "", "git",
		"show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}
	if msg := strings.TrimSpace(string(stderr)); msg != "" {
		return false, fmt.Errorf("git show-ref %s: %w: %s", name, err, msg)
	}
	return false, nil
}

// DeleteBranch deletes the local branch name. Without force it uses
// `git branch -d` (refuses unmerged branches); with force it uses -D.
func (r *CmdRepo) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.git("", "git branch "+flag, "branch", flag, name)
	return err
}

// IsMerged reports whether branch is an ancestor of into (i.e. fully merged).
// It uses `git merge-base --is-ancestor <branch> <into>`: exit 0 means merged,
// exit 1 means not merged, and any other non-zero exit (with stderr) is a real
// error.
func (r *CmdRepo) IsMerged(branch, into string) (bool, error) {
	_, stderr, err := r.run.Run(r.ctx, "", "git",
		"merge-base", "--is-ancestor", branch, into)
	if err == nil {
		return true, nil
	}
	if msg := strings.TrimSpace(string(stderr)); msg != "" {
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w: %s",
			branch, into, err, msg)
	}
	return false, nil
}

// DefaultBranch returns the repository's default branch. It first reads
// `git symbolic-ref --quiet refs/remotes/origin/HEAD` and strips the
// refs/remotes/origin/ prefix. If that is unavailable (no origin remote), it
// falls back to whichever of "main" or "master" exists locally.
func (r *CmdRepo) DefaultBranch() (string, error) {
	stdout, _, err := r.run.Run(r.ctx, "", "git",
		"symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimRight(string(stdout), "\n")
		if name := strings.TrimPrefix(ref, "refs/remotes/origin/"); name != "" {
			return name, nil
		}
	}
	for _, candidate := range []string{"main", "master"} {
		exists, berr := r.BranchExists(candidate)
		if berr != nil {
			return "", berr
		}
		if exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("git default branch: no origin/HEAD and neither main nor master exists")
}
