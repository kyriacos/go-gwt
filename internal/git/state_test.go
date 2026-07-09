package git

import (
	"path/filepath"
	"testing"

	"github.com/kyriacos/go-gwt/internal/testutil"
)

func TestIntegrationBranchStatesUnpushedNotGone(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	bare := filepath.Join(t.TempDir(), "origin.git")
	tr.Git("init", "--bare", "-b", "main", bare)
	tr.Git("remote", "add", "origin", bare)
	tr.Git("push", "-u", "origin", "main")

	tr.CreateBranch("feature")

	states, err := repo.BranchStates()
	if err != nil {
		t.Fatalf("BranchStates: %v", err)
	}
	if got := states["feature"]; got != StateLocal {
		t.Fatalf("feature state = %q, want %q (never-pushed must not be gone)", got, StateLocal)
	}
}

func TestIntegrationBranchStatesRemoteDeletedIsGone(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()
	tr := testutil.NewRepo(t)
	repo := repoAt(tr.Dir)

	bare := filepath.Join(t.TempDir(), "origin.git")
	tr.Git("init", "--bare", "-b", "main", bare)
	tr.Git("remote", "add", "origin", bare)
	tr.Git("push", "-u", "origin", "main")

	tr.CreateBranch("feature")
	tr.Git("push", "-u", "origin", "feature")
	tr.Git("push", "origin", "--delete", "feature")
	tr.Git("fetch", "--prune")

	states, err := repo.BranchStates()
	if err != nil {
		t.Fatalf("BranchStates: %v", err)
	}
	if got := states["feature"]; got != StateGone {
		t.Fatalf("feature state = %q, want %q (deleted remote must be gone)", got, StateGone)
	}
}
