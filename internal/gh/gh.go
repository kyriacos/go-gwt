// Package gh wraps the GitHub CLI (gh). It is the only place that execs gh.
// All features degrade gracefully when gh is unavailable: Available() reports
// false and callers hide gh-dependent UI.
package gh

import (
	"context"
	"errors"

	"github.com/kyriacos/go-gwt/internal/exec"
)

// ErrUnavailable is returned when gh is not installed or not authenticated.
var ErrUnavailable = errors.New("gh: not available")

// ErrNotImplemented is returned by stubs until the gh layer is built.
var ErrNotImplemented = errors.New("gh: not implemented")

// CIState is the rolled-up CI conclusion for a branch/PR.
type CIState string

const (
	CIUnknown CIState = ""
	CIPending CIState = "pending"
	CIPassing CIState = "passing"
	CIFailing CIState = "failing"
)

// PR is a pull request as needed by the UI.
type PR struct {
	Number int
	Title  string
	Author string
	Branch string // headRefName
	State  string // OPEN, etc.
	Draft  bool
}

// CIStatus summarizes checks for a branch.
type CIStatus struct {
	State  CIState
	Passed int
	Failed int
	Total  int
}

// Client is the surface higher layers depend on.
type Client interface {
	Available() bool
	ListPRs() ([]PR, error)
	Checkout(pr int) (branch string, err error) // checks out the PR branch; caller creates the worktree
	Checks(branch string) (CIStatus, error)
}

// CmdClient implements Client over the gh CLI. Methods are stubbed in the
// foundation; the gh-layer agent implements them.
type CmdClient struct {
	run exec.Runner
	ctx context.Context
}

// New returns a CmdClient backed by the given runner.
func New(r exec.Runner) *CmdClient {
	return &CmdClient{run: r, ctx: context.Background()}
}

func (c *CmdClient) Available() bool                  { return false }
func (c *CmdClient) ListPRs() ([]PR, error)           { return nil, ErrNotImplemented }
func (c *CmdClient) Checkout(pr int) (string, error)  { return "", ErrNotImplemented }
func (c *CmdClient) Checks(branch string) (CIStatus, error) { return CIStatus{}, ErrNotImplemented }

var _ Client = (*CmdClient)(nil)
