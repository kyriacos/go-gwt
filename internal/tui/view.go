package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the whole screen for the current mode.
func (m *model) View() string {
	if m.quitting {
		return "" // leave a clean terminal; caller prints the path to stdout
	}

	switch m.mode {
	case modePR:
		return m.viewPR()
	default:
		return m.viewMain()
	}
}

// footerH is the fixed number of lines reserved below the panes (one status
// line + one help/filter line). Keeping it constant — together with
// fixed-height panes — is what stops the screen from jumping as the selection
// or preview changes.
const footerH = 2

// viewMain renders the list + preview split plus a fixed-height footer. Panes
// are sized to fill the viewport exactly and their content is clipped to that
// size, so the layout never grows or shrinks with content.
func (m *model) viewMain() string {
	// Inner content height of each pane (minus the 2 border rows), chosen so
	// panes(border) + footer == terminal height, always.
	innerH := m.height - footerH - 2
	if innerH < 3 {
		innerH = 3
	}

	// Inner content widths. Each pane adds 2 columns of border; split the rest.
	listW := m.width*45/100 - 2
	if listW < 20 {
		listW = 20
	}
	prevW := m.width - 4 - listW
	if prevW < 16 {
		prevW = 16
	}

	left := m.styles.pane.Width(listW).Height(innerH).
		Render(fitBlock(m.renderList(listW, innerH), listW, innerH))
	right := m.styles.previewPane.Width(prevW).Height(innerH).
		Render(fitBlock(m.renderPreview(prevW, innerH), prevW, innerH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var b strings.Builder
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(m.footer())

	base := b.String()

	// Prompt and confirm are modal overlays; they render below the fixed body.
	switch m.mode {
	case modePrompt:
		base += "\n" + m.viewPromptModal()
	case modeConfirm:
		base += "\n" + m.viewConfirmModal()
	}
	return base
}

// footer renders exactly footerH lines: a status line (or blank) and either the
// help bar or, in filter mode, the live filter input — so toggling filter does
// not change the overall height.
func (m *model) footer() string {
	st := m.styles
	status := ""
	if m.statusMsg != "" {
		status = st.warnText.Render(m.statusMsg)
	}
	second := m.keys.helpBar(st)
	if m.mode == modeFilter {
		second = st.prompt.Render("/" + m.filterText + "▌")
	}
	return status + "\n" + second
}

func (m *model) viewPromptModal() string {
	st := m.styles
	body := st.modalTitle.Render("New worktree") + "\n\n" +
		"branch name: " + st.prompt.Render(m.promptText+"▌") + "\n\n" +
		st.help.Render("enter create  esc cancel")
	return st.modal.Render(body)
}

func (m *model) viewConfirmModal() string {
	st := m.styles
	target := ""
	if m.confirmTgt >= 0 && m.confirmTgt < len(m.rows) {
		target = m.rows[m.confirmTgt].label()
	}
	verb := "Remove worktree"
	if m.confirmKind == actRemoveDeleteBranch {
		verb = "Remove worktree AND delete its branch"
	}
	body := st.modalDanger.Render(verb) + "\n\n" +
		fmt.Sprintf("Target: %s\n\n", target) +
		st.help.Render("y confirm  n/esc cancel")
	return st.modal.Render(body)
}

func (m *model) viewPR() string {
	prevH := m.height - 2
	if prevH < 5 {
		prevH = 5
	}
	body := m.styles.pane.Width(m.width - 4).Height(prevH).Render(m.renderPR(m.width-8, prevH))
	if m.statusMsg != "" {
		body += "\n" + m.styles.warnText.Render(m.statusMsg)
	}
	return body
}
