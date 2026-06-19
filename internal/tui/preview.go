package tui

import "strings"

// renderPreview renders the right pane: the cached preview text for the
// highlighted worktree (typically `git log --oneline --graph`), or a status
// line while loading / on error.
func (m *model) renderPreview(width, height int) string {
	st := m.styles
	if m.preview == nil {
		return st.dim.Render("preview disabled")
	}
	ri := m.currentRow()
	if ri < 0 {
		return st.dim.Render("no selection")
	}

	var body string
	switch {
	case m.previewErr != "":
		body = st.errText.Render(m.previewErr)
	case m.previewText == "":
		body = st.dim.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)] + " loading preview…")
	default:
		body = m.previewText
	}

	header := st.title.Render("Log")
	return header + "\n\n" + clip(body, width, height-3)
}

// clip truncates body to fit width x lines (no wrapping; long lines are cut).
func clip(body string, width, lines int) string {
	if lines < 1 {
		lines = 1
	}
	rows := strings.Split(body, "\n")
	if len(rows) > lines {
		rows = rows[:lines]
	}
	for i, r := range rows {
		if width > 0 && len([]rune(r)) > width {
			rows[i] = string([]rune(r)[:width])
		}
	}
	return strings.Join(rows, "\n")
}
