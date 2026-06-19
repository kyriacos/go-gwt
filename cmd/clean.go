package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/tui"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

func newCleanCmd() *cobra.Command {
	var (
		merged bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "clean",
		Short: "Remove worktrees: interactive multi-select, or --merged for a non-interactive sweep",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			if merged {
				return cleanMerged(d, dryRun)
			}
			return cleanInteractive(d)
		},
	}
	c.Flags().BoolVar(&merged, "merged", false, "non-interactive: remove worktrees whose branch is merged into the default branch")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "with --merged, list what would be removed without removing")
	return c
}

// cleanInteractive opens the multi-select picker (stale entries pre-marked) and
// removes whatever the user confirms, deleting branches of stale worktrees.
func cleanInteractive(d *deps) error {
	wts, err := d.repo.List()
	if err != nil {
		return err
	}
	states, _ := d.repo.BranchStates()

	var items []tui.CleanItem
	stateByPath := map[string]string{}
	for _, wt := range wts {
		if wt.IsMain || wt.Bare {
			continue
		}
		st := git.ClassifyWorktree(wt, states)
		stateByPath[wt.Path] = st
		items = append(items, tui.CleanItem{Path: wt.Path, Branch: wt.Branch, State: st})
	}
	if len(items) == 0 {
		ui.Info("No removable worktrees.")
		return nil
	}

	paths, err := tui.PickForClean(items)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		ui.Info("Nothing selected.")
		return nil
	}

	for _, p := range paths {
		st := stateByPath[p]
		_, rerr := d.svc.Remove(worktree.RemoveOpts{
			Target:      p,
			Force:       st == git.StateMissing, // missing dirs can't be checked; force
			ForceDelete: git.IsStale(st),        // delete branches of stale (gone/missing)
		})
		if rerr != nil {
			ui.Warn("skipped %s: %v", p, rerr)
		}
	}
	_ = d.repo.Prune() // tidy any leftover missing-worktree metadata
	return nil
}

// cleanMerged is the non-interactive --merged sweep.
func cleanMerged(d *deps, dryRun bool) error {
	results, err := d.svc.CleanMerged(dryRun)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		ui.Info("No merged worktrees to clean.")
		return nil
	}
	if dryRun {
		ui.Info("Would remove %d merged worktree(s):", len(results))
		for _, r := range results {
			ui.Dim("  %s [%s]", r.Path, r.Branch)
		}
	}
	return nil
}
