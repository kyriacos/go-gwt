package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/fzf"
	"github.com/kyriacos/go-gwt/internal/ui"
)

func runSearch(d *deps) error {
	if fzfReady(d.cfg) {
		lines, _, err := fzf.BuildWorktreeLines(d.repo)
		if err != nil {
			return err
		}
		path, err := fzf.PickWorktree(lines)
		if err != nil {
			return err
		}
		if path != "" {
			ui.Path(path)
		}
		return nil
	}
	return runDashboard()
}

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "search",
		Aliases: []string{"pick"},
		Short:   "Fuzzy-search worktrees and print the chosen path",
		Long:    searchLong,
		Example: searchExample,
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			return runSearch(d)
		},
	}
}
