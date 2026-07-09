package git

import (
	"fmt"
	"os"
	"strings"
)

// Worktree-state classifications, mirroring the legacy shell tool. A worktree
// is exactly one of these.
const (
	StateActive   = "active"   // branch with a live upstream
	StateLocal    = "local"    // branch with no upstream (never pushed); not stale
	StateGone     = "gone"     // upstream was deleted (e.g. merged PR); stale
	StateMissing  = "missing"  // working directory is gone / prunable; stale
	StateDetached = "detached" // not on a branch
	StateBare     = "bare"     // the bare main repo entry
)

// IsStale reports whether a state denotes a removable, stale worktree.
func IsStale(state string) bool {
	return state == StateGone || state == StateMissing
}

// BranchStates returns a map of local branch name to its upstream-derived state
// (active, local, or gone), computed in a single for-each-ref so callers can
// classify many worktrees without a git call per branch.
//
//	gone   - the branch had an upstream that no longer exists (track == "gone").
//	local  - the branch has no upstream at all (work in progress; not stale).
//	active - the branch has a live upstream.
func (r *CmdRepo) BranchStates() (map[string]string, error) {
	out, _, err := r.run.Run(r.ctx, "", "git", "for-each-ref",
		"--format=%(refname:short)%09%(upstream)%09%(upstream:track)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref: %w", err)
	}
	states := make(map[string]string)
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		name := fields[0]
		var upstream, track string
		if len(fields) > 1 {
			upstream = fields[1]
		}
		if len(fields) > 2 {
			track = fields[2]
		}
		switch {
		case strings.Contains(track, "gone"):
			states[name] = StateGone
		case upstream == "":
			states[name] = StateLocal
		default:
			states[name] = StateActive
		}
	}
	return states, nil
}

// ClassifyWorktree returns the state of a worktree given the branch-state map
// from BranchStates. Precedence: bare > missing > detached > branch state.
func ClassifyWorktree(wt Worktree, branchStates map[string]string) string {
	switch {
	case wt.Bare:
		return StateBare
	case wt.Prunable || !pathExists(wt.Path):
		return StateMissing
	case wt.Detached || wt.Branch == "":
		return StateDetached
	default:
		if s, ok := branchStates[wt.Branch]; ok {
			return s
		}
		return StateActive
	}
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

