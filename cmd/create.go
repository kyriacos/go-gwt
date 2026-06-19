package cmd

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/tui"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

// branchArg returns the branch from args, or opens the interactive picker when
// no argument was given. Returns "" when the picker is cancelled.
func branchArg(d *deps, args []string) (string, error) {
	if len(args) >= 1 {
		return args[0], nil
	}
	return pickBranch(d)
}

// pickBranch opens the interactive branch picker (used by from/co with no
// argument). Returns "" if the user cancels.
func pickBranch(d *deps) (string, error) {
	states, err := d.repo.BranchStates()
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(states))
	for n := range states {
		names = append(names, n)
	}
	sort.Strings(names)
	items := make([]tui.BranchItem, len(names))
	for i, n := range names {
		items[i] = tui.BranchItem{Name: n, State: states[n]}
	}
	return tui.PickBranch(items)
}

// createFlags are the options shared by new/from/co/pr.
type createFlags struct {
	path     string
	runSetup bool
	noSetup  bool
	open     bool
}

func (f *createFlags) bind(c *cobra.Command) {
	c.Flags().StringVarP(&f.path, "path", "p", "", "parent dir for the new worktree")
	c.Flags().BoolVar(&f.runSetup, "run-setup", false, "run repo setup commands without prompting")
	c.Flags().BoolVar(&f.noSetup, "no-setup", false, "skip repo setup commands")
	c.Flags().BoolVar(&f.open, "open", false, "open the worktree in your editor after creating it")
}

func (f *createFlags) setupMode() worktree.SetupMode {
	switch {
	case f.noSetup:
		return worktree.SetupNo
	case f.runSetup:
		return worktree.SetupYes
	default:
		return worktree.SetupDefault
	}
}

func (f *createFlags) createOpts(name, base string, newBranch bool) worktree.CreateOpts {
	return worktree.CreateOpts{
		Name:        name,
		Base:        base,
		NewBranch:   newBranch,
		ParentDir:   f.path,
		SetupChoice: f.setupMode(),
		OpenEditor:  f.open,
	}
}

func newNewCmd() *cobra.Command {
	var f createFlags
	c := &cobra.Command{
		Use:   "new <name> [base]",
		Short: "Create a worktree on a new branch",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			base := ""
			if len(args) == 2 {
				base = args[1]
			}
			res, err := d.svc.Create(f.createOpts(args[0], base, true))
			if err != nil {
				return err
			}
			ui.Path(res.Path)
			return nil
		},
	}
	f.bind(c)
	return c
}

func newFromCmd() *cobra.Command {
	var f createFlags
	c := &cobra.Command{
		Use:   "from [branch]",
		Short: "Create a worktree for an existing branch (no arg opens a picker)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			branch, err := branchArg(d, args)
			if err != nil || branch == "" {
				return err
			}
			res, err := d.svc.Create(f.createOpts(branch, "", false))
			if err != nil {
				return err
			}
			ui.Path(res.Path)
			return nil
		},
	}
	f.bind(c)
	return c
}

func newCoCmd() *cobra.Command {
	var f createFlags
	c := &cobra.Command{
		Use:     "co [name]",
		Aliases: []string{"checkout"},
		Short:   "Switch to a worktree, creating it from a branch if needed (no arg opens a picker)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			name, err := branchArg(d, args)
			if err != nil || name == "" {
				return err
			}
			res, err := d.svc.Switch(name, f.createOpts(name, "", false))
			if err != nil {
				return err
			}
			ui.Path(res.Path)
			return nil
		},
	}
	f.bind(c)
	return c
}

func newPRCmd() *cobra.Command {
	var f createFlags
	c := &cobra.Command{
		Use:   "pr <number>",
		Short: "Check out a GitHub PR into a fresh worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			if !d.ghc.Available() {
				return errGHUnavailable
			}
			n, err := parsePRNumber(args[0])
			if err != nil {
				return err
			}
			branch, err := d.ghc.Checkout(n)
			if err != nil {
				return err
			}
			res, err := d.svc.Create(f.createOpts(branch, "", false))
			if err != nil {
				return err
			}
			ui.Path(res.Path)
			return nil
		},
	}
	f.bind(c)
	return c
}
