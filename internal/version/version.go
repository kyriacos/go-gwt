// Package version holds build metadata injected at link time (see main.go and
// .goreleaser.yaml). When commit is unset, it falls back to the VCS revision
// embedded by the Go toolchain (go build -buildvcs).
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	// Version is the release tag or "dev" for local builds.
	Version = "dev"
	// Commit is the full git SHA at build time.
	Commit = "none"
	// Date is the build timestamp (RFC3339).
	Date = "unknown"
)

// Set records link-time metadata from main.
func Set(v, commit, date string) {
	Version = v
	Commit = commit
	Date = date
	if Commit == "" || Commit == "none" {
		Commit = vcsRevision()
	}
}

// String is the full version line for `gwt version` and `--version`.
func String() string {
	return fmt.Sprintf("gwt %s (commit %s, built %s)", Version, ShortCommit(), Date)
}

// Short is a compact label for the TUI footer and help header.
func Short() string {
	return fmt.Sprintf("gwt %s · %s", Version, ShortCommit())
}

// ShortCommit returns a 7-character commit id, or "dev" when unknown.
func ShortCommit() string {
	c := Commit
	if c == "" || c == "none" {
		c = vcsRevision()
	}
	if c == "" {
		return "dev"
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return strings.TrimSpace(s.Value)
		}
	}
	return ""
}
