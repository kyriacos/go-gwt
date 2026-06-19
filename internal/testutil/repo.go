// Package testutil builds real git repositories in temporary directories for
// integration tests of the git layer. It shells out to the git CLI directly
// (not through the exec.Runner) since its job is to construct fixtures, not to
// be the thing under test.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a real git repository rooted in a temporary directory, with helpers
// to evolve its history for tests.
type Repo struct {
	t   *testing.T
	Dir string // path to the main worktree
}

// NewRepo initializes a fresh git repository in t.TempDir() with a deterministic
// identity, an initial branch named "main", and one initial commit. It is
// parallel-safe: each call gets its own temp directory and uses local config
// only, never touching the caller's global git configuration.
func NewRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	r := &Repo{t: t, Dir: dir}
	r.git("init", "-b", "main")
	r.git("config", "user.name", "Test User")
	r.git("config", "user.email", "test@example.com")
	r.git("config", "commit.gpgsign", "false")
	r.WriteFile("README.md", "init\n")
	r.git("add", "README.md")
	r.git("commit", "-m", "initial commit")
	return r
}

// Git runs an arbitrary git command in the repo's main worktree, failing the
// test on error, and returns the combined output. It is exposed so tests can
// reach for git operations the typed helpers do not cover.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	return r.gitIn(r.Dir, args...)
}

// git runs a git command in the repo's main worktree and fails the test on
// error.
func (r *Repo) git(args ...string) string {
	r.t.Helper()
	return r.gitIn(r.Dir, args...)
}

// gitIn runs a git command with its working directory set to dir.
func (r *Repo) gitIn(dir string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Isolate from the developer's environment so commits are reproducible and
	// no global hooks/config interfere.
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// WriteFile writes (or overwrites) a file at the given path relative to the main
// worktree, creating parent directories as needed.
func (r *Repo) WriteFile(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.Dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatalf("write %s: %v", rel, err)
	}
}

// Commit writes a file with the given content and records a commit with msg on
// the current branch. It returns the new commit's short SHA.
func (r *Repo) Commit(msg, file, content string) string {
	r.t.Helper()
	r.WriteFile(file, content)
	r.git("add", file)
	r.git("commit", "-m", msg)
	return r.Head()
}

// Head returns the short SHA of the current HEAD.
func (r *Repo) Head() string {
	r.t.Helper()
	return r.git("rev-parse", "--short=7", "HEAD")
}

// CreateBranch creates a new branch from the current HEAD without switching to
// it.
func (r *Repo) CreateBranch(name string) {
	r.t.Helper()
	r.git("branch", name)
}

// Checkout switches the main worktree to the given branch.
func (r *Repo) Checkout(name string) {
	r.t.Helper()
	r.git("checkout", name)
}

// AddWorktree creates a worktree for an existing branch at the given absolute
// path.
func (r *Repo) AddWorktree(path, branch string) {
	r.t.Helper()
	r.git("worktree", "add", path, branch)
}

// AddWorktreeNewBranch creates a worktree with a new branch at the given path.
func (r *Repo) AddWorktreeNewBranch(path, branch string) {
	r.t.Helper()
	r.git("worktree", "add", "-b", branch, path)
}

// Path joins one or more elements onto the main worktree directory.
func (r *Repo) Path(elem ...string) string {
	return filepath.Join(append([]string{r.Dir}, elem...)...)
}
