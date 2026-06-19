package tui

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
)

// ---- fakes ----------------------------------------------------------------

type fakeRepo struct {
	wts    []git.Worktree
	status map[string]git.Status
	sizes  map[string]int64
}

func (f *fakeRepo) Root() (string, error)         { return "/repo", nil }
func (f *fakeRepo) MainWorktree() (string, error) { return "/repo", nil }
func (f *fakeRepo) List() ([]git.Worktree, error) { return f.wts, nil }
func (f *fakeRepo) Add(git.AddOpts) error         { return nil }
func (f *fakeRepo) Remove(string, bool) error     { return nil }
func (f *fakeRepo) Prune() error                  { return nil }
func (f *fakeRepo) Status(p string) (git.Status, error) {
	if s, ok := f.status[p]; ok {
		return s, nil
	}
	return git.Status{}, nil
}
func (f *fakeRepo) BranchExists(string) (bool, error)        { return true, nil }
func (f *fakeRepo) BranchStates() (map[string]string, error) { return nil, nil }
func (f *fakeRepo) DeleteBranch(string, bool) error          { return nil }
func (f *fakeRepo) IsMerged(string, string) (bool, error)    { return false, nil }
func (f *fakeRepo) DefaultBranch() (string, error)           { return "main", nil }
func (f *fakeRepo) DiskUsage(p string) (int64, error)        { return f.sizes[p], nil }

type fakeGH struct {
	avail bool
	prs   []gh.PR
}

func (f *fakeGH) Available() bool              { return f.avail }
func (f *fakeGH) ListPRs() ([]gh.PR, error)    { return f.prs, nil }
func (f *fakeGH) Checkout(int) (string, error) { return "", nil }
func (f *fakeGH) Checks(string) (gh.CIStatus, error) {
	return gh.CIStatus{}, nil
}

type fakeActions struct {
	mu         sync.Mutex
	created    []string
	removed    []string
	removedDel []bool
	prCheckout []int
	opened     []string
}

func (a *fakeActions) Create(name string, newBranch bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.created = append(a.created, name)
	return "/repo-" + name, nil
}
func (a *fakeActions) Remove(target string, deleteBranch, force bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.removed = append(a.removed, target)
	a.removedDel = append(a.removedDel, deleteBranch)
	return nil
}
func (a *fakeActions) CleanMerged(bool) ([]string, error) { return nil, nil }
func (a *fakeActions) Open(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.opened = append(a.opened, path)
	return nil
}
func (a *fakeActions) CheckoutPR(n int) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prCheckout = append(a.prCheckout, n)
	return "/repo-pr", nil
}

// ---- helpers --------------------------------------------------------------

func sampleWorktrees() []git.Worktree {
	return []git.Worktree{
		{Path: "/repo", Branch: "main", Head: "aaaaaaa1", IsMain: true},
		{Path: "/repo-feature", Branch: "feature", Head: "bbbbbbb2"},
		{Path: "/repo-bugfix", Branch: "bugfix", Head: "ccccccc3"},
	}
}

func newTestModel(ghAvail bool) (*model, *fakeActions) {
	repo := &fakeRepo{
		wts:    sampleWorktrees(),
		status: map[string]git.Status{},
		sizes:  map[string]int64{},
	}
	acts := &fakeActions{}
	m := newModel(repo, &fakeGH{avail: ghAvail}, acts, func(p string) (string, error) {
		return "log for " + p, nil
	})
	return m, acts
}

// rune key
func rk(s string) KeyMsg { return KeyMsg{Type: keyRunes, Runes: []rune(s)} }

// load worktrees + statuses into the model synchronously.
func (m *model) seed(t *testing.T) {
	t.Helper()
	next, _ := m.Update(worktreesMsg{wts: m.repo.(*fakeRepo).wts})
	*m = *next.(*model)
}

// ---- tests ----------------------------------------------------------------

