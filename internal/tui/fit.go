package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// truncVis truncates s to at most w visible columns, preserving ANSI SGR escape
// sequences (so colored text stays valid) and appending a reset at the cut.
func truncVis(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	var b strings.Builder
	rs := []rune(s)
	vis, i, sawEsc := 0, 0, false
	for i < len(rs) {
		r := rs[i]
		if r == 0x1b { // ESC: copy the whole escape sequence without counting it
			sawEsc = true
			j := i + 1
			for j < len(rs) && !((rs[j] >= 'a' && rs[j] <= 'z') || (rs[j] >= 'A' && rs[j] <= 'Z')) {
				j++
			}
			if j < len(rs) {
				j++ // include the terminating letter
			}
			b.WriteString(string(rs[i:j]))
			i = j
			continue
		}
		if vis >= w {
			break
		}
		b.WriteRune(r)
		vis++
		i++
	}
	if sawEsc {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// padVis truncates s to w visible columns then right-pads with spaces so its
// visible width is exactly w.
func padVis(s string, w int) string {
	s = truncVis(s, w)
	if pad := w - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// fitBlock forces content into exactly width x height: every line is clipped
// and padded to width, and the block is clipped/padded to height lines. This is
// what keeps the panes a constant size regardless of content, so the screen
// never grows or shrinks as the selection or preview changes.
func fitBlock(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	out := make([]string, height)
	blank := strings.Repeat(" ", width)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out[i] = padVis(lines[i], width)
		} else {
			out[i] = blank
		}
	}
	return strings.Join(out, "\n")
}

// normalizeView pads or clips content to exactly width x height lines. Carriage
// returns from cellbuf are stripped so the render loop can split on newlines.
func normalizeView(content string, width, height int) string {
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	out := make([]string, height)
	blank := strings.Repeat(" ", width)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out[i] = padVis(lines[i], width)
		} else {
			out[i] = blank
		}
	}
	return strings.Join(out, "\n")
}
