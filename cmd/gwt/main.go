// Command gwt is a git worktree helper with a Charm TUI and gh integration.
package main

import (
	"github.com/kyriacos/go-gwt/cmd"
	ver "github.com/kyriacos/go-gwt/internal/version"
)

// Populated via -ldflags at release time; see .goreleaser.yaml.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ver.Set(version, commit, date)
	cmd.Execute()
}
