package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// This file is a minimal, self-contained reimplementation of the slice of the
// Bubble Tea runtime that the dashboard needs: a Model with Init/Update/View,
// asynchronous Cmds that produce Msgs, command batching, a key-event input
// reader, raw-mode terminal handling, real terminal-size detection, and resize
// support. It exists so the dashboard has no init-time terminal probe (the real
// bubbletea blocks ~5s on an OSC 11 query at package init). The public surface
// mirrors bubbletea names so it can be ported by swapping this file.

// Msg is any value delivered to Update.
type Msg interface{}

// Cmd performs work off the update loop and returns a Msg (or nil).
type Cmd func() Msg

// Model is the standard Elm-style model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// quitMsg is delivered by Quit; the program stops after processing it.
type quitMsg struct{}

// Quit is a Cmd that tells the program to stop.
func Quit() Msg { return quitMsg{} }

// windowSizeMsg reports the terminal size at startup and on resize.
type windowSizeMsg struct{ width, height int }

// batchMsg carries several Cmds to be run.
type batchMsg []Cmd

// Batch groups Cmds so they run concurrently. nil Cmds are dropped.
func Batch(cmds ...Cmd) Cmd {
	var out []Cmd
	for _, c := range cmds {
		if c != nil {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	if len(out) == 1 {
		return out[0]
	}
	return func() Msg { return batchMsg(out) }
}

// KeyMsg is a decoded key press.
type KeyMsg struct {
	Type  keyType
	Runes []rune
}

func (k KeyMsg) String() string {
	if k.Type == keyRunes {
		return string(k.Runes)
	}
	return keyNames[k.Type]
}

type keyType int

const (
	keyRunes keyType = iota
	keyEnter
	keyEsc
	keyTab
	keyBackspace
	keyUp
	keyDown
	keyLeft
	keyRight
	keyCtrlC
	keySpace
)

var keyNames = map[keyType]string{
	keyEnter:     "enter",
	keyEsc:       "esc",
	keyTab:       "tab",
	keyBackspace: "backspace",
	keyUp:        "up",
	keyDown:      "down",
	keyLeft:      "left",
	keyRight:     "right",
	keyCtrlC:     "ctrl+c",
	keySpace:     "space",
}

// program drives a Model. It renders to out (stderr) and reads keys from in
// (the tty). When tty is non-nil and a real terminal, it switches to raw mode,
// uses the alternate screen, reads the real size, and watches for resizes.
type program struct {
	model Model
	in    io.Reader
	out   io.Writer
	tty   *os.File // controlling terminal; nil in tests (in is a buffer)

	msgs chan Msg
	mu   sync.Mutex // serializes writes to out
}

func newProgram(m Model, in io.Reader, out io.Writer, tty *os.File) *program {
	return &program{model: m, in: in, out: out, tty: tty, msgs: make(chan Msg, 64)}
}

func (p *program) send(m Msg) {
	if m == nil {
		return
	}
	p.msgs <- m
}

// exec runs a Cmd in a goroutine and feeds its result back in.
func (p *program) exec(c Cmd) {
	if c == nil {
		return
	}
	go func() {
		msg := c()
		if msg == nil {
			return
		}
		if b, ok := msg.(batchMsg); ok {
			for _, child := range b {
				p.exec(child)
			}
			return
		}
		p.send(msg)
	}()
}

// raw reports whether we're driving a real terminal (vs a test buffer).
func (p *program) raw() bool {
	return p.tty != nil && term.IsTerminal(int(p.tty.Fd()))
}

// size returns the current terminal size, preferring the render target.
func (p *program) size() (int, int) {
	if f, ok := p.out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if w, h, err := term.GetSize(int(f.Fd())); err == nil {
			return w, h
		}
	}
	if p.tty != nil {
		if w, h, err := term.GetSize(int(p.tty.Fd())); err == nil {
			return w, h
		}
	}
	return 0, 0
}

// run is the event loop. It sets up raw mode and the alternate screen, seeds the
// model with the real terminal size, then renders on every message until quit.
func (p *program) run() error {
	if p.raw() {
		if old, err := term.MakeRaw(int(p.tty.Fd())); err == nil {
			defer func() { _ = term.Restore(int(p.tty.Fd()), old) }()
		}
		fmt.Fprint(p.out, "\x1b[?1049h\x1b[?25l") // alt screen + hide cursor
		defer fmt.Fprint(p.out, "\x1b[?25h\x1b[?1049l")
	}

	// Seed the size before the first render so the layout fills the screen.
	if w, h := p.size(); w > 0 && h > 0 {
		p.model, _ = p.model.Update(windowSizeMsg{width: w, height: h})
	}

	if c := p.model.Init(); c != nil {
		p.exec(c)
	}
	p.render()

	stopWinch := p.watchResize()
	defer stopWinch()

	stop := make(chan struct{})
	go p.readInput(stop)
	defer close(stop)

	for msg := range p.msgs {
		if _, ok := msg.(quitMsg); ok {
			return nil // leaving the alt screen restores the prior terminal
		}
		var cmd Cmd
		p.model, cmd = p.model.Update(msg)
		p.render()
		if cmd != nil {
			p.exec(cmd)
		}
	}
	return nil
}

// watchResize sends a windowSizeMsg on SIGWINCH. Returns a stop func.
func (p *program) watchResize() func() {
	if p.tty == nil {
		return func() {}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ch:
				if w, h := p.size(); w > 0 && h > 0 {
					p.send(windowSizeMsg{width: w, height: h})
				}
			}
		}
	}()
	return func() { signal.Stop(ch); close(done) }
}

