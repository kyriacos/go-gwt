package tui

// renderPreview renders the right pane: a "Log" header plus the cached preview
// text for the highlighted worktree (typically `git log --oneline --graph`), or
// a status line while loading / on error. The caller (fitBlock) clips the
// result to the pane's exact width x height, ANSI-aware, so it never wraps.
func (m *model) renderPreview(width, height int) string {
	st := m.styles
	header := st.title.Render("Log") + "\n\n"

	if m.preview == nil {
		return header + st.dim.Render("preview disabled")
	}
	if m.currentRow() < 0 {
		return header + st.dim.Render("no selection")
	}

	switch {
	case m.previewErr != "":
		return header + st.errText.Render(m.previewErr)
	case m.previewText == "":
		return header + st.dim.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)]+" loading preview…")
	default:
		return header + m.previewText
	}
}
