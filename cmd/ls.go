package cmd

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/ui"
)

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List worktrees for this repo",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			wts, err := d.repo.List()
			if err != nil {
				return err
			}
			cur, _ := d.repo.Root()

			// Fetch status concurrently; ls is read-only and order is preserved.
			statuses := make([]git.Status, len(wts))
			var wg sync.WaitGroup
			sem := make(chan struct{}, 8)
			for i, wt := range wts {
				if wt.Bare || wt.Detached || wt.Branch == "" {
					continue
				}
				wg.Add(1)
				sem <- struct{}{}
				go func(i int, path string) {
					defer wg.Done()
					defer func() { <-sem }()
					statuses[i], _ = d.repo.Status(path)
				}(i, wt.Path)
			}
			wg.Wait()

			width := 0
			for _, wt := range wts {
				if len(wt.Path) > width {
					width = len(wt.Path)
				}
			}

			for i, wt := range wts {
				marker := "  "
				path := wt.Path
				if filepath.Clean(wt.Path) == filepath.Clean(cur) {
					marker = ui.Bold("* ")
					path = ui.Bold(wt.Path)
				}
				var branch string
				switch {
				case wt.Bare:
					branch = "(bare)"
				case wt.Detached:
					branch = "(detached)"
				default:
					branch = "[" + wt.Branch + "]"
				}
				st := statuses[i]
				flags := ""
				if st.Ahead > 0 {
					flags += fmt.Sprintf(" ↑%d", st.Ahead)
				}
				if st.Behind > 0 {
					flags += fmt.Sprintf(" ↓%d", st.Behind)
				}
				if st.Dirty {
					flags += " ●"
				}
				pad := width - len(wt.Path) + 2
				fmt.Printf("%s%s%*s%s  %s%s\n", marker, path, pad, "", wt.Head, branch, flags)
			}
			return nil
		},
	}
}
