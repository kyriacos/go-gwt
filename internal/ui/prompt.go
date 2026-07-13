package ui

import (
	"fmt"
	"os"
	"strings"
)

// Confirm asks a yes/no question on the controlling terminal and returns the
// answer. The prompt is written to /dev/tty (not stdout, which is reserved for
// paths). When no terminal is available it returns def without prompting, so
// callers behave sanely under `cd "$(gwt ...)"` and in CI.
func Confirm(question string, def bool) bool {
	ResetTTY()
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return def
	}
	defer tty.Close()

	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(tty, "%s %s ", render(styleWarn, question), suffix)

	line, err := ReadTTYLine(tty)
	if err != nil && line == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		DrainTTY()
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

// HasTTY reports whether a controlling terminal is available for prompting.
func HasTTY() bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = tty.Close()
	return true
}
