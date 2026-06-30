package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/tui"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

// tuiActions adapts *worktree.Service to the tui.Actions interface.
type tuiActions struct {
	svc *worktree.Service
	ghc gh.Client
}

func (a tuiActions) Create(name string, newBranch bool) (string, error) {
	res, err := a.svc.Create(worktree.CreateOpts{Name: name, NewBranch: newBranch})
	return res.Path, err
}

func (a tuiActions) Remove(target string, deleteBranch, force bool) error {
	_, err := a.svc.Remove(worktree.RemoveOpts{Target: target, DeleteBranch: deleteBranch, Force: force})
	return err
}

func (a tuiActions) CleanMerged(dryRun bool) ([]string, error) {
	results, err := a.svc.CleanMerged(dryRun, false, false)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(results))
	for i, r := range results {
		paths[i] = r.Path
	}
	return paths, nil
}

func (a tuiActions) Open(path string) error {
	return a.svc.OpenEditor(path)
}

func (a tuiActions) CheckoutPR(number int) (string, error) {
	branch, err := a.ghc.Checkout(number)
	if err != nil {
		return "", err
	}
	res, err := a.svc.Create(worktree.CreateOpts{Name: branch, NewBranch: false})
	return res.Path, err
}

// gitPreview returns a PreviewFunc that renders a short colored log graph.
func gitPreview(runner exec.Runner) tui.PreviewFunc {
	return func(path string) (string, error) {
		out, _, err := runner.Run(context.Background(), path,
			"git", "log", "--oneline", "--graph", "--decorate", "--color=always", "-n", "15")
		return string(out), err
	}
}

// branchPreview returns a BranchPreviewFunc that renders recent commits on a branch.
func branchPreview(runner exec.Runner) tui.BranchPreviewFunc {
	return func(branch string) (string, error) {
		out, _, err := runner.Run(context.Background(), "",
			"git", "log", "--oneline", "--graph", "--decorate", "--color=always", "-n", "10", branch)
		return string(out), err
	}
}

// runDashboard launches the TUI and, if the user selected a worktree, prints
// its path to stdout (the cd contract).
func runDashboard() error {
	d, err := build()
	if err != nil {
		return err
	}
	acts := tuiActions{svc: d.svc, ghc: d.ghc}
	selected, err := tui.Run(d.repo, d.ghc, acts, gitPreview(d.runner))
	if err != nil {
		return err
	}
	if selected != "" {
		ui.Path(selected)
	}
	return nil
}

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "dashboard",
		Short:   "Open the interactive worktree dashboard",
		Long:    dashboardLong,
		Example: dashboardExample,
		Args:    cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return runDashboard() },
	}
}
