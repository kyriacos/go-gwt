package ui

import (
	"fmt"
	"os"
)

// PathOutEnv is set by the shell wrapper so gwt can emit the cd path without
// stdout being captured in command substitution (which breaks setup progress).
const PathOutEnv = "GWT_PATH_OUT"

// pathOutEnv is the legacy unexported alias used within this package.
const pathOutEnv = PathOutEnv

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

func writeMachineLine(line string) {
	if out := os.Getenv(pathOutEnv); out != "" {
		_ = os.WriteFile(out, []byte(line), 0o600)
		return
	}
	fmt.Fprint(os.Stdout, line)
}

// Path emits the worktree path for the shell wrapper to cd. When GWT_PATH_OUT
// is set, the path is written there instead of stdout so long-running setup can
// stream to the terminal while the wrapper waits.
func Path(p string) {
	writeMachineLine(p + "\n")
}

// Populate emits a GWT_POPULATE line so the shell wrapper can park the
// suggested command in the line buffer for review before running.
func Populate(cmd string) {
	writeMachineLine("GWT_POPULATE:" + cmd + "\n")
}

// Bold returns text styled bold (no-op when color is disabled).
func Bold(text string) string { return render(styleBold, text) }
