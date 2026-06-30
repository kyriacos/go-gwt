package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/worktree"
)

func newRmCmd() *cobra.Command {
	var (
		force        bool
		deleteBranch bool
		forceDelete  bool
	)
	c := &cobra.Command{
		Use:     "rm [name]",
		Aliases: []string{"remove"},
		Short:   "Remove a worktree (optionally deleting its branch)",
		Long:    rmLong,
		Example: rmExample,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			_, err = d.svc.Remove(worktree.RemoveOpts{
				Target:       target,
				Force:        force,
				DeleteBranch: deleteBranch,
				ForceDelete:  forceDelete,
			})
			return err
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "discard uncommitted changes")
	c.Flags().BoolVarP(&deleteBranch, "delete-branch", "d", false, "also delete the local branch")
	c.Flags().BoolVarP(&forceDelete, "force-delete-branch", "D", false, "also force-delete the local branch")
	return c
}
