package tui

import (
	"time"

	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
)

// ---- Messages -------------------------------------------------------------

// worktreesMsg carries the initial (or refreshed) worktree list.
type worktreesMsg struct {
	wts []git.Worktree
	err error
}

// statusMsg carries one worktree's loaded status + disk size. One arrives per
// worktree as the concurrent loads complete.
type statusMsg struct {
	path   string
	status git.Status
	size   int64
	err    error
}

// previewMsg carries the preview text for a path.
type previewMsg struct {
	path string
	text string
	err  error
}

// prListMsg carries the PR list.
type prListMsg struct {
	prs []gh.PR
	err error
}

// actionDoneMsg reports the result of a create/remove/clean/PR-checkout.
type actionDoneMsg struct {
	verb string // "created", "removed", "cleaned", "checkout"
	path string // resulting path for create/checkout; selects + quits on success
	msg  string // human message for the status line
	err  error
}

// tickMsg advances the spinner animation.
type tickMsg struct{}

// ---- Commands -------------------------------------------------------------

// loadWorktrees lists worktrees.
func loadWorktrees(repo git.Repo) Cmd {
	return func() Msg {
		wts, err := repo.List()
		return worktreesMsg{wts: wts, err: err}
	}
}

// loadStatuses fans out one status+disk command per worktree, bounded by a
// semaphore to ~min(8, NumCPU). It returns a batchMsg of per-worktree Cmds so
// the runtime runs them concurrently and each posts a statusMsg as it lands.
func loadStatuses(repo git.Repo, wts []git.Worktree) Cmd {
	sem := make(chan struct{}, concurrency())
	cmds := make([]Cmd, 0, len(wts))
	for _, w := range wts {
		w := w
		cmds = append(cmds, func() Msg {
			sem <- struct{}{}
			defer func() { <-sem }()
			st, err := repo.Status(w.Path)
			size, szErr := repo.DiskUsage(w.Path)
			if err == nil {
				err = szErr
			}
			return statusMsg{path: w.Path, status: st, size: size, err: err}
		})
	}
	return Batch(cmds...)
}

// loadPreview fetches the preview text for a path.
func loadPreview(fn PreviewFunc, path string) Cmd {
	if fn == nil {
		return nil
	}
	return func() Msg {
		txt, err := fn(path)
		return previewMsg{path: path, text: txt, err: err}
	}
}

// loadPRs lists open PRs.
func loadPRs(c gh.Client) Cmd {
	return func() Msg {
		prs, err := c.ListPRs()
		return prListMsg{prs: prs, err: err}
	}
}

// doCreate creates a worktree for a new branch.
func doCreate(a Actions, name string) Cmd {
	return func() Msg {
		path, err := a.Create(name, true)
		return actionDoneMsg{verb: "created", path: path, msg: "created " + name, err: err}
	}
}

// doRemove removes a worktree (optionally deleting the branch).
func doRemove(a Actions, target string, deleteBranch bool) Cmd {
	return func() Msg {
		err := a.Remove(target, deleteBranch, false)
		return actionDoneMsg{verb: "removed", msg: "removed " + target, err: err}
	}
}

// doCheckoutPR checks a PR out into a new worktree.
func doCheckoutPR(a Actions, number int) Cmd {
	return func() Msg {
		path, err := a.CheckoutPR(number)
		return actionDoneMsg{verb: "checkout", path: path, msg: "checked out PR", err: err}
	}
}

// tick schedules the next spinner frame.
func tick() Cmd {
	return func() Msg {
		time.Sleep(100 * time.Millisecond)
		return tickMsg{}
	}
}
