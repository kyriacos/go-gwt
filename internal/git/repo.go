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
	run  exec.Runner
	ctx  context.Context
	list listCached
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

// The Repo methods are implemented across worktree.go, branch.go, status.go,
// and parse.go.

// compile-time check that CmdRepo satisfies Repo.
var _ Repo = (*CmdRepo)(nil)
