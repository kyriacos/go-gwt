package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/kyriacos/go-gwt/internal/git"
)

// label is the display name for a worktree row: the directory basename, which
// for the default naming scheme is repo-branch.
func (r row) label() string {
	return filepath.Base(r.wt.Path)
}

// applyFilter recomputes m.filtered from m.filterText (case-insensitive
// substring over the row label and branch) and clamps the cursor.
func (m *model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filterText))
	m.filtered = m.filtered[:0]
	for i, r := range m.rows {
		if q == "" ||
			strings.Contains(strings.ToLower(r.label()), q) ||
			strings.Contains(strings.ToLower(r.wt.Branch), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// currentRow returns the highlighted row index into m.rows, or -1 if the list
// is empty.
func (m *model) currentRow() int {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return -1
	}
	return m.filtered[m.cursor]
}

// moveCursor adjusts the cursor by delta within the filtered range.
func (m *model) moveCursor(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// renderList renders the left pane into a fixed height: a 2-line header
// followed by a scrolling window of rows sized so the highlighted row is always
// visible. Output is clipped to width x height by the caller (fitBlock).
func (m *model) renderList(width, height int) string {
	st := m.styles
	var lines []string

	header := fmt.Sprintf("Worktrees (%d)", len(m.rows))
	if m.loading > 0 {
		header += "  " + st.spinner.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)]+" loading")
	}
	lines = append(lines, st.title.Render(header), "")

	if len(m.filtered) == 0 {
		lines = append(lines, st.dim.Render("(no worktrees match)"))
		return strings.Join(lines, "\n")
	}

	// Scrolling window: keep the cursor visible within the available rows.
	visN := height - 2 // minus the header + blank line
	if visN < 1 {
		visN = 1
	}
	start := 0
	if m.cursor >= visN {
		start = m.cursor - visN + 1
	}
	end := start + visN
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for vi := start; vi < end; vi++ {
		ri := m.filtered[vi]
		lines = append(lines, m.renderRow(m.rows[ri], vi == m.cursor, width))
	}
	return strings.Join(lines, "\n")
}

// renderRow renders exactly one line of the given visible width:
//
//	> repo-branch              ↑2 ↓0 ● abc1234 12KB
//
// The meta cluster sits on the right; the name fills the remaining space and is
// truncated with an ellipsis so the row can never wrap.
func (m *model) renderRow(r row, selected bool, width int) string {
	st := m.styles

	marker := " "
	if r.wt.IsMain {
		marker = "*"
	}

	var meta string
	switch {
	case r.loadErr != "":
		meta = "status error"
	case r.loaded:
		meta = m.renderMeta(r)
	default:
		meta = st.dim.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)])
	}

	cursor := "  "
	if selected {
		cursor = "> "
	}

	// Width budget: cursor(2) + marker(1) + space(1) + name + space(1) + meta.
	nameW := width - 5 - lipgloss.Width(meta)
	if nameW < 3 {
		nameW = 3
	}
	name := truncRunes(r.label(), nameW)

	if selected {
		// A single highlight spans the whole row; per-segment color would be
		// hidden under the selection background anyway.
		plain := fmt.Sprintf("%s%s %-*s %s", cursor, marker, nameW, name, m.metaPlain(r))
		return st.rowSelected.Render(padVis(plain, width))
	}

	nameStyled := m.styleName(r, name)
	gap := nameW - lipgloss.Width(nameStyled)
	if gap < 0 {
		gap = 0
	}
	line := cursor + st.marker.Render(marker) + " " + nameStyled +
		strings.Repeat(" ", gap) + " " + meta
	return padVis(line, width)
}

// styleName colors the worktree name by its state (mirrors the legacy tool).
func (m *model) styleName(r row, name string) string {
	st := m.styles
	switch {
	case r.wt.Bare:
		return st.stateBare.Render(name)
	case r.wt.Detached:
		return st.stateDetached.Render(name)
	case r.wt.IsMain:
		return st.stateActive.Render(name)
	case r.loaded && r.status.Upstream == "":
		return st.stateLocalOnly.Render(name)
	default:
		return name
	}
}

// truncRunes truncates s to w runes, using a trailing ellipsis when it cuts.
func truncRunes(s string, w int) string {
	rs := []rune(s)
	if len(rs) <= w {
		return s
	}
	if w <= 1 {
		return string(rs[:w])
	}
	return string(rs[:w-1]) + "…"
}

// renderMeta builds the styled ahead/behind/dirty/sha/size cluster.
func (m *model) renderMeta(r row) string {
	st := m.styles
	var parts []string

	if r.status.Ahead > 0 {
		parts = append(parts, st.ahead.Render(fmt.Sprintf("↑%d", r.status.Ahead)))
	}
	if r.status.Behind > 0 {
		parts = append(parts, st.behind.Render(fmt.Sprintf("↓%d", r.status.Behind)))
	}
	if r.status.Dirty {
		parts = append(parts, st.dirty.Render("●"))
	} else {
		parts = append(parts, st.clean.Render("○"))
	}
	if sha := shortSHA(r.wt.Head); sha != "" {
		parts = append(parts, st.sha.Render(sha))
	}
	parts = append(parts, st.size.Render(humanSize(r.size)))
	if r.hasCI {
		parts = append(parts, ciBadge(st, r.ciState))
	}
	return strings.Join(parts, " ")
}

// metaPlain is the uncolored variant used inside the selected-row highlight.
func (m *model) metaPlain(r row) string {
	if r.loadErr != "" {
		return "status error"
	}
	if !r.loaded {
		return "…"
	}
	var parts []string
	if r.status.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", r.status.Ahead))
	}
	if r.status.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", r.status.Behind))
	}
	if r.status.Dirty {
		parts = append(parts, "●")
	} else {
		parts = append(parts, "○")
	}
	if sha := shortSHA(r.wt.Head); sha != "" {
		parts = append(parts, sha)
	}
	parts = append(parts, humanSize(r.size))
	return strings.Join(parts, " ")
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// humanSize formats a byte count compactly (B, KB, MB, GB) using 1024 steps.
func humanSize(n int64) string {
	if n <= 0 {
		return "-"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// rowsFromWorktrees builds initial (unloaded) rows from a worktree list.
func rowsFromWorktrees(wts []git.Worktree) []row {
	rows := make([]row, len(wts))
	for i, w := range wts {
		rows[i] = row{wt: w}
	}
	return rows
}
