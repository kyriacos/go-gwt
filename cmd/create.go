package cmd

import (
	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

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
		Use:   "from <branch>",
		Short: "Create a worktree for an existing branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			res, err := d.svc.Create(f.createOpts(args[0], "", false))
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
		Use:     "co <name>",
		Aliases: []string{"checkout"},
		Short:   "Switch to a worktree, creating it from a branch if needed",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			res, err := d.svc.Switch(args[0], f.createOpts(args[0], "", false))
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
