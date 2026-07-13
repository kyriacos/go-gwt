// Package changelog embeds the project changelog for in-app display (TUI).
package changelog

import _ "embed"

// Markdown is the contents of CHANGELOG.md at build time.
//
//go:embed CHANGELOG.md
var Markdown string