// render draws the current frame. In raw mode it redraws in place (home, then
// clear-to-EOL per line, clear-below at the end) using CR/LF, which avoids the
// full-screen clear that causes flicker. Outside raw mode (tests) it falls back
// to a simple clear-and-draw.
func (p *program) render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	view := p.model.View()
	if !p.raw() {
		fmt.Fprint(p.out, "\x1b[2J\x1b[H"+view)
		return
	}
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, line := range strings.Split(view, "\n") {
		if i > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString(line)
		b.WriteString("\x1b[K") // clear to end of line
	}
	b.WriteString("\x1b[J") // clear everything below the frame
	fmt.Fprint(p.out, b.String())
}

// readInput decodes bytes from p.in into KeyMsgs, including the arrow-key escape
// sequences. In raw mode each keypress arrives immediately.
func (p *program) readInput(stop <-chan struct{}) {
	r := bufio.NewReader(p.in)
	for {
		select {
		case <-stop:
			return
		default:
		}
		b, err := r.ReadByte()
		if err != nil {
			return
		}
		switch b {
		case '\r', '\n':
			p.send(KeyMsg{Type: keyEnter})
		case 0x7f, 0x08:
			p.send(KeyMsg{Type: keyBackspace})
		case '\t':
			p.send(KeyMsg{Type: keyTab})
		case 0x03:
			p.send(KeyMsg{Type: keyCtrlC})
		case ' ':
			p.send(KeyMsg{Type: keySpace, Runes: []rune{' '}})
		case 0x1b: // ESC or CSI sequence
			n, _ := r.Peek(2)
			if len(n) >= 2 && n[0] == '[' {
				_, _ = r.ReadByte() // consume '['
				dir, _ := r.ReadByte()
				switch dir {
				case 'A':
					p.send(KeyMsg{Type: keyUp})
				case 'B':
					p.send(KeyMsg{Type: keyDown})
				case 'C':
					p.send(KeyMsg{Type: keyRight})
				case 'D':
					p.send(KeyMsg{Type: keyLeft})
				}
				continue
			}
			p.send(KeyMsg{Type: keyEsc})
		default:
			if b >= 0x20 {
				p.send(KeyMsg{Type: keyRunes, Runes: []rune{rune(b)}})
			}
		}
	}
}
