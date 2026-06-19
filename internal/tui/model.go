package tui

import (
	"io"
	"os"
	"runtime"

	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
)

// Actions is the small action surface the dashboard triggers. It is satisfied
// by *worktree.Service; it is kept local so the TUI does not import the
// worktree package (decoupling for parallel development).
//
// The integration agent should add thin adapter methods on *worktree.Service if
// its signatures differ:
//
//	Create(name string, newBranch bool) (path string, err error)
//	    Create a worktree. newBranch=true means create a new branch named name
//	    (git worktree add -b); newBranch=false means check out an existing
//	    branch. Returns the absolute path of the created worktree.
//	Remove(target string, deleteBranch, force bool) error
//	    Remove the worktree at path/branch `target`. deleteBranch deletes the
//	    branch too (-d, or -D when force). force bypasses dirty/unpushed checks.
//	CleanMerged(dryRun bool) ([]string, error)
//	    Remove worktrees whose branch is merged into the default branch; returns
//	    the paths affected (or that would be, when dryRun).
//	CheckoutPR(number int) (path string, err error)
//	    Check out PR #number into a new worktree and return its path.
type Actions interface {
	Create(name string, newBranch bool) (path string, err error)
	Remove(target string, deleteBranch, force bool) error
	CleanMerged(dryRun bool) ([]string, error)
	CheckoutPR(number int) (path string, err error)
}

// PreviewFunc returns the preview text for the worktree at path, typically
// `git log --oneline --graph --decorate -n 15`. It is provided by the caller so
// the TUI stays decoupled from any git exec. A nil PreviewFunc disables the
// preview pane.
type PreviewFunc func(path string) (string, error)

// row is one worktree line plus its lazily loaded status.
type row struct {
	wt      git.Worktree
	status  git.Status
	size    int64
	loaded  bool   // status + size have arrived
	loadErr string // non-empty if the status load failed
	ciState gh.CIState
	hasCI   bool
}

// mode is the dashboard's top-level screen/state.
type mode int

const (
	modeList    mode = iota // browsing the worktree list
	modeFilter              // typing in the filter box
	modePrompt              // typing a new-worktree branch name
	modeConfirm             // destructive-action confirmation modal
	modePR                  // PR list
)

// pendingAction records what a confirm modal will do when accepted.
type pendingAction int

const (
	actNone pendingAction = iota
	actRemove
	actRemoveDeleteBranch
)

// model is the dashboard state. It implements the local Model interface.
type model struct {
	repo    git.Repo
	ghc     gh.Client
	acts    Actions
	preview PreviewFunc

	keys   keyMap
	styles styles

	rows     []row
	filtered []int // indices into rows matching the current filter
	cursor   int   // index into filtered

	mode        mode
	filterText  string
	promptText  string
	confirmKind pendingAction
	confirmTgt  int // row index targeted by the confirm modal

	prRows   []gh.PR
	prCursor int
	prErr    string
	prLoaded bool

	// preview cache for the highlighted row
	previewPath string
	previewText string
	previewErr  string

	spinFrame int
	loading   int // outstanding status loads

	width, height int

	statusMsg string // transient one-line message (errors, results)

	// result + exit
	selectedPath string
	quitting     bool

	ghAvailable bool
}

// newModel builds the initial model. It does not start any commands; Init does.
func newModel(repo git.Repo, ghc gh.Client, acts Actions, preview PreviewFunc) *model {
	ghOK := ghc != nil && ghc.Available()
	return &model{
		repo:        repo,
		ghc:         ghc,
		acts:        acts,
		preview:     preview,
		keys:        defaultKeyMap(ghOK),
		styles:      newStyles(),
		mode:        modeList,
		width:       100,
		height:      30,
		ghAvailable: ghOK,
	}
}

// concurrency bounds the status-load fan-out to ~min(8, NumCPU).
func concurrency() int {
	n := runtime.NumCPU()
	if n > 8 {
		n = 8
	}
	if n < 1 {
		n = 1
	}
	return n
}

// Run launches the dashboard on the terminal and returns the path the user
// selected (empty string if they quit without selecting).
//
// repo and ghc supply reads; acts triggers create/remove/clean/PR-checkout;
// preview supplies the right-pane log text (may be nil to disable it).
//
// The program renders to os.Stderr and reads from the controlling tty (falling
// back to os.Stdin), because stdout is reserved for the selected path under the
// cd protocol. The caller prints the returned path to stdout on success.
func Run(repo git.Repo, ghc gh.Client, acts Actions, preview PreviewFunc) (selectedPath string, err error) {
	final, err := runModel(newModel(repo, ghc, acts, preview))
	if err != nil {
		return "", err
	}
	if fm, ok := final.(*model); ok {
		return fm.selectedPath, nil
	}
	return "", nil
}

// runModel drives any Model to completion on the controlling terminal and
// returns the final model. It prefers /dev/tty (so it works even when
// stdin/stdout are redirected and so it can enter raw mode and read the real
// size), falling back to os.Stdin. Rendering goes to stderr; stdout is reserved
// for machine output (the chosen path) under the cd protocol.
func runModel(m Model) (Model, error) {
	var tty *os.File
	if f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		tty = f
		defer tty.Close()
	}
	var in io.Reader = os.Stdin
	if tty != nil {
		in = tty
	}
	p := newProgram(m, in, os.Stderr, tty)
	if err := p.run(); err != nil {
		return nil, err
	}
	return p.model, nil
}
