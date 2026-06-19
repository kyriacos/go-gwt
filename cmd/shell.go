package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/shell"
)

func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "shell-init <" + strings.Join(shell.Shells(), "|") + ">",
		Short:     "Print the shell wrapper that lets gwt cd for you",
		Long:      "Emit a shell function wrapper. Add it to your shell rc, e.g.\n  eval \"$(gwt shell-init zsh)\"   # bash\n  gwt shell-init fish | source    # fish",
		Args:      cobra.ExactArgs(1),
		ValidArgs: shell.Shells(),
		RunE: func(_ *cobra.Command, args []string) error {
			script, err := shell.Init(args[0])
			if err != nil {
				return err
			}
			fmt.Print(script)
			return nil
		},
	}
}
