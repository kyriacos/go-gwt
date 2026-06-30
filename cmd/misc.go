package cmd

import (
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var errGHUnavailable = errors.New("gh is not available (install it and run `gh auth login`)")

func parsePRNumber(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(s, "#"))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid PR number: %q", s)
	}
	return n, nil
}

func newPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "prune",
		Short:   "Prune stale worktree metadata",
		Long:    pruneLong,
		Example: pruneExample,
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			d, err := build()
			if err != nil {
				return err
			}
			return d.repo.Prune()
		},
	}
}

// newPassthroughCmd builds a command that execs git with fixed leading args
// plus any extra user args, inheriting stdio for a normal terminal experience.
// Used for st (status) and log.
func newPassthroughCmd(use, short, long, example string, gitArgs []string, aliases ...string) *cobra.Command {
	return &cobra.Command{
		Use:                use + " [git args...]",
		Aliases:            aliases,
		Short:              short,
		Long:               long,
		Example:            example,
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			full := append(append([]string{}, gitArgs...), args...)
			g := osexec.Command("git", full...)
			g.Stdin, g.Stdout, g.Stderr = os.Stdin, os.Stdout, os.Stderr
			return g.Run()
		},
	}
}
