package tui

import (
	"fmt"
	"strings"

	"github.com/kyriacos/go-gwt/internal/gh"
)

// ciBadge renders a one-glyph CI indicator for a PR/branch.
func ciBadge(st styles, state gh.CIState) string {
	switch state {
	case gh.CIPassing:
		return st.ciPassing.Render("✓")
	case gh.CIFailing:
		return st.ciFailing.Render("✗")
	case gh.CIPending:
		return st.ciPending.Render("•")
	default:
		return st.dim.Render("·")
	}
}

// renderPR renders the PR list view.
func (m *model) renderPR(width, height int) string {
	st := m.styles
	var b strings.Builder
	b.WriteString(st.title.Render("Pull Requests"))
	b.WriteString("\n\n")

	if !m.ghAvailable {
		b.WriteString(st.warnText.Render("gh is not available"))
		return b.String()
	}
	if m.prErr != "" {
		b.WriteString(st.errText.Render(m.prErr))
		return b.String()
	}
	if !m.prLoaded {
		b.WriteString(st.dim.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)] + " loading PRs…"))
		return b.String()
	}
	if len(m.prRows) == 0 {
		b.WriteString(st.dim.Render("no open pull requests"))
		b.WriteString("\n\n")
		b.WriteString(st.help.Render("esc back  q quit"))
		return b.String()
	}

	for i, pr := range m.prRows {
		cursor := "  "
		line := fmt.Sprintf("#%-4d %s", pr.Number, pr.Title)
		if pr.Draft {
			line = st.prDraft.Render(line + " (draft)")
		}
		meta := st.dim.Render(fmt.Sprintf("@%s  %s", pr.Author, pr.Branch))
		full := fmt.Sprintf("%s%s  %s", cursor, line, meta)
		if i == m.prCursor {
			full = st.rowSelected.Render(fmt.Sprintf("> #%-4d %s", pr.Number, pr.Title))
		}
		b.WriteString(truncVis(full, width))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(st.help.Render("enter checkout  esc back  q quit"))
	return b.String()
}

// currentPR returns the highlighted PR, or nil.
func (m *model) currentPR() *gh.PR {
	if m.prCursor < 0 || m.prCursor >= len(m.prRows) {
		return nil
	}
	return &m.prRows[m.prCursor]
}
