// Package worktree holds the domain logic that sits above git/gh/config: it
// resolves destination paths, applies the naming template, and orchestrates
// create/remove flows with their safety checks. It depends only on the
// git.Repo and gh.Client interfaces, so it is fully testable against fakes.
package worktree

import (
	"errors"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
)

// ErrNotImplemented is returned by stubs until the service is built.
var ErrNotImplemented = errors.New("worktree: not implemented")

// Service orchestrates worktree operations.
type Service struct {
	Repo git.Repo
	GH   gh.Client
	Cfg  config.Config
}

// New builds a Service.
func New(repo git.Repo, ghc gh.Client, cfg config.Config) *Service {
	return &Service{Repo: repo, GH: ghc, Cfg: cfg}
}

// CreateOpts drives New/From/PR creation.
type CreateOpts struct {
	Name        string // branch name (new) or existing branch
	Base        string // base ref when creating a new branch
	NewBranch   bool
	ParentDir   string    // overrides config; empty means resolve from config
	SetupChoice SetupMode // explicit setup decision from a flag
	OpenEditor  bool
}

// SetupMode is an explicit per-invocation setup decision.
type SetupMode int

const (
	SetupDefault SetupMode = iota // fall back to config
	SetupYes
	SetupNo
)

// RemoveOpts drives removal.
type RemoveOpts struct {
	Target       string // name/branch; empty means the current worktree
	Force        bool
	DeleteBranch bool // -d
	ForceDelete  bool // -D
}

// Result reports what an operation produced.
type Result struct {
	Path   string
	Branch string
}

// Create makes a new worktree (new branch, existing branch, or PR checkout)
// and returns its path. Implemented by the worktree-service agent.
func (s *Service) Create(opts CreateOpts) (Result, error) { return Result{}, ErrNotImplemented }

// Switch returns the path of the worktree matching name, creating it from an
// existing branch if none exists (the `co` semantics).
func (s *Service) Switch(name string, opts CreateOpts) (Result, error) {
	return Result{}, ErrNotImplemented
}

// Remove deletes a worktree (and optionally its branch) after safety checks.
func (s *Service) Remove(opts RemoveOpts) (Result, error) { return Result{}, ErrNotImplemented }

// ResolveDest computes the destination directory for a branch name, applying
// the parent-dir precedence and the naming template.
func (s *Service) ResolveDest(name, parentOverride string) (string, error) {
	return "", ErrNotImplemented
}

// CleanMerged removes worktrees whose branch is merged into the default branch.
// When dryRun is true it reports candidates without removing anything.
func (s *Service) CleanMerged(dryRun bool) ([]Result, error) { return nil, ErrNotImplemented }
