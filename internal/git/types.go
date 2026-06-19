// Package git is the only place that execs git. It exposes the Repo interface
// over a repository's worktrees, branches, and status. Higher layers depend on
// the interface, never on the concrete implementation, so they can be tested
// against fakes.
package git

import "errors"

// ErrNotImplemented is returned by stub methods until the git layer is built.
// It exists so the foundation compiles; implemented methods must not return it.
var ErrNotImplemented = errors.New("git: not implemented")

// Worktree is one entry from `git worktree list --porcelain`.
type Worktree struct {
	Path     string
	Branch   string // "" when detached or bare
	Head     string // short sha
	Bare     bool
	Detached bool
	IsMain   bool // first entry of the list
}

// Status summarizes the working tree and upstream divergence for one worktree.
type Status struct {
	Dirty     bool
	Staged    int
	Unstaged  int
	Untracked int
	Upstream  string // "" when the branch has no upstream
	Ahead     int    // commits ahead of upstream
	Behind    int    // commits behind upstream
}

// Unpushed reports whether there are local commits not on the upstream.
func (s Status) Unpushed() bool { return s.Upstream != "" && s.Ahead > 0 }

// AddOpts describes a worktree to create.
type AddOpts struct {
	Path      string // destination directory
	Branch    string // branch to check out (existing) or create (NewBranch)
	NewBranch bool   // git worktree add -b <Branch>
	Base      string // base ref for NewBranch; empty means HEAD
}

// Repo is the surface every higher layer depends on. The concrete CmdRepo
// implements it over the git CLI.
type Repo interface {
	Root() (string, error)         // toplevel of the current directory
	MainWorktree() (string, error) // first entry of `worktree list`
	List() ([]Worktree, error)
	Add(opts AddOpts) error
	Remove(path string, force bool) error
	Prune() error
	Status(path string) (Status, error)
	BranchExists(name string) (bool, error)
	DeleteBranch(name string, force bool) error
	IsMerged(branch, into string) (bool, error)
	DefaultBranch() (string, error) // e.g. main/master via origin/HEAD
	DiskUsage(path string) (int64, error)
}
