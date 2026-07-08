package git

import (
	"path/filepath"
	"testing"

	"github.com/kyriacos/go-gwt/internal/testutil"
)

func TestParseUpstreamRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in             string
		remote, branch string
		ok             bool
	}{
		{"refs/remotes/origin/feature", "origin", "feature", true},
		{"refs/remotes/upstream/my-branch", "upstream", "my-branch", true},
		{"refs/heads/feature", "", "", false},
		{"", "", "", false},
	}
	for _, tc := range tests {
		remote, branch, ok := parseUpstreamRef(tc.in)
		if remote != tc.remote || branch != tc.branch || ok != tc.ok {
			t.Errorf("parseUpstreamRef(%q) = %q %q %v, want %q %q %v",
				tc.in, remote, branch, ok, tc.remote, tc.branch, tc.ok)
		}
	}
}

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
	if err := repo.SetUpstream("feature", "origin", "feature"); err != nil {
		t.Fatalf("SetUpstream: %v", err)
	}

	states, err := repo.BranchStates()
	if err != nil {
		t.Fatalf("BranchStates: %v", err)
	}
	if got := states["feature"]; got != StateLocal {
		t.Fatalf("feature state = %q, want %q (never-pushed must not be gone)", got, StateLocal)
	}
}
