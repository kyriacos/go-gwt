package ui

import (
	"fmt"
	"os"
)

// Err prints a diagnostic line to stderr.
func Err(format string, a ...any) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
}

// Die prints an error to stderr and exits with status 1.
func Die(format string, a ...any) {
	fmt.Fprintln(os.Stderr, render(styleErr, "gwt: ")+fmt.Sprintf(format, a...))
	os.Exit(1)
}

// OK prints a success line (green) to stderr.
func OK(format string, a ...any) {
	fmt.Fprintln(os.Stderr, render(styleOK, fmt.Sprintf(format, a...)))
}

// Warn prints a warning line (yellow) to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, render(styleWarn, fmt.Sprintf(format, a...)))
}

// Info prints an informational line (cyan) to stderr.
func Info(format string, a ...any) {
	fmt.Fprintln(os.Stderr, render(styleInfo, fmt.Sprintf(format, a...)))
}

// Dim prints a de-emphasized line to stderr.
func Dim(format string, a ...any) {
	fmt.Fprintln(os.Stderr, render(styleDim, fmt.Sprintf(format, a...)))
}

// Path prints a worktree path to stdout. This is the machine-readable contract
// the shell wrapper reads to cd; it must be the only thing on stdout.
func Path(p string) {
	fmt.Fprintln(os.Stdout, p)
}

// Bold returns text styled bold (no-op when color is disabled).
func Bold(text string) string { return render(styleBold, text) }
