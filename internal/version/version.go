// Package version holds build metadata injected at link time (see main.go and
// .goreleaser.yaml). When commit is unset, it falls back to the VCS revision
// embedded by the Go toolchain (go build -buildvcs).
package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

var (
	// Version is the release tag or "dev" for local builds.
	Version = "dev"
	// Commit is the full git SHA at build time.
	Commit = "none"
	// Date is the build timestamp (RFC3339).
	Date = "unknown"
	// Binary is the resolved path of the running executable.
	Binary = ""
)

// Set records link-time metadata from main.
func Set(v, commit, date string) {
	Version = v
	Commit = commit
	Date = date
	if Commit == "" || Commit == "none" {
		Commit = vcsRevision()
	}
	if date == "" || date == "unknown" {
		Date = time.Now().UTC().Format(time.RFC3339)
	}
	Binary = executablePath()
}

// String is the full version line for `gwt version` and `--version`.
func String() string {
	line := fmt.Sprintf("gwt %s (commit %s, built %s)", Version, ShortCommit(), Date)
	if Binary != "" {
		line += "\n  binary: " + Binary
	}
	return line
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
		c = c[:7]
	}
	if vcsDirty() {
		c += "+"
	}
	return c
}

func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return path
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

func vcsDirty() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.modified" {
			return strings.TrimSpace(s.Value) == "true"
		}
	}
	return false
}
