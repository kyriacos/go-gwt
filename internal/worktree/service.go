// Package worktree holds the domain logic that sits above git/gh/config: it
// resolves destination paths, applies the naming template, and orchestrates
// create/remove flows with their safety checks. It depends only on the
// git.Repo and gh.Client interfaces (plus an exec.Runner and a *setup.Runner
// for side effects), so it is fully testable against fakes.
//
// # Service shape (for the integration agent wiring cmd/)
//
// The constructor and field list are:
//
//	func New(repo git.Repo, ghc gh.Client, cfg config.Config, run exec.Runner) *Service
//
//	type Service struct {
//	    Repo  git.Repo        // git porcelain surface
//	    GH    gh.Client       // gh CLI surface (may be unavailable)
//	    Cfg   config.Config   // fully-resolved config
//	    Run   exec.Runner     // used to launch the editor / tmux
//	    Setup *setup.Runner   // runs repo setup commands + lifecycle hooks
//	}
//
// New constructs the *setup.Runner internally from (run, cfg). Pass the same
// exec.Runner used elsewhere. A nil run is tolerated (editor/tmux/setup become
// no-ops) so the zero-config path stays safe.
package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/setup"
	"github.com/kyriacos/go-gwt/internal/ui"
)

// ErrNotImplemented is returned by stubs until the service is built.
var ErrNotImplemented = errors.New("worktree: not implemented")

// Service orchestrates worktree operations.
type Service struct {
	Repo  git.Repo
	GH    gh.Client
	Cfg   config.Config
	Run   exec.Runner
	Setup *setup.Runner
}

// New builds a Service. It wires a setup.Runner from run+cfg so create/remove
// flows can execute repo setup commands and user hooks.
func New(repo git.Repo, ghc gh.Client, cfg config.Config, run exec.Runner) *Service {
	return &Service{
		Repo:  repo,
		GH:    ghc,
		Cfg:   cfg,
		Run:   run,
		Setup: setup.New(run, cfg),
	}
}

// CreateOpts drives New/From/PR creation.
type CreateOpts struct {
	Name              string // branch name (new) or existing branch
	Base              string // base ref when creating a new branch
	NewBranch         bool
	ParentDir         string    // overrides config; empty means resolve from config
	CursorSetupChoice SetupMode // explicit Cursor worktree_setup decision from a flag
	ClaudeSetupChoice SetupMode // explicit Claude worktree_setup decision from a flag
	OpenEditor        bool
}

// SetupMode is an explicit per-invocation worktree_setup decision for an IDE
// integration (Cursor or Claude).
type SetupMode int

const (
	SetupDefault SetupMode = iota // fall back to config
	SetupYes
	SetupNo
)

// decision maps a SetupMode to the setup package's Decision.
func (m SetupMode) decision() setup.Decision {
	switch m {
	case SetupYes:
		return setup.DecisionYes
	case SetupNo:
		return setup.DecisionNo
	default:
		return setup.DecisionDefault
	}
}

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
// and returns its path. PR checkout is handled by cmd (gh.Checkout then Create
// with NewBranch=false), so Create only needs to add an existing branch when
// NewBranch is false.
func (s *Service) Create(opts CreateOpts) (Result, error) {
	root, err := s.Repo.MainWorktree()
	if err != nil {
		return Result{}, fmt.Errorf("locate main worktree: %w", err)
	}

	dest, err := s.ResolveDest(opts.Name, opts.ParentDir)
	if err != nil {
		return Result{}, err
	}

	if err := s.Repo.Add(git.AddOpts{
		Path:      dest,
		Branch:    opts.Name,
		NewBranch: opts.NewBranch,
		Base:      opts.Base,
	}); err != nil {
		return Result{}, fmt.Errorf("create worktree: %w", err)
	}

	s.alignBranchUpstream(opts.Name)

	ctx := context.Background()

	// Lifecycle: user hooks first (trusted), then IDE setup (consent-gated).
	if s.Setup != nil {
		if err := s.Setup.RunHooks(ctx, setup.PostCreate, dest, root); err != nil {
			ui.Warn("post_create hooks: %v", err)
		}
		if err := s.Setup.RunCursorSetup(ctx, dest, root, opts.CursorSetupChoice.decision()); err != nil {
			ui.Warn("cursor setup: %v", err)
		}
		if err := s.Setup.RunClaudeSetup(ctx, dest, root, opts.ClaudeSetupChoice.decision()); err != nil {
			ui.Warn("claude setup: %v", err)
		}
	}

	ui.OK("Created worktree for '%s' at %s", opts.Name, dest)

	// Optional editor / tmux launch in the new worktree.
	if opts.OpenEditor || s.Cfg.OpenEditor {
		s.openEditor(ctx, dest)
	}
	if s.Cfg.Tmux {
		s.openTmux(ctx, dest, opts.Name)
	}

	return Result{Path: dest, Branch: opts.Name}, nil
}

