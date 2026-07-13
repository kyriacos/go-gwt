package tui

import (
	"image"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"

	"github.com/kyriacos/go-gwt/internal/changelog"
	"github.com/kyriacos/go-gwt/internal/version"
)

func (m *model) changelogLines() []string {
	if changelog.Markdown == "" {
		return []string{"(changelog not embedded)"}
	}
	return strings.Split(strings.TrimRight(changelog.Markdown, "\n"), "\n")
}

// joinFooterLine places left and right text on one row, right-aligning right when
// there is room.
func joinFooterLine(left, right string, width int) string {
	if right == "" {
		return left
	}
	if left == "" {
		return alignRight(right, width)
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func alignRight(s string, width int) string {
	pad := width - lipgloss.Width(s)
	if pad < 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

func (m *model) overlayOpen() bool {
	switch m.mode {
	case modeHelp, modeChangelog, modePrompt, modeConfirm:
		return true
	default:
		return false
	}
}

func (m *model) geom() (w, h int) {
	w, h = m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	return w, h
}

func (m *model) cacheOverlayBackdrop() {
	m.overlayBackdrop = m.dashboardContent()
}

func (m *model) clearOverlayBackdrop() {
	m.overlayBackdrop = ""
}

func (m *model) animating() bool {
	if m.overlayOpen() {
		return false
	}
	if m.loading > 0 {
		return true
	}
	if m.mode == modePR && !m.prLoaded {
		return true
	}
	if m.preview != nil && m.currentRow() >= 0 {
		path := m.rows[m.currentRow()].wt.Path
		if m.previewPath != path || (m.previewText == "" && m.previewErr == "") {
			return true
		}
	}
	for _, r := range m.rows {
		if !r.loaded && r.loadErr == "" {
			return true
		}
	}
	return false
}

// overlayModalSize picks a centered modal box that fits the terminal.
func (m *model) overlayModalSize(minW, minH int) (w, h int) {
	w = m.width * 4 / 5
	if w < minW {
		w = minW
	}
	if w > m.width-2 {
		w = m.width - 2
	}
	if w < 20 {
		w = 20
	}

	h = m.height * 4 / 5
	if h < minH {
		h = minH
	}
	if h > m.height-2 {
		h = m.height - 2
	}
	if h < 8 {
		h = 8
	}
	return w, h
}

// composeOverlay draws the dashboard dimmed full-screen with a centered modal on
// top. Terminals cannot blur pixels; faint styling on the frozen backdrop
// approximates a frosted scrim.
func (m *model) composeOverlay(modal string) string {
	w, h := m.geom()

	bg := m.overlayBackdrop
	if bg == "" {
		bg = m.dashboardContent()
	}
	bgLines := strings.Split(normalizeView(bg, w, h), "\n")
	for i := range bgLines {
		bgLines[i] = m.styles.scrim.Render(padVis(bgLines[i], w))
	}
	scrim := strings.Join(bgLines, "\n")

	buf := cellbuf.NewBuffer(w, h)
	cellbuf.SetContent(buf, scrim)

	modal = m.styles.overlayModal.Render(modal)
	mw := lipgloss.Width(modal)
	mh := lipgloss.Height(modal)
	x := (w - mw) / 2
	y := (h - mh) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	rect := image.Rect(x, y, x+mw, y+mh)
	cellbuf.SetContentRect(buf, modal, rect)

	return strings.ReplaceAll(cellbuf.Render(buf), "\r\n", "\n")
}

func (m *model) helpModalContent() string {
	st := m.styles
	modalW, modalH := m.overlayModalSize(52, 18)
	innerW := modalW - 4
	innerH := modalH - 8
	if innerW < 20 {
		innerW = 20
	}
	if innerH < 6 {
		innerH = 6
	}

	title := st.modalTitle.Render("Help")
	sub := st.dim.Render(version.String())
	body := fitBlock(dashboardHelp, innerW, innerH)
	hint := joinFooterLine(
		st.help.Render("esc ? q close"),
		st.dim.Render(version.Short()),
		innerW,
	)
	return title + "\n" + sub + "\n\n" + body + "\n\n" + hint
}

func (m *model) changelogModalContent() string {
	st := m.styles
	modalW, modalH := m.overlayModalSize(52, 14)
	innerW := modalW - 4
	innerH := modalH - 8
	if innerW < 20 {
		innerW = 20
	}
	if innerH < 4 {
		innerH = 4
	}

	lines := m.changelogLines()
	start := m.scrollOffset
	end := start + innerH
	if end > len(lines) {
		end = len(lines)
	}
	chunk := strings.Join(lines[start:end], "\n")

	title := st.modalTitle.Render("Changelog")
	sub := st.dim.Render(version.Short())
	body := fitBlock(chunk, innerW, innerH)
	hint := joinFooterLine(
		st.help.Render("↑/k ↓/j scroll  esc c q close"),
		st.dim.Render(version.Short()),
		innerW,
	)
	return title + "\n" + sub + "\n\n" + body + "\n\n" + hint
}

func (m *model) scrollVisibleLines() int {
	_, modalH := m.overlayModalSize(52, 14)
	innerH := modalH - 8
	if innerH < 4 {
		return 4
	}
	return innerH
}
