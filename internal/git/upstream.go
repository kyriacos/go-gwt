package git

import (
	"fmt"
	"strings"
)

const defaultRemote = "origin"

// RemoteBranchExists reports whether refs/remotes/<remote>/<branch> exists.
func (r *CmdRepo) RemoteBranchExists(remote, branch string) (bool, error) {
	if remote == "" {
		remote = defaultRemote
	}
	_, stderr, err := r.run.Run(r.ctx, "", "git",
		"show-ref", "--verify", "--quiet", "refs/remotes/"+remote+"/"+branch)
	if err == nil {
		return true, nil
	}
	if msg := strings.TrimSpace(string(stderr)); msg != "" {
		return false, fmt.Errorf("git show-ref %s/%s: %w: %s", remote, branch, err, msg)
	}
	return false, nil
}

// BranchUpstream returns the configured upstream for a local branch. configured
// is false when the branch has no upstream.
func (r *CmdRepo) BranchUpstream(branch string) (remote, upstreamBranch string, configured bool, err error) {
	remote, err = r.git("", "git config branch upstream remote",
		"config", "--get", "branch."+branch+".remote")
	if err != nil {
		return "", "", false, nil
	}
	merge, err := r.git("", "git config branch upstream merge",
		"config", "--get", "branch."+branch+".merge")
	if err != nil {
		return "", "", false, nil
	}
	upstreamBranch = strings.TrimPrefix(merge, "refs/heads/")
	return remote, upstreamBranch, true, nil
}

// SetUpstream configures branch to track remote/upstreamBranch. Uses branch.*
// config directly so new branches work before the remote ref exists (plain
// `git push` then creates and pushes to the right branch).
func (r *CmdRepo) SetUpstream(branch, remote, upstreamBranch string) error {
	if remote == "" {
		remote = defaultRemote
	}
	if _, err := r.git("", "git config branch upstream remote",
		"config", "branch."+branch+".remote", remote); err != nil {
		return err
	}
	_, err := r.git("", "git config branch upstream merge",
		"config", "branch."+branch+".merge", "refs/heads/"+upstreamBranch)
	return err
}

// UnsetUpstream clears the upstream configuration for branch.
func (r *CmdRepo) UnsetUpstream(branch string) error {
	_, err := r.git("", "git branch --unset-upstream", "branch", "--unset-upstream", branch)
	return err
}
