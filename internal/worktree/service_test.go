package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
)

// --- fakes ---------------------------------------------------------------

type fakeRepo struct {
	main      string
	root      string // current worktree (Repo.Root); defaults to main
	worktrees []git.Worktree
	statuses  map[string]git.Status
	merged    map[string]bool // branch -> merged into default
	defBranch string

	addCalls    []git.AddOpts
	removeCalls []string
	delBranch   []delCall
}

type delCall struct {
	name  string
	force bool
}

func (f *fakeRepo) Root() (string, error) {
	if f.root != "" {
		return f.root, nil
	}
	return f.main, nil
}
func (f *fakeRepo) MainWorktree() (string, error) { return f.main, nil }
func (f *fakeRepo) List() ([]git.Worktree, error) { return f.worktrees, nil }
func (f *fakeRepo) Add(opts git.AddOpts) error {
	f.addCalls = append(f.addCalls, opts)
	return nil
}
func (f *fakeRepo) Remove(path string, force bool) error {
	f.removeCalls = append(f.removeCalls, path)
	return nil
}
func (f *fakeRepo) Prune() error { return nil }
func (f *fakeRepo) Status(path string) (git.Status, error) {
	if f.statuses == nil {
		return git.Status{}, nil
	}
	return f.statuses[path], nil
}
func (f *fakeRepo) BranchExists(name string) (bool, error)   { return true, nil }
func (f *fakeRepo) BranchStates() (map[string]string, error) { return nil, nil }
func (f *fakeRepo) DeleteBranch(name string, force bool) error {
	f.delBranch = append(f.delBranch, delCall{name, force})
	return nil
}
func (f *fakeRepo) IsMerged(branch, into string) (bool, error) {
	return f.merged[branch], nil
}
func (f *fakeRepo) DefaultBranch() (string, error) {
	if f.defBranch != "" {
		return f.defBranch, nil
	}
	return "main", nil
}
func (f *fakeRepo) DiskUsage(path string) (int64, error) { return 0, nil }

var _ git.Repo = (*fakeRepo)(nil)

type fakeGH struct{}

func (fakeGH) Available() bool                      { return false }
func (fakeGH) ListPRs() ([]gh.PR, error)            { return nil, nil }
func (fakeGH) Checkout(pr int) (string, error)      { return "", nil }
func (fakeGH) Checks(b string) (gh.CIStatus, error) { return gh.CIStatus{}, nil }

var _ gh.Client = fakeGH{}

func newService(t *testing.T, repo *fakeRepo, cfg config.Config) *Service {
	t.Helper()
	return New(repo, fakeGH{}, cfg, &exec.Fake{Default: &exec.FakeResult{}})
}

// --- ResolveDest ---------------------------------------------------------

func TestResolveDest_DefaultParentAndNaming(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "myrepo")
	repo := &fakeRepo{main: main}
	svc := newService(t, repo, config.Defaults())

	dest, err := svc.ResolveDest("feature/foo", "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "myrepo-feature-foo")
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		want = filepath.Join(real, "myrepo-feature-foo")
	}
	if dest != want {
		t.Fatalf("dest = %q, want %q", dest, want)
	}
}

