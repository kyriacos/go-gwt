package tui

import (
	"fmt"
	"path/filepath"
	"strings"

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

// renderList renders the left pane: header + one line per filtered row.
func (m *model) renderList(width int) string {
	var b strings.Builder
	st := m.styles

	header := fmt.Sprintf("Worktrees (%d)", len(m.rows))
	if m.loading > 0 {
		header += "  " + st.spinner.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)]+" loading")
	}
	b.WriteString(st.title.Render(header))
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(st.dim.Render("  (no worktrees match)"))
		return b.String()
	}

	for vi, ri := range m.filtered {
		line := m.renderRow(m.rows[ri], vi == m.cursor, width)
		b.WriteString(line)
		if vi < len(m.filtered)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// renderRow renders one worktree row:
//
//   - repo-branch        ↑2 ↓0   ●  abc1234   12 MB
func (m *model) renderRow(r row, selected bool, width int) string {
	st := m.styles

	marker := " "
	if r.wt.IsMain {
		marker = "*"
	}

	// name colored by worktree state.
	name := r.label()
	var nameStyled string
	switch {
	case r.wt.Bare:
		nameStyled = st.stateBare.Render(name + " (bare)")
	case r.wt.Detached:
		nameStyled = st.stateDetached.Render(name + " (detached)")
	case r.wt.IsMain:
		nameStyled = st.stateActive.Render(name)
	case r.loaded && r.status.Upstream == "" && !r.wt.Detached:
		nameStyled = st.stateLocalOnly.Render(name)
	default:
		nameStyled = name
	}

	// ahead/behind + dirty + sha + size, only once loaded.
	var meta string
	if r.loadErr != "" {
		meta = st.errText.Render("status error")
	} else if r.loaded {
		meta = m.renderMeta(r)
	} else {
		meta = st.dim.Render(spinnerFrames[m.spinFrame%len(spinnerFrames)])
	}

	left := fmt.Sprintf("%s %s", st.marker.Render(marker), nameStyled)
	line := fmt.Sprintf("%-40s %s", left, meta)

	if selected {
		// Render selection over the (possibly already-styled) label by
		// re-styling the visible label portion; keep it simple: highlight the
		// whole line.
		plain := fmt.Sprintf("%s %-38s %s", marker, name, m.metaPlain(r))
		return st.rowSelected.Render("> " + plain)
	}
	return "  " + line
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
