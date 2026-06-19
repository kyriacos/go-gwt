package git

// Status reports the working-tree and upstream state of the worktree at path.
// The git command runs with its working directory set to path so the report is
// scoped to that worktree. When the branch has no upstream, Upstream is "" and
// Ahead/Behind are zero (the `# branch.ab` header is absent), which is handled
// gracefully by the parser.
func (r *CmdRepo) Status(path string) (Status, error) {
	out, err := r.git(path, "git status", "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Status{}, err
	}
	return parseStatusV2(out), nil
}
