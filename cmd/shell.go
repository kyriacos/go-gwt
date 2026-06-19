package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/shell"
)

func newShellInitCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "shell-init <" + strings.Join(shell.Shells(), "|") + ">",
		Short: "Print the shell wrapper that lets gwt cd for you",
		Long: "Emit a shell function wrapper. Add it to your shell rc, e.g.\n" +
			"  eval \"$(gwt shell-init zsh)\"          # zsh/bash\n" +
			"  gwt shell-init fish | source          # fish\n\n" +
			"Use --name when the binary is installed under a different name, so the\n" +
			"wrapper function and the command it calls match it:\n" +
			"  eval \"$(gogwt shell-init zsh --name gogwt)\"",
		Args:      cobra.ExactArgs(1),
		ValidArgs: shell.Shells(),
		RunE: func(_ *cobra.Command, args []string) error {
			script, err := shell.Init(args[0], name)
			if err != nil {
				return err
			}
			fmt.Print(script)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "gwt", "command/function name the wrapper uses (match the installed binary)")
	return c
}
