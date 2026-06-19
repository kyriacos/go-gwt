package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/ui"
)

func newCleanCmd() *cobra.Command {
	var (
		merged bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "clean",
		Short: "Bulk-remove worktrees whose branch is merged into the default branch",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !merged {
				return errCleanNeedsMerged
			}
			d, err := build()
			if err != nil {
				return err
			}
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
		},
	}
	c.Flags().BoolVar(&merged, "merged", false, "remove worktrees whose branch is merged (required)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "list what would be removed without removing")
	return c
}