// Switch returns the path of the worktree matching name, creating it from an
// existing branch if none exists (the `co` semantics). A match is any worktree
// whose branch equals name (case-insensitively) or whose directory basename
// equals name, or whose path matches the resolved destination (handles
// case-only directory collisions on case-insensitive filesystems).
func (s *Service) Switch(name string, opts CreateOpts) (Result, error) {
	wt, found, err := s.findWorktree(name)
	if err != nil {
		return Result{}, err
	}
	if found {
		s.alignBranchUpstream(wt.Branch)
		return Result{Path: wt.Path, Branch: wt.Branch}, nil
	}

	// On case-insensitive filesystems the resolved destination path may already
	// be a registered worktree even when branch/basename matching failed.
	dest, err := s.computeDest(name, opts.ParentDir)
	if err != nil {
		return Result{}, err
	}
	if wt, found, err := s.findWorktreeByPath(dest); err != nil {
		return Result{}, err
	} else if found {
		s.alignBranchUpstream(wt.Branch)
		return Result{Path: wt.Path, Branch: wt.Branch}, nil
	}

	// No existing worktree: create one from the existing branch.
	co := opts
	co.Name = name
	co.NewBranch = false
	return s.Create(co)
}

// Remove deletes a worktree (and optionally its branch) after safety checks.
func (s *Service) Remove(opts RemoveOpts) (Result, error) {
	root, err := s.Repo.MainWorktree()
	if err != nil {
		return Result{}, fmt.Errorf("locate main worktree: %w", err)
	}

	// Resolve the target path.
	var target string
	if opts.Target != "" {
		wt, found, ferr := s.findWorktree(opts.Target)
		if ferr != nil {
			return Result{}, ferr
		}
		if !found {
			return Result{}, fmt.Errorf("no worktree matching %q (see `gwt ls`)", opts.Target)
		}
		target = wt.Path
	} else {
		target, err = s.Repo.Root()
		if err != nil {
			return Result{}, fmt.Errorf("locate current worktree: %w", err)
		}
	}

	if sameDir(target, root) {
		return Result{}, errors.New("refusing to remove the main worktree")
	}

	// Capture the branch BEFORE removal: the worktree disappears afterwards.
	branch := s.branchForPath(target)

	// Safety checks (skipped under Force).
	if !opts.Force {
		st, serr := s.Repo.Status(target)
		if serr != nil {
			return Result{}, fmt.Errorf("status %s: %w", target, serr)
		}
		if st.Dirty {
			return Result{}, fmt.Errorf("worktree %s has uncommitted changes; commit/stash them or use -f", target)
		}
		if st.Unpushed() {
			ui.Warn("Worktree %s has unpushed commits (%d ahead of %s).", target, st.Ahead, st.Upstream)
			if !ui.Confirm("Remove anyway?", false) {
				return Result{}, errors.New("aborted")
			}
		}
	}

	// pre_remove hooks run in the target before it goes away.
	if s.Setup != nil {
		if err := s.Setup.RunHooks(context.Background(), setup.PreRemove, target, root); err != nil {
			ui.Warn("pre_remove hooks: %v", err)
		}
	}

	if err := s.Repo.Remove(target, opts.Force); err != nil {
		return Result{}, fmt.Errorf("remove worktree: %w", err)
	}
	ui.OK("Removed worktree %s", target)

	// Branch handling.
	switch {
	case (opts.DeleteBranch || opts.ForceDelete) && branch != "":
		if err := s.Repo.DeleteBranch(branch, opts.ForceDelete); err != nil {
			if opts.ForceDelete {
				ui.Warn("could not delete branch '%s': %v", branch, err)
			} else {
				ui.Warn("branch '%s' is not fully merged; not deleted. Use -D to force.", branch)
			}
		} else {
			ui.OK("Deleted branch '%s'", branch)
		}
	case branch != "":
		ui.Dim("Branch '%s' kept. Add -d (or -D to force) to delete it too next time.", branch)
	}

	return Result{Path: target, Branch: branch}, nil
}