func TestStatusMessagesPopulateRows(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)

	if len(m.rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(m.rows))
	}
	if m.loading != 3 {
		t.Fatalf("want loading=3 after seed, got %d", m.loading)
	}

	next, _ := m.Update(statusMsg{
		path:   "/repo-feature",
		status: git.Status{Dirty: true, Ahead: 2, Behind: 1, Upstream: "origin/feature"},
		size:   2048,
	})
	m = next.(*model)

	var fr *row
	for i := range m.rows {
		if m.rows[i].wt.Path == "/repo-feature" {
			fr = &m.rows[i]
		}
	}
	if fr == nil || !fr.loaded {
		t.Fatal("feature row not marked loaded")
	}
	if !fr.status.Dirty || fr.status.Ahead != 2 || fr.size != 2048 {
		t.Fatalf("status not applied: %+v", fr.status)
	}
	if m.loading != 2 {
		t.Fatalf("want loading=2 after one status, got %d", m.loading)
	}
}

func TestEnterSelectsAndQuits(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)
	m.moveCursor(1) // highlight /repo-feature

	next, cmd := m.Update(KeyMsg{Type: keyEnter})
	m = next.(*model)

	if m.selectedPath != "/repo-feature" {
		t.Fatalf("want selectedPath=/repo-feature, got %q", m.selectedPath)
	}
	if !m.quitting {
		t.Fatal("want quitting=true after enter")
	}
	if cmd == nil {
		t.Fatal("want a quit Cmd")
	}
	if _, ok := cmd().(quitMsg); !ok {
		t.Fatal("enter Cmd should produce quitMsg")
	}
}

func TestRemoveOpensModalAndConfirmCallsActions(t *testing.T) {
	t.Parallel()
	m, acts := newTestModel(false)
	m.seed(t)
	m.moveCursor(1) // /repo-feature

	next, _ := m.Update(rk("d"))
	m = next.(*model)
	if m.mode != modeConfirm {
		t.Fatalf("want modeConfirm, got %v", m.mode)
	}
	if m.confirmKind != actRemove {
		t.Fatalf("want actRemove, got %v", m.confirmKind)
	}

	// confirm with y -> should fire doRemove cmd
	next, cmd := m.Update(rk("y"))
	m = next.(*model)
	if m.mode != modeList {
		t.Fatalf("modal should close, mode=%v", m.mode)
	}
	if cmd == nil {
		t.Fatal("expected remove Cmd")
	}
	cmd() // run the action

	acts.mu.Lock()
	defer acts.mu.Unlock()
	if len(acts.removed) != 1 || acts.removed[0] != "/repo-feature" {
		t.Fatalf("Remove not called correctly: %v", acts.removed)
	}
	if acts.removedDel[0] != false {
		t.Fatal("d should not delete branch")
	}
}

func TestRemoveDeleteBranchUppercase(t *testing.T) {
	t.Parallel()
	m, acts := newTestModel(false)
	m.seed(t)
	m.moveCursor(2) // /repo-bugfix

	next, _ := m.Update(rk("D"))
	m = next.(*model)
	if m.confirmKind != actRemoveDeleteBranch {
		t.Fatalf("want actRemoveDeleteBranch, got %v", m.confirmKind)
	}
	next, cmd := m.Update(rk("y"))
	m = next.(*model)
	cmd()

	acts.mu.Lock()
	defer acts.mu.Unlock()
	if len(acts.removed) != 1 || acts.removed[0] != "/repo-bugfix" {
		t.Fatalf("Remove target wrong: %v", acts.removed)
	}
	if acts.removedDel[0] != true {
		t.Fatal("D should delete branch")
	}
}

