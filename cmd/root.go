// Package cmd wires the CLI: flag parsing and dependency assembly only, no
// business logic (that lives in internal/worktree and internal/tui).
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/gh"
	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/worktree"
)

var versionInfo struct {
	version, commit, date string
}

// forceTUI makes interactive pickers use the built-in Bubble Tea UI instead of
// fzf when both are available.
var forceTUI bool

// deps bundles the assembled dependencies handed to command bodies.
type deps struct {
	runner exec.Runner
	repo   *git.CmdRepo
	ghc    gh.Client
	cfg    config.Config
	svc    *worktree.Service
}

// build assembles dependencies for a command run.
func build() (*deps, error) {
	runner := exec.New()
	repo := git.New(runner)
	root, _ := repo.MainWorktree() // best-effort; "" is fine for config lookup
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	ghc := gh.New(runner)
	svc := worktree.New(repo, ghc, cfg, runner)
	return &deps{runner: runner, repo: repo, ghc: ghc, cfg: cfg, svc: svc}, nil
}

// NewRootCmd constructs the root command and all subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gwt",
		Short:         "git worktree helper with a TUI and gh integration",
		Long:          rootLong,
		Example:       rootExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Apply the color policy before any command runs.
		PersistentPreRun: func(c *cobra.Command, _ []string) {
			switch v, _ := c.Flags().GetString("color"); v {
			case "always":
				ui.SetColor(ui.Always)
			case "never":
				ui.SetColor(ui.Never)
			default:
				ui.SetColor(ui.Auto)
			}
		},
		// Bare `gwt` with a terminal opens the dashboard; otherwise help.
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 && ui.HasTTY() {
				return runDashboard()
			}
			return c.Help()
		},
	}

	root.PersistentFlags().String("color", "always", "color output: always|auto|never")
	root.PersistentFlags().BoolVar(&forceTUI, "tui", false, "use built-in TUI pickers (with log preview) instead of fzf")

	root.AddCommand(
		newNewCmd(),
		newFromCmd(),
		newCoCmd(),
		newRmCmd(),
		newPRCmd(),
		newLsCmd(),
		newSearchCmd(),
		newCleanCmd(),
		newPruneCmd(),
		newPassthroughCmd("st", "git status (short) for the current worktree", stLong, stExample, []string{"status", "-sb"}, "status"),
		newPassthroughCmd("log", "git log (oneline graph) for the current worktree", logLong, logExample, []string{"log", "--oneline", "--graph", "--decorate", "-n", "20"}),
		newDashboardCmd(),
		newShellInitCmd(),
		newVersionCmd(),
	)
	initHelp(root)
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print version information",
		Long:    versionLong,
		Example: versionExample,
		Run: func(*cobra.Command, []string) {
			fmt.Printf("gwt %s (commit %s, built %s)\n",
				versionInfo.version, versionInfo.commit, versionInfo.date)
		},
	}
}

// Execute runs the root command. Called from main.
func Execute(version, commit, date string) {
	versionInfo.version, versionInfo.commit, versionInfo.date = version, commit, date
	if err := NewRootCmd().Execute(); err != nil {
		ui.Die("%v", err)
	}
}
