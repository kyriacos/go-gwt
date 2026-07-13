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

// RunTTY runs a command with stdout/stderr attached to /dev/tty when available
// so progress is visible while gwt's stdout is captured by the shell wrapper.
// Stdin is attached to /dev/null so stray Enter cannot interrupt setup; use
// Ctrl+C. When /dev/tty is unavailable, both streams go to stderr — never to
// stdout, which may be a pipe that would block once full.
func (Cmd) RunTTY(ctx context.Context, dir, name string, args ...string) error {
	c := osexec.CommandContext(ctx, name, args...)
	c.Dir = dir
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		c.Stdin = nil
	} else {
		c.Stdin = devNull
		defer devNull.Close()
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		defer tty.Close()
		c.Stdout = tty
		c.Stderr = tty
	} else {
		c.Stdout = os.Stderr
		c.Stderr = os.Stderr
	}
	err = c.Run()
	return err
}
