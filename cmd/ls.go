package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/git"
)

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List worktrees for this repo, color-coded by state",
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
			states, _ := d.repo.BranchStates() // best-effort; classify falls back to active
			cur, _ := d.repo.Root()

			// Classify and measure for column alignment (on plain text).
			cells := make([]string, len(wts))
			classes := make([]string, len(wts))
			maxBranch, maxPath := 0, 0
			anyStale := false
			for i, wt := range wts {
				st := git.ClassifyWorktree(wt, states)
				classes[i] = st
				cells[i] = branchCell(wt, st)
				if len(cells[i]) > maxBranch {
					maxBranch = len(cells[i])
				}
				if len(wt.Path) > maxPath {
					maxPath = len(wt.Path)
				}
				if git.IsStale(st) {
					anyStale = true
				}
			}

			for i, wt := range wts {
				isCur := filepath.Clean(wt.Path) == filepath.Clean(cur)
				marker := "  "
				if isCur {
					marker = stateColors.cyan.Render("* ")
				}
				branch := stateStyle(classes[i]).Render(fmt.Sprintf("%-*s", maxBranch, cells[i]))
				pathStyle := stateColors.bold
				if isCur {
					pathStyle = stateColors.cyanBold
				}
				path := pathStyle.Render(fmt.Sprintf("%-*s", maxPath, wt.Path))
				fmt.Printf("%s%s  %s  %s\n", marker, branch, path, stateColors.dim.Render(wt.Head))
			}

			if anyStale {
				fmt.Println(legend())
			}
			return nil
		},
	}
}

// branchCell renders the plain (uncolored) branch/state column for a worktree:
//
//	[branch]            active or local-only (color carries the meaning)
//	[branch] (gone)     stale states tag the branch
//	(detached)          no branch (detached / bare / missing-without-branch)
func branchCell(wt git.Worktree, state string) string {
	if wt.Branch == "" {
		return "(" + state + ")"
	}
	if git.IsStale(state) {
		return "[" + wt.Branch + "] (" + state + ")"
	}
	return "[" + wt.Branch + "]"
}

// stateColors holds the lipgloss styles used by ls (and the legend). Color is
// governed by the global lipgloss profile, set from the --color policy.
var stateColors = struct {
	green, blue, red, yellow, cyan, cyanBold, bold, dim lipgloss.Style
}{
	green:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	blue:     lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	red:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	yellow:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	cyan:     lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	cyanBold: lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
	bold:     lipgloss.NewStyle().Bold(true),
	dim:      lipgloss.NewStyle().Faint(true),
}

func stateStyle(state string) lipgloss.Style {
	switch state {
	case git.StateActive:
		return stateColors.green
	case git.StateLocal:
		return stateColors.blue
	case git.StateGone, git.StateMissing:
		return stateColors.red
	case git.StateDetached:
		return stateColors.yellow
	default: // bare
		return stateColors.dim
	}
}

// legend is the one-line color key, shown by ls when stale worktrees exist.
func legend() string {
	c := stateColors
	return c.dim.Render("states: ") +
		c.green.Render("active") + "  " +
		c.blue.Render("local-only") + "  " +
		c.red.Render("gone") + "  " +
		c.red.Render("missing") + "  " +
		c.yellow.Render("detached")
}
