package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"golang.org/x/term"
)

// ResetTTY best-effort restores the controlling terminal after a full-screen TUI
// or other raw-mode UI. It leaves the alt screen, shows the cursor, and runs
// stty sane so subsequent line prompts read Enter correctly.
func ResetTTY() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer tty.Close()

	_, _ = fmt.Fprint(tty, "\x1b[?25h\x1b[?1049l\x1b[0m\r\n")
	if !term.IsTerminal(int(tty.Fd())) {
		return
	}
	c := exec.Command("stty", "sane")
	c.Stdin = tty
	_ = c.Run()
}

// ReadTTYLine reads one line from the controlling terminal. It accepts either
// LF or CR as the line terminator so prompts still work when the terminal is in
// raw mode (common after the built-in TUI exits).
func ReadTTYLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	var line []byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			return string(line), err
		}
		if b == '\n' || b == '\r' {
			return string(line), nil
		}
		line = append(line, b)
	}
}

// DrainTTY discards any already-typed input waiting on the controlling
// terminal. Call after a yes/no prompt so a stray Enter cannot reach setup.
func DrainTTY() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer tty.Close()
	buf := make([]byte, 256)
	for {
		_ = tty.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		n, err := tty.Read(buf)
		if n == 0 || err != nil {
			return
		}
	}
}

// StdoutIsTerminal reports whether process stdout is an interactive character
// device. When false (e.g. under the shell wrapper's command substitution),
// child setup commands must not write progress to stdout.
func StdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
