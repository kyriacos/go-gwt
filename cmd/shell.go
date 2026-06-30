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
		Use:     "shell-init <" + strings.Join(shell.Shells(), "|") + ">",
		Short:   "Print the shell wrapper that lets gwt cd for you",
		Long:    shellInitLong,
		Example: shellInitExample,
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
