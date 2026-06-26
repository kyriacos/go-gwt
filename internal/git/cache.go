package git

// listCached holds the result of a single `git worktree list --porcelain` for
// the process lifetime. Mutating commands invalidate it so repeated List /
// MainWorktree / findWorktree calls in one invocation share one subprocess.
type listCached struct {
	ok  bool
	wts []Worktree
	err error
}

func (r *CmdRepo) invalidateList() {
	r.list.ok = false
}

func (r *CmdRepo) fetchList() ([]Worktree, error) {
	if r.list.ok {
		return r.list.wts, r.list.err
	}
	out, err := r.git("", "git worktree list", "worktree", "list", "--porcelain")
	if err != nil {
		r.list.err = err
		r.list.ok = true
		return nil, err
	}
	r.list.wts = parseWorktreeList(out)
	r.list.err = nil
	r.list.ok = true
	return r.list.wts, nil
}
