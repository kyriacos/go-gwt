package git

import (
	"testing"

	xexec "github.com/kyriacos/go-gwt/internal/exec"
)

// TestListCache verifies that List and MainWorktree share one git subprocess
// and that mutating commands invalidate the cache.
func TestListCache(t *testing.T) {
	porcelain := "worktree /main\nHEAD abcdef0\nbranch refs/heads/main\n\nworktree /feat\nHEAD 1234567\nbranch refs/heads/feat\n"
	f := &xexec.Fake{
		Responses: map[string]xexec.FakeResult{
			xexec.Key("git", "worktree", "list", "--porcelain"):            {Stdout: porcelain},
			xexec.Key("git", "worktree", "add", "-b", "x", "/new", "HEAD"): {},
		},
	}
	r := New(f)

	if _, err := r.MainWorktree(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.List(); err != nil {
		t.Fatal(err)
	}
	listCalls := 0
	for _, c := range f.Calls {
		if c == xexec.Key("git", "worktree", "list", "--porcelain") {
			listCalls++
		}
	}
	if listCalls != 1 {
		t.Fatalf("expected 1 worktree list call, got %d", listCalls)
	}

	if err := r.Add(AddOpts{Path: "/new", Branch: "x", NewBranch: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.List(); err != nil {
		t.Fatal(err)
	}
	listCalls = 0
	for _, c := range f.Calls {
		if c == xexec.Key("git", "worktree", "list", "--porcelain") {
			listCalls++
		}
	}
	if listCalls != 2 {
		t.Fatalf("after Add invalidate, expected 2 list calls, got %d", listCalls)
	}
}
