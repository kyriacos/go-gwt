package git

import (
	"reflect"
	"testing"

	"github.com/kyriacos/go-gwt/internal/exec"
)

func TestParseWorktreeList(t *testing.T) {
	t.Parallel()
	const fixture = `worktree /home/u/proj
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /home/u/proj-feature
HEAD 2222222222222222222222222222222222222222
branch refs/heads/feature/x

worktree /home/u/proj-detached
HEAD 3333333333333333333333333333333333333333
detached

worktree /home/u/bare
bare
`
	got := parseWorktreeList(fixture)
	want := []Worktree{
		{Path: "/home/u/proj", Head: "1111111", Branch: "main", IsMain: true},
		{Path: "/home/u/proj-feature", Head: "2222222", Branch: "feature/x"},
		{Path: "/home/u/proj-detached", Head: "3333333", Detached: true},
		{Path: "/home/u/bare", Bare: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseWorktreeList mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParseWorktreeListEmpty(t *testing.T) {
	t.Parallel()
	if got := parseWorktreeList(""); len(got) != 0 {
		t.Fatalf("expected no worktrees, got %+v", got)
	}
}

func TestParseStatusV2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want Status
	}{
		{
			name: "clean with upstream",
			in: `# branch.oid 1111111
# branch.head main
# branch.upstream origin/main
# branch.ab +0 -0
`,
			want: Status{Upstream: "origin/main"},
		},
		{
			name: "ahead and behind",
			in: `# branch.head main
# branch.upstream origin/main
# branch.ab +2 -3
`,
			want: Status{Upstream: "origin/main", Ahead: 2, Behind: 3},
		},
		{
			name: "mixed changes no upstream",
			in: `# branch.head feature
1 M. N... 100644 100644 100644 aaa bbb staged.go
1 .M N... 100644 100644 100644 ccc ddd unstaged.go
1 MM N... 100644 100644 100644 eee fff both.go
? untracked.txt
`,
			want: Status{Dirty: true, Staged: 2, Unstaged: 2, Untracked: 1},
		},
		{
			name: "renamed entry counts as staged",
			in: `# branch.head main
2 R. N... 100644 100644 100644 aaa bbb R100 new.go	old.go
`,
			want: Status{Dirty: true, Staged: 1},
		},
		{
			name: "unmerged entry",
			in: `# branch.head main
u UU N... 100644 100644 100644 100644 aaa bbb ccc conflict.go
`,
			want: Status{Dirty: true, Unstaged: 1},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseStatusV2(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseStatusV2(%q)\n got: %+v\nwant: %+v", tt.name, got, tt.want)
			}
		})
	}
}

// TestListUsesPorcelain exercises List through the Fake runner to confirm the
// exact command line and that parsing is wired in.
func TestListUsesPorcelain(t *testing.T) {
	t.Parallel()
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git worktree list --porcelain": {Stdout: "worktree /a\nHEAD abcdef0123456789\nbranch refs/heads/main\n"},
	}}
	repo := New(f)
	wts, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 || wts[0].Branch != "main" || wts[0].Head != "abcdef0" || !wts[0].IsMain {
		t.Fatalf("unexpected worktrees: %+v", wts)
	}
}

func TestMainWorktree(t *testing.T) {
	t.Parallel()
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git worktree list --porcelain": {Stdout: "worktree /main\nbranch refs/heads/main\n\nworktree /other\nbranch refs/heads/x\n"},
	}}
	got, err := New(f).MainWorktree()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/main" {
		t.Fatalf("MainWorktree = %q, want /main", got)
	}
}

func TestAddCommandLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    AddOpts
		wantKey string
	}{
		{"new branch default base", AddOpts{Path: "/w", Branch: "feat", NewBranch: true}, "git worktree add -b feat /w HEAD"},
		{"new branch with base", AddOpts{Path: "/w", Branch: "feat", NewBranch: true, Base: "main"}, "git worktree add -b feat /w main"},
		{"existing branch", AddOpts{Path: "/w", Branch: "feat"}, "git worktree add /w feat"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &exec.Fake{Default: &exec.FakeResult{}}
			if err := New(f).Add(tt.opts); err != nil {
				t.Fatal(err)
			}
			if len(f.Calls) != 1 || f.Calls[0] != tt.wantKey {
				t.Fatalf("calls = %v, want [%q]", f.Calls, tt.wantKey)
			}
		})
	}
}

func TestRemoveAndPruneCommandLines(t *testing.T) {
	t.Parallel()
	f := &exec.Fake{Default: &exec.FakeResult{}}
	repo := New(f)
	if err := repo.Remove("/w", false); err != nil {
		t.Fatal(err)
	}
	if err := repo.Remove("/w2", true); err != nil {
		t.Fatal(err)
	}
	if err := repo.Prune(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"git worktree remove /w",
		"git worktree remove --force /w2",
		"git worktree prune",
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Fatalf("calls = %v, want %v", f.Calls, want)
	}
}

func TestBranchExistsFake(t *testing.T) {
	t.Parallel()
	exists := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git show-ref --verify --quiet refs/heads/foo": {},
	}, Default: &exec.FakeResult{Err: errExit{}}}
	ok, err := New(exists).BranchExists("foo")
	if err != nil || !ok {
		t.Fatalf("expected exists, got ok=%v err=%v", ok, err)
	}
	ok, err = New(exists).BranchExists("missing")
	if err != nil || ok {
		t.Fatalf("expected not-exists no-error, got ok=%v err=%v", ok, err)
	}
}

func TestIsMergedFake(t *testing.T) {
	t.Parallel()
	merged := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git merge-base --is-ancestor a b": {},
	}, Default: &exec.FakeResult{Err: errExit{}}}
	ok, err := New(merged).IsMerged("a", "b")
	if err != nil || !ok {
		t.Fatalf("expected merged, got ok=%v err=%v", ok, err)
	}
	ok, err = New(merged).IsMerged("x", "y")
	if err != nil || ok {
		t.Fatalf("expected not-merged no-error, got ok=%v err=%v", ok, err)
	}
}

func TestDefaultBranchFromOriginHead(t *testing.T) {
	t.Parallel()
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git symbolic-ref --quiet refs/remotes/origin/HEAD": {Stdout: "refs/remotes/origin/main\n"},
	}}
	got, err := New(f).DefaultBranch()
	if err != nil || got != "main" {
		t.Fatalf("DefaultBranch = %q err=%v, want main", got, err)
	}
}

func TestDefaultBranchFallback(t *testing.T) {
	t.Parallel()
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"git symbolic-ref --quiet refs/remotes/origin/HEAD": {Err: errExit{}},
		"git show-ref --verify --quiet refs/heads/main":     {Err: errExit{}},
		"git show-ref --verify --quiet refs/heads/master":   {},
	}}
	got, err := New(f).DefaultBranch()
	if err != nil || got != "master" {
		t.Fatalf("DefaultBranch = %q err=%v, want master", got, err)
	}
}

// errExit is a stand-in non-zero exit error for the Fake runner.
type errExit struct{}

func (errExit) Error() string { return "exit status 1" }
