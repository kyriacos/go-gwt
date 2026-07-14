package ui

import (
	"fmt"
	"os"
	"strings"
)

// Confirm asks a yes/no question on stderr and reads the answer from /dev/tty.
func Confirm(question string, def bool) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return def
	}
	defer tty.Close()

	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(os.Stderr, "%s %s ", render(styleWarn, question), suffix)

	line, err := ReadTTYLine(tty)
	if err != nil && line == "" {
		return def
	}
	if line != "" {
		fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(line))
	}
	return parseConfirm(line, def)
}

func parseConfirm(line string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
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
