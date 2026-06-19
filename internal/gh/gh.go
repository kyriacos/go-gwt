// Package gh wraps the GitHub CLI (gh). It is the only place that execs gh.
// All features degrade gracefully when gh is unavailable: Available() reports
// false and callers hide gh-dependent UI.
package gh

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// availability caches the result of the gh auth check for the process lifetime.
type availability struct {
	checked bool
	ok      bool
}

// CmdClient implements Client over the gh CLI.
type CmdClient struct {
	run exec.Runner
	ctx context.Context

	avail availability
}

// New returns a CmdClient backed by the given runner.
func New(r exec.Runner) *CmdClient {
	return &CmdClient{run: r, ctx: context.Background()}
}

// Available reports whether the gh binary is present AND authenticated. The
// result is cached on the struct for the process lifetime so repeated calls do
// not re-shell out. It never panics; any error yields false.
//
// Authentication is probed with `gh auth status`, which exits non-zero when gh
// is missing or the user is not logged in.
func (c *CmdClient) Available() bool {
	if c.avail.checked {
		return c.avail.ok
	}
	c.avail.checked = true
	_, _, err := c.run.Run(c.ctx, "", "gh", "auth", "status")
	c.avail.ok = err == nil
	return c.avail.ok
}

// runGH execs gh with the given args. It returns the trimmed stdout, or an
// error that wraps gh's stderr for context.
func (c *CmdClient) runGH(args ...string) ([]byte, error) {
	stdout, stderr, err := c.run.Run(c.ctx, "", "gh", args...)
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(stdout))
		}
		if msg != "" {
			return nil, fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return stdout, nil
}

var _ Client = (*CmdClient)(nil)
