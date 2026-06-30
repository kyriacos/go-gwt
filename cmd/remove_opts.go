package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/config"
)

// resolveBranchDeletion maps CLI -d/-D flags and config defaults into RemoveOpts
// fields. When either CLI flag was set, config is ignored.
func resolveBranchDeletion(c *cobra.Command, cfg config.Config, cliDelete, cliForce bool) (deleteBranch, forceDelete bool) {
	if c.Flags().Changed("delete-branch") || c.Flags().Changed("force-delete-branch") {
		if cliForce {
			return true, true
		}
		if cliDelete {
			return true, false
		}
		return false, false
	}
	return cfg.DefaultBranchDeletion()
}