// computeDest resolves the destination directory for a branch name without
// checking whether it already exists.
func (s *Service) computeDest(name, parentOverride string) (string, error) {
	root, err := s.Repo.MainWorktree()
	if err != nil {
		return "", fmt.Errorf("locate main worktree: %w", err)
	}

	parent := parentOverride
	if parent == "" {
		parent = s.Cfg.WorktreeDir
	}
	if parent == "" {
		parent = filepath.Dir(root)
	}

	// Resolve a relative parent against the current working directory.
	if !filepath.IsAbs(parent) {
		cwd, cerr := os.Getwd()
		if cerr != nil {
			return "", fmt.Errorf("resolve relative parent: %w", cerr)
		}
		parent = filepath.Join(cwd, parent)
	}

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create parent dir %s: %w", parent, err)
	}
	// Canonicalize (resolve symlinks, e.g. /tmp -> /private/tmp on macOS).
	if resolved, rerr := filepath.EvalSymlinks(parent); rerr == nil {
		parent = resolved
	}

	dirName := applyTemplate(s.Cfg.Naming, root, name)
	return filepath.Join(parent, dirName), nil
}

// ResolveDest computes the destination directory for a branch name, applying
// the parent-dir precedence and the naming template.
//
// Parent precedence (highest first): parentOverride > Cfg.WorktreeDir >
// filepath.Dir(MainWorktree()). A relative parent is resolved against the
// current working directory and created (MkdirAll) if missing, then
// canonicalized. The resolved destination must NOT already exist; callers rely
// on this to detect collisions.
func (s *Service) ResolveDest(name, parentOverride string) (string, error) {
	dest, err := s.computeDest(name, parentOverride)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("destination already exists: %s", dest)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat destination %s: %w", dest, err)
	}

	return dest, nil
}

// CleanMerged removes worktrees whose branch is merged into the default branch.
// When dryRun is true it reports candidates without removing anything.
// deleteBranch and forceDelete control branch deletion; when both are false,
// merged branches are still deleted (historical behavior for --merged).
func (s *Service) CleanMerged(dryRun, deleteBranch, forceDelete bool) ([]Result, error) {
	def, err := s.Repo.DefaultBranch()
	if err != nil {
		return nil, fmt.Errorf("determine default branch: %w", err)
	}

	wts, err := s.Repo.List()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	var candidates []git.Worktree
	for _, wt := range wts {
		if wt.IsMain || wt.Branch == "" || wt.Branch == def {
			continue
		}
		merged, merr := s.Repo.IsMerged(wt.Branch, def)
		if merr != nil {
			ui.Warn("could not check merge status of '%s': %v", wt.Branch, merr)
			continue
		}
		if merged {
			candidates = append(candidates, wt)
		}
	}

	if len(candidates) == 0 {
		ui.Info("No merged worktrees to clean.")
		return nil, nil
	}

	if dryRun {
		ui.Info("Worktrees merged into '%s' (dry run, nothing removed):", def)
		results := make([]Result, 0, len(candidates))
		for _, wt := range candidates {
			ui.Dim("  %s  (%s)", wt.Path, wt.Branch)
			results = append(results, Result{Path: wt.Path, Branch: wt.Branch})
		}
		return results, nil
	}

	root, err := s.Repo.MainWorktree()
	if err != nil {
		return nil, fmt.Errorf("locate main worktree: %w", err)
	}

	ctx := context.Background()
	results := make([]Result, 0, len(candidates))
	for _, wt := range candidates {
		if s.Setup != nil {
			if err := s.Setup.RunHooks(ctx, setup.PreRemove, wt.Path, root); err != nil {
				ui.Warn("pre_remove hooks: %v", err)
			}
		}
		if err := s.Repo.Remove(wt.Path, false); err != nil {
			ui.Warn("could not remove %s: %v", wt.Path, err)
			continue
		}
		del, force := deleteBranch, forceDelete
		if !del && !force {
			del = true // --merged: delete branch by default
		}
		if del || force {
			if err := s.Repo.DeleteBranch(wt.Branch, force); err != nil {
				ui.Warn("removed %s but could not delete branch '%s': %v", wt.Path, wt.Branch, err)
			}
		}
		ui.OK("Removed merged worktree %s (%s)", wt.Path, wt.Branch)
		results = append(results, Result{Path: wt.Path, Branch: wt.Branch})
	}
	return results, nil
}