func TestResolveDest_Precedence(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main", "myrepo")
	cfgDir := filepath.Join(tmp, "cfgparent")
	overrideDir := filepath.Join(tmp, "override")

	cfg := config.Defaults()
	cfg.WorktreeDir = cfgDir
	repo := &fakeRepo{main: main}
	svc := newService(t, repo, cfg)

	// Override beats config.
	dest, err := svc.ResolveDest("x", overrideDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Dir(dest); !sameDir(got, overrideDir) {
		t.Fatalf("override parent = %q, want %q", got, overrideDir)
	}

	// Config beats default (no override).
	dest, err = svc.ResolveDest("y", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Dir(dest); !sameDir(got, cfgDir) {
		t.Fatalf("config parent = %q, want %q", got, cfgDir)
	}
}

func TestResolveDest_AlreadyExistsErrors(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "myrepo")
	repo := &fakeRepo{main: main}
	svc := newService(t, repo, config.Defaults())

	dest, err := svc.ResolveDest("dup", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveDest("dup", ""); err == nil {
		t.Fatal("expected error for existing destination")
	}
}

func TestResolveDest_CustomNaming(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	cfg := config.Defaults()
	cfg.Naming = "{branch_slug}"
	repo := &fakeRepo{main: main}
	svc := newService(t, repo, cfg)

	dest, err := svc.ResolveDest("Feature/Big Thing", "")
	if err != nil {
		t.Fatal(err)
	}
	if base := filepath.Base(dest); base != "feature-big-thing" {
		t.Fatalf("base = %q, want feature-big-thing", base)
	}
}

// --- Create --------------------------------------------------------------

func TestCreate_CallsAddWithOpts(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	repo := &fakeRepo{main: main}
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Create(CreateOpts{Name: "feat", Base: "main", NewBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.addCalls) != 1 {
		t.Fatalf("expected 1 Add call, got %d", len(repo.addCalls))
	}
	got := repo.addCalls[0]
	if !got.NewBranch || got.Branch != "feat" || got.Base != "main" {
		t.Fatalf("AddOpts = %+v", got)
	}
	if got.Path != res.Path {
		t.Fatalf("Add path %q != result path %q", got.Path, res.Path)
	}
	if res.Branch != "feat" {
		t.Fatalf("result branch = %q", res.Branch)
	}
	if filepath.Base(res.Path) != "repo-feat" {
		t.Fatalf("result path base = %q", filepath.Base(res.Path))
	}
}

func TestCreate_RunsHooksViaRunner(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	repo := &fakeRepo{main: main}
	cfg := config.Defaults()
	cfg.Hooks.PostCreate = []string{"echo hi"}
	fake := &exec.Fake{Default: &exec.FakeResult{}}
	svc := New(repo, fakeGH{}, cfg, fake)

	if _, err := svc.Create(CreateOpts{Name: "h", NewBranch: true}); err != nil {
		t.Fatal(err)
	}
	// The post_create hook should have run via `sh -c "echo hi"`.
	var ran bool
	for _, c := range fake.Calls {
		if strings.Contains(c, "sh -c echo hi") {
			ran = true
		}
	}
	if !ran {
		t.Fatalf("post_create hook not run; calls=%v", fake.Calls)
	}
}

// --- Switch --------------------------------------------------------------

func TestSwitch_ReturnsExisting(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	existing := filepath.Join(tmp, "repo-feat")
	repo := &fakeRepo{
		main: main,
		worktrees: []git.Worktree{
			{Path: main, Branch: "main", IsMain: true},
			{Path: existing, Branch: "feat"},
		},
	}
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Switch("feat", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != existing {
		t.Fatalf("path = %q, want %q", res.Path, existing)
	}
	if len(repo.addCalls) != 0 {
		t.Fatalf("Switch should not create when worktree exists; addCalls=%d", len(repo.addCalls))
	}
}

func TestSwitch_MatchesByDirBasename(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	existing := filepath.Join(tmp, "repo-feat")
	repo := &fakeRepo{
		main: main,
		worktrees: []git.Worktree{
			{Path: main, Branch: "main", IsMain: true},
			{Path: existing, Branch: "feature/foo"},
		},
	}
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Switch("repo-feat", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != existing {
		t.Fatalf("path = %q, want %q", res.Path, existing)
	}
}

func TestSwitch_CreatesWhenMissing(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "repo")
	repo := &fakeRepo{
		main:      main,
		worktrees: []git.Worktree{{Path: main, Branch: "main", IsMain: true}},
	}
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Switch("other", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.addCalls) != 1 {
		t.Fatalf("expected create; addCalls=%d", len(repo.addCalls))
	}
	if repo.addCalls[0].NewBranch {
		t.Fatal("Switch create must use existing branch (NewBranch=false)")
	}
	if repo.addCalls[0].Branch != "other" {
		t.Fatalf("branch = %q", repo.addCalls[0].Branch)
	}
	if filepath.Base(res.Path) != "repo-other" {
		t.Fatalf("path base = %q", filepath.Base(res.Path))
	}
}

// --- Remove --------------------------------------------------------------

func baseRemoveRepo(tmp string) *fakeRepo {
	main := filepath.Join(tmp, "repo")
	target := filepath.Join(tmp, "repo-feat")
	return &fakeRepo{
		main: main,
		root: target, // current worktree is the target
		worktrees: []git.Worktree{
			{Path: main, Branch: "main", IsMain: true},
			{Path: target, Branch: "feat"},
		},
	}
}

func TestRemove_DirtyRefusesWithoutForce(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	target := repo.root
	repo.statuses = map[string]git.Status{target: {Dirty: true}}
	svc := newService(t, repo, config.Defaults())

	if _, err := svc.Remove(RemoveOpts{Target: "feat"}); err == nil {
		t.Fatal("expected refusal for dirty worktree")
	}
	if len(repo.removeCalls) != 0 {
		t.Fatal("must not remove a dirty worktree without force")
	}
}

func TestRemove_ForceRemovesDirty(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	repo.statuses = map[string]git.Status{repo.root: {Dirty: true}}
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Remove(RemoveOpts{Target: "feat", Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.removeCalls) != 1 {
		t.Fatalf("expected 1 remove, got %d", len(repo.removeCalls))
	}
	if res.Branch != "feat" {
		t.Fatalf("captured branch = %q", res.Branch)
	}
}

func TestRemove_RefusesMainWorktree(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	repo.root = repo.main // standing in main, no target -> resolves to main
	svc := newService(t, repo, config.Defaults())

	if _, err := svc.Remove(RemoveOpts{}); err == nil || !strings.Contains(err.Error(), "main worktree") {
		t.Fatalf("expected main-worktree refusal, got %v", err)
	}
	if len(repo.removeCalls) != 0 {
		t.Fatal("must not remove main worktree")
	}
}

func TestRemove_NoMatchErrors(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	_, err := svc.Remove(RemoveOpts{Target: "nope"})
	if err == nil || !strings.Contains(err.Error(), "gwt ls") {
		t.Fatalf("expected no-match error mentioning `gwt ls`, got %v", err)
	}
}

func TestRemove_DeletesBranchAndCaptures(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	res, err := svc.Remove(RemoveOpts{Target: "feat", DeleteBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.delBranch) != 1 || repo.delBranch[0].name != "feat" || repo.delBranch[0].force {
		t.Fatalf("DeleteBranch calls = %+v", repo.delBranch)
	}
	if res.Branch != "feat" {
		t.Fatalf("branch = %q", res.Branch)
	}
}

func TestRemove_ForceDeleteUsesForce(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	if _, err := svc.Remove(RemoveOpts{Target: "feat", ForceDelete: true}); err != nil {
		t.Fatal(err)
	}
	if len(repo.delBranch) != 1 || !repo.delBranch[0].force {
		t.Fatalf("expected forced branch delete, got %+v", repo.delBranch)
	}
}

func TestRemove_KeepsBranchWhenNoFlag(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := baseRemoveRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	if _, err := svc.Remove(RemoveOpts{Target: "feat"}); err != nil {
		t.Fatal(err)
	}
	if len(repo.delBranch) != 0 {
		t.Fatal("must not delete branch when neither -d nor -D given")
	}
}

// --- CleanMerged ---------------------------------------------------------

func cleanRepo(tmp string) *fakeRepo {
	main := filepath.Join(tmp, "repo")
	return &fakeRepo{
		main:      main,
		defBranch: "main",
		worktrees: []git.Worktree{
			{Path: main, Branch: "main", IsMain: true},
			{Path: filepath.Join(tmp, "repo-merged"), Branch: "merged"},
			{Path: filepath.Join(tmp, "repo-active"), Branch: "active"},
			{Path: filepath.Join(tmp, "repo-detached"), Branch: ""},
		},
		merged: map[string]bool{"merged": true, "active": false},
	}
}

func TestCleanMerged_DryRunSelectsCandidates(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := cleanRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	results, err := svc.CleanMerged(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Branch != "merged" {
		t.Fatalf("dry-run candidates = %+v", results)
	}
	if len(repo.removeCalls) != 0 {
		t.Fatal("dry-run must not remove anything")
	}
}

func TestCleanMerged_RemovesAndDeletesBranch(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	repo := cleanRepo(tmp)
	svc := newService(t, repo, config.Defaults())

	results, err := svc.CleanMerged(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Branch != "merged" {
		t.Fatalf("results = %+v", results)
	}
	if len(repo.removeCalls) != 1 {
		t.Fatalf("expected 1 remove, got %v", repo.removeCalls)
	}
	if len(repo.delBranch) != 1 || repo.delBranch[0].name != "merged" {
		t.Fatalf("branch deletes = %+v", repo.delBranch)
	}
}
