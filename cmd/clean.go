package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/fzf"
	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/tui"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

func newCleanCmd() *cobra.Command {
	var (
		merged       bool
		dryRun       bool
		deleteBranch bool
		forceDelete  bool
	)
	c := &cobra.Command{
		Use:     "clean",
		Short:   "Remove worktrees: interactive multi-select, or --merged for a non-interactive sweep",
		Long:    cleanLong,
		Example: cleanExample,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			del, forceDel := resolveBranchDeletion(cmd, d.cfg, deleteBranch, forceDelete)
			if merged {
				return cleanMerged(d, dryRun, del, forceDel)
			}
			return cleanInteractive(d, del, forceDel)
		},
	}
	c.Flags().BoolVar(&merged, "merged", false, "non-interactive: remove worktrees whose branch is merged into the default branch")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "with --merged, list what would be removed without removing")
	c.Flags().BoolVarP(&deleteBranch, "delete-branch", "d", false, "also delete the local branch of each removed worktree")
	c.Flags().BoolVarP(&forceDelete, "force-delete-branch", "D", false, "force-delete the local branch even if not fully merged")
	return c
}

// cleanInteractive opens the multi-select picker (stale entries pre-marked) and
// removes whatever the user confirms.
func cleanInteractive(d *deps, deleteBranch, forceDelete bool) error {
	if fzfReady(d.cfg) {
		return cleanInteractiveFzf(d, deleteBranch, forceDelete)
	}
	return cleanInteractiveTUI(d, deleteBranch, forceDelete)
}

func cleanInteractiveFzf(d *deps, deleteBranch, forceDelete bool) error {
	lines, stateByPath, err := fzf.FormatCleanLines(d.repo)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		ui.Info("No removable worktrees.")
		return nil
	}
	paths, err := fzf.PickWorktreesMulti(lines, fzf.FormatCleanHeader())
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		ui.Info("Nothing selected.")
		return nil
	}
	return removeCleanPaths(d, paths, stateByPath, deleteBranch, forceDelete)
}

func cleanInteractiveTUI(d *deps, deleteBranch, forceDelete bool) error {
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
	return removeCleanPaths(d, paths, stateByPath, deleteBranch, forceDelete)
}

func removeCleanPaths(d *deps, paths []string, stateByPath map[string]string, deleteBranch, forceDelete bool) error {
	for _, p := range paths {
		st := stateByPath[p]
		del, force := deleteBranch, forceDelete
		// Stale worktrees (gone/missing) still force-delete the branch unless the
		// user explicitly passed -d=false… which isn't possible; only skip when
		// neither flag nor stale applies.
		if !del && !force && git.IsStale(st) {
			del, force = true, true
		}
		_, rerr := d.svc.Remove(worktree.RemoveOpts{
			Target:       p,
			Force:        st == git.StateMissing,
			DeleteBranch: del,
			ForceDelete:  force,
		})
		if rerr != nil {
			ui.Warn("skipped %s: %v", p, rerr)
		}
	}
	_ = d.repo.Prune()
	return nil
}

// cleanMerged is the non-interactive --merged sweep.
func cleanMerged(d *deps, dryRun, deleteBranch, forceDelete bool) error {
	results, err := d.svc.CleanMerged(dryRun, deleteBranch, forceDelete)
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
