// Package exec is the single place that shells out to external programs.
// Everything that runs git or gh goes through a Runner so it can be faked in
// tests. The real runner is a thin wrapper over os/exec.
package exec

import (
	"bytes"
	"context"
	"os"
	osexec "os/exec"
)

// Runner executes a command in dir and returns its stdout and stderr. A nil or
// empty dir runs in the current working directory. err is the exec error
// (including non-zero exit, as *exec.ExitError); callers decide whether stderr
// content matters.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (stdout, stderr []byte, err error)
}

// Cmd is the production Runner backed by os/exec.
type Cmd struct{}

// New returns a Runner that executes real commands.
func New() Runner { return Cmd{} }

// Run implements Runner.
func (Cmd) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	c := osexec.CommandContext(ctx, name, args...)
	c.Dir = dir
	var out, errb bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errb
	err := c.Run()
	return out.Bytes(), errb.Bytes(), err
}

// RunInteractive runs a command with the process stdin/stdout/stderr attached
// to the terminal — the same as running the script by hand. Used for worktree
// setup when the shell wrapper passes GWT_PATH_OUT so stdout stays free.
func (Cmd) RunInteractive(ctx context.Context, dir, name string, args ...string) error {
	c := osexec.CommandContext(ctx, name, args...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RunLogged runs a command with stdout/stderr streamed to the terminal stderr.
// Stdin is /dev/null. Used when stdout must stay clean for the cd path line.
func (Cmd) RunLogged(ctx context.Context, dir, name string, args ...string) error {
	c := osexec.CommandContext(ctx, name, args...)
	c.Dir = dir
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		c.Stdin = nil
	} else {
		c.Stdin = devNull
		defer devNull.Close()
	}
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}
