// Command gwt is a git worktree helper with a Charm TUI and gh integration.
package main

import "github.com/kyriacos/go-gwt/cmd"

// Populated via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