func TestConfirmCancelDoesNotRemove(t *testing.T) {
	t.Parallel()
	m, acts := newTestModel(false)
	m.seed(t)
	m.moveCursor(1)

	next, _ := m.Update(rk("d"))
	m = next.(*model)
	next, cmd := m.Update(rk("n")) // cancel
	m = next.(*model)
	if m.mode != modeList {
		t.Fatal("modal should close on cancel")
	}
	if cmd != nil {
		t.Fatal("cancel should issue no command")
	}
	acts.mu.Lock()
	defer acts.mu.Unlock()
	if len(acts.removed) != 0 {
		t.Fatalf("cancel must not call Remove, got %v", acts.removed)
	}
}

func TestFilterNarrowsList(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)

	// enter filter mode
	next, _ := m.Update(rk("/"))
	m = next.(*model)
	if m.mode != modeFilter {
		t.Fatalf("want modeFilter, got %v", m.mode)
	}

	for _, ch := range "feat" {
		next, _ = m.Update(rk(string(ch)))
		m = next.(*model)
	}

	if len(m.filtered) != 1 {
		t.Fatalf("want 1 filtered row for 'feat', got %d", len(m.filtered))
	}
	if got := m.rows[m.filtered[0]].wt.Branch; got != "feature" {
		t.Fatalf("want feature, got %q", got)
	}

	// backspace clears one char; widen back
	next, _ = m.Update(KeyMsg{Type: keyBackspace})
	m = next.(*model)
	if m.filterText != "fea" {
		t.Fatalf("backspace failed, filterText=%q", m.filterText)
	}

	// esc resets filter
	next, _ = m.Update(KeyMsg{Type: keyEsc})
	m = next.(*model)
	if m.filterText != "" || m.mode != modeList {
		t.Fatalf("esc should clear filter and exit, got %q mode=%v", m.filterText, m.mode)
	}
	if len(m.filtered) != 3 {
		t.Fatalf("want all 3 rows after clearing filter, got %d", len(m.filtered))
	}
}

func TestPRDisabledWhenGHUnavailable(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)

	if m.keys.pr.enabled {
		t.Fatal("PR binding should be disabled when gh unavailable")
	}
	next, cmd := m.Update(rk("p"))
	m = next.(*model)
	if m.mode == modePR {
		t.Fatal("p must not open PR view when gh unavailable")
	}
	if cmd != nil {
		t.Fatal("p should be a no-op when gh unavailable")
	}
}

func TestPREnabledAndCheckout(t *testing.T) {
	t.Parallel()
	m, acts := newTestModel(true)
	m.repo.(*fakeRepo).wts = sampleWorktrees()
	m.ghc = &fakeGH{avail: true, prs: []gh.PR{
		{Number: 42, Title: "Add feature", Branch: "feat", Author: "alice"},
		{Number: 7, Title: "Fix bug", Branch: "fix", Author: "bob"},
	}}
	m.seed(t)

	if !m.keys.pr.enabled {
		t.Fatal("PR binding should be enabled when gh available")
	}
	// open PR view
	next, cmd := m.Update(rk("p"))
	m = next.(*model)
	if m.mode != modePR {
		t.Fatalf("want modePR, got %v", m.mode)
	}
	// run the load command and feed result
	if cmd == nil {
		t.Fatal("expected loadPRs cmd")
	}
	res := cmd()
	next, _ = m.Update(res)
	m = next.(*model)
	if !m.prLoaded || len(m.prRows) != 2 {
		t.Fatalf("PRs not loaded: loaded=%v n=%d", m.prLoaded, len(m.prRows))
	}

	// move down to PR #7 and checkout
	next, _ = m.Update(KeyMsg{Type: keyDown})
	m = next.(*model)
	next, cmd = m.Update(KeyMsg{Type: keyEnter})
	m = next.(*model)
	if cmd == nil {
		t.Fatal("enter on PR should issue checkout cmd")
	}
	cmd()

	acts.mu.Lock()
	defer acts.mu.Unlock()
	if len(acts.prCheckout) != 1 || acts.prCheckout[0] != 7 {
		t.Fatalf("want CheckoutPR(7), got %v", acts.prCheckout)
	}
}

