package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	xexec "github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/testutil"
)

func init() {
	// All integration tests require the git binary.
	if _, err := exec.LookPath("git"); err != nil {
		gitMissing = true
	}
}

var gitMissing bool

// repoAt returns a CmdRepo whose git commands run in dir, backed by the real
// runner.
func repoAt(dir string) *CmdRepo {
	r := New(xexec.New())
	return r.WithContext(context.Background()).inDir(dir)
}

// inDir is a test-only convenience: the production CmdRepo runs git with an
// empty dir (current process cwd) for repo-wide commands. For integration tests
// we need those commands to target the temp repo, so we wrap the runner to
// inject dir whenever the caller passes "".
func (r *CmdRepo) inDir(dir string) *CmdRepo {
	c := *r
	c.run = dirRunner{inner: r.run, dir: dir}
	return &c
}

// dirRunner forwards to inner but substitutes its dir whenever the requested
// dir is empty. This lets integration tests scope repo-wide git commands to a
// temp repository without changing the process working directory (keeping tests
// parallel-safe).
type dirRunner struct {
	inner xexec.Runner
	dir   string
}

func (d dirRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	if dir == "" {
		dir = d.dir
	}
	return d.inner.Run(ctx, dir, name, args...)
}

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if gitMissing {
		t.Skip("git not available")
	}
}

func TestIntegrationListAndAddRemove(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	// Initially one worktree (main).
	wts, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 || !wts[0].IsMain || wts[0].Branch != "main" {
		t.Fatalf("unexpected initial worktrees: %+v", wts)
	}

	// Add a worktree with a new branch.
	wtPath := tr.Path("..", "wt-feature")
	if err := repo.Add(AddOpts{Path: wtPath, Branch: "feature", NewBranch: true}); err != nil {
		t.Fatal(err)
	}
	wts, err = repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %+v", wts)
	}
	var found bool
	for _, w := range wts {
		if w.Branch == "feature" {
			found = true
		}
	}
	if !found {
		t.Fatalf("feature worktree not listed: %+v", wts)
	}

	// Remove it.
	if err := repo.Remove(wtPath, false); err != nil {
		t.Fatal(err)
	}
	wts, err = repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree after remove, got %+v", wts)
	}
}

func TestIntegrationRootAndMainWorktree(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	root, err := repo.Root()
	if err != nil {
		t.Fatal(err)
	}
	main, err := repo.MainWorktree()
	if err != nil {
		t.Fatal(err)
	}
	if root != main {
		t.Fatalf("root %q != main worktree %q", root, main)
	}
}

func TestIntegrationStatus(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	// Clean tree, no upstream.
	st, err := repo.Status(tr.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Dirty || st.Upstream != "" || st.Ahead != 0 || st.Behind != 0 {
		t.Fatalf("expected clean no-upstream status, got %+v", st)
	}

	// Introduce a staged and an untracked change.
	tr.WriteFile("staged.go", "package x\n")
	tr.Git("add", "staged.go")
	tr.WriteFile("untracked.txt", "hi\n")
	st, err = repo.Status(tr.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Dirty || st.Staged != 1 || st.Untracked != 1 {
		t.Fatalf("expected staged=1 untracked=1 dirty, got %+v", st)
	}
}

func TestIntegrationBranchOps(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	ok, err := repo.BranchExists("main")
	if err != nil || !ok {
		t.Fatalf("main should exist: ok=%v err=%v", ok, err)
	}
	ok, err = repo.BranchExists("nope")
	if err != nil || ok {
		t.Fatalf("nope should not exist: ok=%v err=%v", ok, err)
	}

	tr.CreateBranch("topic")
	if err := repo.DeleteBranch("topic", false); err != nil {
		t.Fatalf("delete merged branch: %v", err)
	}
	ok, _ = repo.BranchExists("topic")
	if ok {
		t.Fatalf("topic should be deleted")
	}
}

func TestIntegrationIsMerged(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	// topic branched at main, not advanced -> merged into main.
	tr.CreateBranch("topic")
	merged, err := repo.IsMerged("topic", "main")
	if err != nil || !merged {
		t.Fatalf("unadvanced topic should be merged into main: %v %v", merged, err)
	}

	// Advance topic with a new commit -> no longer an ancestor of main.
	tr.Checkout("topic")
	tr.Commit("topic work", "t.txt", "t\n")
	tr.Checkout("main")
	merged, err = repo.IsMerged("topic", "main")
	if err != nil {
		t.Fatal(err)
	}
	if merged {
		t.Fatalf("advanced topic should not be merged into main")
	}
}

func TestIntegrationDefaultBranchFallback(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	// No origin remote configured; should fall back to main.
	got, err := repo.DefaultBranch()
	if err != nil {
		t.Fatal(err)
	}
	if got != "main" {
		t.Fatalf("DefaultBranch = %q, want main", got)
	}
}

func TestIntegrationDiskUsage(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	tr.WriteFile("data.bin", "0123456789")
	size, err := repo.DiskUsage(tr.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if size < 10 {
		t.Fatalf("disk usage %d should be at least 10 bytes", size)
	}
}

func TestIntegrationPrune(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)
	// Prune on a clean repo is a no-op but must not error.
	if err := repo.Prune(); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationUpstreamAlignment(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	bare := filepath.Join(t.TempDir(), "origin.git")
	tr.Git("init", "--bare", "-b", "main", bare)
	tr.Git("remote", "add", "origin", bare)
	tr.Git("push", "-u", "origin", "main")

	tr.CreateBranch("feature")
	tr.Git("config", "branch.feature.remote", "origin")
	tr.Git("config", "branch.feature.merge", "refs/heads/main")

	remote, branch, ok, err := repo.BranchUpstream("feature")
	if err != nil || !ok || remote != "origin" || branch != "main" {
		t.Fatalf("BranchUpstream = %q %q %v err=%v", remote, branch, ok, err)
	}

	if err := repo.UnsetUpstream("feature"); err != nil {
		t.Fatalf("UnsetUpstream: %v", err)
	}
	_, _, ok, err = repo.BranchUpstream("feature")
	if err != nil || ok {
		t.Fatalf("expected no upstream after unset, ok=%v err=%v", ok, err)
	}

	tr.Git("push", "origin", "feature:feature")
	if err := repo.SetUpstream("feature", "origin", "feature"); err != nil {
		t.Fatalf("SetUpstream: %v", err)
	}
	remote, branch, ok, err = repo.BranchUpstream("feature")
	if err != nil || !ok || remote != "origin" || branch != "feature" {
		t.Fatalf("BranchUpstream after set = %q %q %v err=%v", remote, branch, ok, err)
	}
}