// findWorktree returns the worktree whose branch equals name (case-insensitively)
// or whose directory basename equals name. found is false when no worktree matches.
func (s *Service) findWorktree(name string) (git.Worktree, bool, error) {
	wts, err := s.Repo.List()
	if err != nil {
		return git.Worktree{}, false, fmt.Errorf("list worktrees: %w", err)
	}
	for _, wt := range wts {
		if strings.EqualFold(wt.Branch, name) ||
			strings.EqualFold(filepath.Base(wt.Path), name) ||
			sameDir(wt.Path, name) {
			return wt, true, nil
		}
	}
	return git.Worktree{}, false, nil
}

// findWorktreeByPath returns the registered worktree at dest, comparing paths
// case-insensitively where the filesystem does.
func (s *Service) findWorktreeByPath(dest string) (git.Worktree, bool, error) {
	wts, err := s.Repo.List()
	if err != nil {
		return git.Worktree{}, false, fmt.Errorf("list worktrees: %w", err)
	}
	for _, wt := range wts {
		if sameDir(wt.Path, dest) {
			return wt, true, nil
		}
	}
	return git.Worktree{}, false, nil
}

// branchForPath returns the branch of the worktree at path, or "" when not
// found or detached. Failures are treated as "no branch" rather than fatal.
func (s *Service) branchForPath(path string) string {
	wts, err := s.Repo.List()
	if err != nil {
		return ""
	}
	for _, wt := range wts {
		if sameDir(wt.Path, path) {
			return wt.Branch
		}
	}
	return ""
}

// OpenEditor launches the configured editor in dir, detached so it does not
// block (works best with GUI editors like code/cursor). The editor is resolved
// from Cfg.Editor, then $EDITOR, then $VISUAL. Returns an error if none is set
// or the process fails to start.
func (s *Service) OpenEditor(dir string) error {
	editor := s.Cfg.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return errors.New("no editor configured (set 'editor' in config or $EDITOR)")
	}
	fields := strings.Fields(editor)
	c := osexec.Command(fields[0], append(fields[1:], ".")...)
	c.Dir = dir
	// Detached: nil std streams connect to the null device so the editor does
	// not draw over the TUI; Start returns immediately.
	if err := c.Start(); err != nil {
		return fmt.Errorf("open editor %q: %w", editor, err)
	}
	return nil
}

// openEditor is the create-flow wrapper: it opens the editor and reports any
// problem without failing the create.
func (s *Service) openEditor(_ context.Context, dir string) {
	if err := s.OpenEditor(dir); err != nil {
		ui.Dim("skipping --open: %v", err)
	}
}

// openTmux opens a new tmux window rooted in dir, named after the branch.
func (s *Service) openTmux(ctx context.Context, dir, name string) {
	if s.Run == nil {
		return
	}
	window := branchDir(name)
	if _, _, err := s.Run.Run(ctx, dir, "tmux", "new-window", "-c", dir, "-n", window); err != nil {
		ui.Warn("could not open tmux window: %v", err)
	}
}

// sameDir reports whether two paths refer to the same directory after cleaning
// and resolving symlinks where possible.
func sameDir(a, b string) bool {
	if a == b {
		return true
	}
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if ca == cb {
		return true
	}
	if ra, err := filepath.EvalSymlinks(ca); err == nil {
		ca = ra
	}
	if rb, err := filepath.EvalSymlinks(cb); err == nil {
		cb = rb
	}
	return ca == cb
}
