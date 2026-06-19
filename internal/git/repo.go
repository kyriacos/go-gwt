package git

import (
	"context"

	"github.com/kyriacos/go-gwt/internal/exec"
)

// CmdRepo implements Repo over the git CLI via an exec.Runner.
//
// All methods are stubbed in the foundation and return ErrNotImplemented; the
// git-layer agent implements them. The signatures here are frozen — add
// methods rather than changing existing ones.
type CmdRepo struct {
	run exec.Runner
	ctx context.Context
}

// New returns a CmdRepo backed by the given runner.
func New(r exec.Runner) *CmdRepo {
	return &CmdRepo{run: r, ctx: context.Background()}
}

// WithContext returns a copy of the repo that uses ctx for git invocations.
func (r *CmdRepo) WithContext(ctx context.Context) *CmdRepo {
	c := *r
	c.ctx = ctx
	return &c
}

func (r *CmdRepo) Root() (string, error)               { return "", ErrNotImplemented }
func (r *CmdRepo) MainWorktree() (string, error)       { return "", ErrNotImplemented }
func (r *CmdRepo) List() ([]Worktree, error)           { return nil, ErrNotImplemented }
func (r *CmdRepo) Add(opts AddOpts) error              { return ErrNotImplemented }
func (r *CmdRepo) Remove(path string, force bool) error { return ErrNotImplemented }
func (r *CmdRepo) Prune() error                        { return ErrNotImplemented }
func (r *CmdRepo) Status(path string) (Status, error)  { return Status{}, ErrNotImplemented }
func (r *CmdRepo) BranchExists(name string) (bool, error) { return false, ErrNotImplemented }
func (r *CmdRepo) DeleteBranch(name string, force bool) error { return ErrNotImplemented }
func (r *CmdRepo) IsMerged(branch, into string) (bool, error) { return false, ErrNotImplemented }
func (r *CmdRepo) DefaultBranch() (string, error)      { return "", ErrNotImplemented }
func (r *CmdRepo) DiskUsage(path string) (int64, error) { return 0, ErrNotImplemented }

// compile-time check that CmdRepo satisfies Repo.
var _ Repo = (*CmdRepo)(nil)
