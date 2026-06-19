package tui

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// This file is a minimal, self-contained reimplementation of the slice of the
// Bubble Tea runtime that the dashboard needs: a Model with Init/Update/View,
// asynchronous Cmds that produce Msgs, command batching, and a key-event input
// reader. It exists only because bubbletea is not in go.mod and adding it is
// out of scope. The public surface (Model, Msg, Cmd, KeyMsg, Batch, Quit,
// program) mirrors bubbletea names so the code reads the same and can be ported
// by swapping this file for the real library.

// Msg is any value delivered to Update.
type Msg interface{}

// Cmd performs work off the update loop and returns a Msg (or nil). Cmds run in
// their own goroutines; returning a Msg re-enters Update.
type Cmd func() Msg

// Model is the standard Elm-style model.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// quitMsg is delivered by Quit; the program stops after the next render.
type quitMsg struct{}

// Quit is a Cmd that tells the program to stop.
func Quit() Msg { return quitMsg{} }

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

// KeyMsg is a decoded key press. For printable keys, Runes holds the rune(s)
// and Type is KeyRunes. Special keys set Type and leave Runes empty.
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
// (the tty). It is intentionally small: a single message channel, a render on
// every step, and cooked-mode-tolerant input parsing.
type program struct {
	model Model
	in    io.Reader
	out   io.Writer

	msgs chan Msg
	mu   sync.Mutex // serializes writes to out
}

func newProgram(m Model, in io.Reader, out io.Writer) *program {
	return &program{model: m, in: in, out: out, msgs: make(chan Msg, 64)}
}

// send enqueues a message from any goroutine.
func (p *program) send(m Msg) {
	if m == nil {
		return
	}
	p.msgs <- m
}

// exec runs a Cmd in a goroutine and feeds its result back in. batchMsg fans
// out into one goroutine per child Cmd.
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

// run is the event loop. It returns when a quitMsg is processed.
func (p *program) run() error {
	if c := p.model.Init(); c != nil {
		p.exec(c)
	}
	p.render()

	// Input reader posts KeyMsgs until the channel is closed by quit.
	stop := make(chan struct{})
	go p.readInput(stop)
	defer close(stop)

	for msg := range p.msgs {
		if _, ok := msg.(quitMsg); ok {
			p.render() // final frame
			return nil
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

func (p *program) render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	// clear screen + home, then draw.
	fmt.Fprint(p.out, "\x1b[2J\x1b[H")
	fmt.Fprint(p.out, p.model.View())
}

// readInput decodes bytes from p.in into KeyMsgs. It understands the small set
// of escape sequences the dashboard uses (arrows). It is robust to cooked-mode
// line input (used by tests via a bytes buffer) as well as raw tty bytes.
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