func TestNewWorktreePromptCreates(t *testing.T) {
	t.Parallel()
	m, acts := newTestModel(false)
	m.seed(t)

	next, _ := m.Update(rk("n"))
	m = next.(*model)
	if m.mode != modePrompt {
		t.Fatalf("want modePrompt, got %v", m.mode)
	}
	for _, ch := range "mybranch" {
		next, _ = m.Update(rk(string(ch)))
		m = next.(*model)
	}
	next, cmd := m.Update(KeyMsg{Type: keyEnter})
	m = next.(*model)
	if m.mode != modeList {
		t.Fatal("prompt should close on enter")
	}
	if cmd == nil {
		t.Fatal("expected create cmd")
	}
	res := cmd()
	// creating selects + quits with the new path
	next, qcmd := m.Update(res)
	m = next.(*model)
	if m.selectedPath != "/repo-mybranch" {
		t.Fatalf("want selectedPath=/repo-mybranch, got %q", m.selectedPath)
	}
	if !m.quitting || qcmd == nil {
		t.Fatal("create success should quit with the new path")
	}
	acts.mu.Lock()
	defer acts.mu.Unlock()
	if len(acts.created) != 1 || acts.created[0] != "mybranch" {
		t.Fatalf("Create not called right: %v", acts.created)
	}
}

func TestRefreshReloadsList(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)
	next, cmd := m.Update(rk("r"))
	m = next.(*model)
	if cmd == nil {
		t.Fatal("r should issue a reload cmd")
	}
	if _, ok := cmd().(worktreesMsg); !ok {
		t.Fatal("refresh cmd should produce worktreesMsg")
	}
}

func TestQuitWithoutSelection(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	m.seed(t)
	next, cmd := m.Update(rk("q"))
	m = next.(*model)
	if m.selectedPath != "" {
		t.Fatal("q must not select a path")
	}
	if !m.quitting || cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(true)
	m.seed(t)
	out := m.View()
	if !strings.Contains(out, "Worktrees") {
		t.Fatalf("list view missing header: %q", out)
	}
	m.mode = modeConfirm
	m.confirmTgt = 1
	m.confirmKind = actRemoveDeleteBranch
	if !strings.Contains(m.View(), "delete its branch") {
		t.Fatal("confirm modal not rendered")
	}
	m.mode = modePR
	_ = m.View()
}

func TestConcurrencyBounded(t *testing.T) {
	t.Parallel()
	if got := concurrency(); got < 1 || got > 8 {
		t.Fatalf("concurrency out of range: %d", got)
	}
}

func TestLoadStatusesFansOut(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{
		wts:    sampleWorktrees(),
		status: map[string]git.Status{"/repo-feature": {Dirty: true}},
		sizes:  map[string]int64{"/repo-feature": 100},
	}
	cmd := loadStatuses(repo, repo.wts)
	msg := cmd()
	batch, ok := msg.(batchMsg)
	if !ok {
		t.Fatalf("loadStatuses should return a batch, got %T", msg)
	}
	if len(batch) != 3 {
		t.Fatalf("want 3 child cmds, got %d", len(batch))
	}
	// each child returns a statusMsg
	for _, c := range batch {
		if _, ok := c().(statusMsg); !ok {
			t.Fatal("child cmd should return statusMsg")
		}
	}
}

func TestWorktreesErrorSurfaced(t *testing.T) {
	t.Parallel()
	m, _ := newTestModel(false)
	next, _ := m.Update(worktreesMsg{err: errors.New("boom")})
	m = next.(*model)
	if !strings.Contains(m.statusMsg, "boom") {
		t.Fatalf("error not surfaced: %q", m.statusMsg)
	}
}

var _ git.Repo = (*fakeRepo)(nil)
var _ gh.Client = (*fakeGH)(nil)
var _ Actions = (*fakeActions)(nil)
