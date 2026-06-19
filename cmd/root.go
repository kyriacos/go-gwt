// Package cmd wires the CLI: flag parsing and dependency assembly only, no
// business logic (that lives in internal/worktree and internal/tui).
package cmd

import (
	"fmt"
	"os"

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

// deps bundles the assembled dependencies handed to command bodies.
type deps struct {
	repo *git.CmdRepo
	ghc  gh.Client
	cfg  config.Config
	svc  *worktree.Service
}

// build assembles dependencies for a command run. The integration agent uses
// this from each command's RunE.
func build() (*deps, error) {
	runner := exec.New()
	repo := git.New(runner)
	root, _ := repo.MainWorktree() // best-effort; "" is fine for config lookup
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	ghc := gh.New(runner)
	svc := worktree.New(repo, ghc, cfg)
	return &deps{repo: repo, ghc: ghc, cfg: cfg, svc: svc}, nil
}

// NewRootCmd constructs the root command and its subcommands. Subcommand
// bodies are filled in by the integration agent; the foundation registers
// version and a placeholder so the tree compiles and --help works.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gwt",
		Short:         "git worktree helper with a TUI and gh integration",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			// Bare `gwt` will launch the dashboard (TUI agent). For now, help.
			return c.Help()
		},
	}

	root.PersistentFlags().String("color", "auto", "color output: auto|always|never")

	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
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
		ui.Err("%v", err)
		os.Exit(1)
	}
}
