package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/shell"
)

func newShellInitCmd() *cobra.Command {
	var (
		name string
		bin  string
	)
	c := &cobra.Command{
		Use:       "shell-init <" + strings.Join(shell.Shells(), "|") + ">",
		Short:     "Print the shell wrapper that lets gwt cd for you",
		Long:      shellInitLong,
		Example:   shellInitExample,
		Args:      cobra.ExactArgs(1),
		ValidArgs: shell.Shells(),
		RunE: func(_ *cobra.Command, args []string) error {
			binPath := bin
			if binPath == "" {
				var err error
				binPath, err = os.Executable()
				if err != nil {
					return err
				}
				binPath, err = filepath.EvalSymlinks(binPath)
				if err != nil {
					return err
				}
			}
			script, err := shell.Init(args[0], name, binPath)
			if err != nil {
				return err
			}
			fmt.Print(script)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "gwt", "command/function name the wrapper uses (match the installed binary)")
	c.Flags().StringVar(&bin, "bin", "", "gwt binary path the wrapper invokes (default: this executable)")
	return c
}
