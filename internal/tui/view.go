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

// viewMain renders the list + preview split, the status line, help, and any
// active overlay (filter box, prompt, confirm modal).
func (m *model) viewMain() string {
	listW := m.width * 45 / 100
	if listW < 30 {
		listW = 30
	}
	prevW := m.width - listW - 6
	if prevW < 20 {
		prevW = 20
	}
	bodyH := m.height - 5
	if bodyH < 5 {
		bodyH = 5
	}

	left := m.styles.pane.Width(listW).Height(bodyH).Render(m.renderList(listW))
	right := m.styles.previewPane.Width(prevW).Height(bodyH).Render(m.renderPreview(prevW, bodyH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var b strings.Builder
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(m.footer())

	base := b.String()

	// Overlays render below the footer for simplicity and testability.
	switch m.mode {
	case modeFilter:
		base += "\n" + m.styles.prompt.Render("/"+m.filterText+"▌")
	case modePrompt:
		base += "\n" + m.viewPromptModal()
	case modeConfirm:
		base += "\n" + m.viewConfirmModal()
	}
	return base
}

func (m *model) footer() string {
	st := m.styles
	var lines []string
	if m.statusMsg != "" {
		lines = append(lines, st.warnText.Render(m.statusMsg))
	}
	lines = append(lines, m.keys.helpBar(st))
	return strings.Join(lines, "\n")
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
