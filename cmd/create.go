package cmd

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/fzf"
	"github.com/kyriacos/go-gwt/internal/tui"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

// branchArg returns the branch from args, or opens the interactive picker when
// no argument was given. Returns "" when the picker is cancelled.
// populateVerb is the subcommand name ("from" or "co") used for GWT_POPULATE.
func branchArg(d *deps, args []string, populateVerb string) (string, error) {
	if len(args) >= 1 {
		return args[0], nil
	}
	return pickBranch(d, populateVerb)
}

// pickBranch opens the interactive branch picker (used by from/co with no
// argument). When fzf is available, emits GWT_POPULATE for shell review (bash
// parity). Returns "" if the user cancels.
func pickBranch(d *deps, populateVerb string) (string, error) {
	if fzf.Available() {
		lines, err := fzf.BuildBranchLines(d.repo)
		if err != nil {
			return "", err
		}
		branch, populate, err := fzf.PickBranch(populateVerb, lines)
		if err != nil || populate != "" {
			if populate != "" {
				ui.Populate(populate)
			}
			return "", err
		}
		return branch, nil
	}
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
	path            string
	cursorRunSetup  bool
	cursorNoSetup   bool
	claudeRunSetup  bool
	claudeNoSetup   bool
	legacyRunSetup  bool
	legacyNoSetup   bool
	open            bool
}

func (f *createFlags) bind(c *cobra.Command) {
	c.Flags().StringVarP(&f.path, "path", "p", "", "parent dir for the new worktree")
	c.Flags().BoolVar(&f.cursorRunSetup, "cursor-run-setup", false, "run Cursor worktree setup (.cursor/worktrees.json) without prompting")
	c.Flags().BoolVar(&f.cursorNoSetup, "cursor-no-setup", false, "skip Cursor worktree setup")
	c.Flags().BoolVar(&f.claudeRunSetup, "claude-run-setup", false, "run Claude worktree setup (.worktreeinclude) without prompting")
	c.Flags().BoolVar(&f.claudeNoSetup, "claude-no-setup", false, "skip Claude worktree setup")
	c.Flags().BoolVar(&f.legacyRunSetup, "run-setup", false, "deprecated alias for --cursor-run-setup")
	c.Flags().BoolVar(&f.legacyNoSetup, "no-setup", false, "deprecated alias for --cursor-no-setup")
	c.Flags().BoolVar(&f.open, "open", false, "open the worktree in your editor after creating it")
}

func (f *createFlags) cursorSetupMode() worktree.SetupMode {
	switch {
	case f.cursorNoSetup || f.legacyNoSetup:
		return worktree.SetupNo
	case f.cursorRunSetup || f.legacyRunSetup:
		return worktree.SetupYes
	default:
		return worktree.SetupDefault
	}
}

func (f *createFlags) claudeSetupMode() worktree.SetupMode {
	switch {
	case f.claudeNoSetup:
		return worktree.SetupNo
	case f.claudeRunSetup:
		return worktree.SetupYes
	default:
		return worktree.SetupDefault
	}
}

func (f *createFlags) createOpts(name, base string, newBranch bool) worktree.CreateOpts {
	return worktree.CreateOpts{
		Name:              name,
		Base:              base,
		NewBranch:         newBranch,
		ParentDir:         f.path,
		CursorSetupChoice: f.cursorSetupMode(),
		ClaudeSetupChoice: f.claudeSetupMode(),
		OpenEditor:        f.open,
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
			branch, err := branchArg(d, args, "from")
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
			name, err := branchArg(d, args, "co")
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
