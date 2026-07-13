package tui

import "strings"

// binding is one keybinding: the keys that trigger it plus help text. It mirrors
// the shape of bubbles/key.Binding (Keys + Help{Key,Desc}) so the help bar reads
// the same way.
type binding struct {
	keys []string
	key  string // help label, e.g. "enter"
	desc string // help description, e.g. "select"
	// enabled gates whether the binding shows in help and fires; used to hide
	// the PR key when gh is unavailable.
	enabled bool
}

func newBinding(help, desc string, enabled bool, keys ...string) binding {
	return binding{keys: keys, key: help, desc: desc, enabled: enabled}
}

// matches reports whether s (a KeyMsg.String()) triggers this binding.
func (b binding) matches(s string) bool {
	if !b.enabled {
		return false
	}
	for _, k := range b.keys {
		if k == s {
			return true
		}
	}
	return false
}

// keyMap is the full set of dashboard bindings.
type keyMap struct {
	up      binding
	down    binding
	enter   binding
	filter  binding
	refresh binding
	newWT   binding
	remove  binding
	removeD binding
	pr      binding
	open    binding
	help    binding
	changelog binding
	quit    binding
	confirm binding // y in modal
	cancel  binding // n/esc in modal
}

// defaultKeyMap builds the bindings. ghAvailable gates the PR binding.
func defaultKeyMap(ghAvailable bool) keyMap {
	return keyMap{
		up:      newBinding("↑/k", "up", true, "up", "k"),
		down:    newBinding("↓/j", "down", true, "down", "j"),
		enter:   newBinding("enter", "select", true, "enter"),
		filter:  newBinding("/", "filter", true, "/"),
		refresh: newBinding("r", "refresh", true, "r"),
		newWT:   newBinding("n", "new", true, "n"),
		remove:  newBinding("d", "remove", true, "d"),
		removeD: newBinding("D", "remove+branch", true, "D"),
		pr:      newBinding("p", "PRs", ghAvailable, "p"),
		open:      newBinding("o", "open", true, "o"),
		help:      newBinding("?", "help", true, "?"),
		changelog: newBinding("c", "changelog", true, "c"),
		quit:      newBinding("q", "quit", true, "q", "esc"),
		confirm: newBinding("y", "yes", true, "y"),
		cancel:  newBinding("n", "no", true, "n", "esc"),
	}
}

// helpBar renders the one-line help string for the list view.
func (k keyMap) helpBar(st styles) string {
	parts := []binding{k.enter, k.filter, k.refresh, k.newWT, k.remove, k.removeD, k.pr, k.open, k.help, k.changelog, k.quit}
	var b strings.Builder
	first := true
	for _, bd := range parts {
		if !bd.enabled {
			continue
		}
		if !first {
			b.WriteString("  ")
		}
		first = false
		b.WriteString(bd.key)
		b.WriteString(" ")
		b.WriteString(bd.desc)
	}
	return st.help.Render(b.String())
}
