// Package ui holds non-TUI presentation: color policy, styled stderr output,
// fatal errors, and terminal prompts. stdout is reserved for machine output
// (worktree paths), so everything here writes to stderr or /dev/tty.
package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Mode mirrors config.ColorMode without importing config (avoids a cycle).
type Mode int

const (
	Auto Mode = iota
	Always
	Never
)

var (
	enabled bool

	styleErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleDim  = lipgloss.NewStyle().Faint(true)
	styleBold = lipgloss.NewStyle().Bold(true)
)

func init() { SetColor(Auto) }

// SetColor applies the color policy. Auto enables color when stderr is a
// terminal and NO_COLOR is unset. Because gwt's colored output targets stderr,
// we set lipgloss's profile explicitly rather than letting it probe stdout.
func SetColor(m Mode) {
	switch m {
	case Always:
		enabled = true
	case Never:
		enabled = false
	default:
		_, noColor := os.LookupEnv("NO_COLOR")
		enabled = !noColor && IsTerminal(os.Stderr)
	}
	if enabled {
		lipgloss.SetColorProfile(termenv.ANSI256)
	} else {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

// IsTerminal reports whether f is a character device (a terminal).
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func render(s lipgloss.Style, text string) string {
	if !enabled {
		return text
	}
	return s.Render(text)
}
