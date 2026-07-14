package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/kyriacos/go-gwt/internal/version"
)

// View renders the current mode. The runtime normalizes output to the terminal
// size before drawing.
func (m *model) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case modePR:
		return m.viewPRBody()
	case modeHelp:
		return m.composeOverlay(m.helpModalContent())
	case modeChangelog:
		return m.composeOverlay(m.changelogModalContent())
	case modePrompt:
		return m.composeOverlay(m.promptModalContent())
	case modeConfirm:
		return m.composeOverlay(m.confirmModalContent())
	default:
		return m.dashboardContent()
	}
}

// footerH is the fixed number of lines reserved below the panes (one status
// line + one help/filter line). Keeping it constant — together with
// fixed-height panes — is what stops the screen from jumping as the selection
// or preview changes.
const footerH = 2

// dashboardContent renders the list + preview split plus the footer.
func (m *model) dashboardContent() string {
	innerH := m.height - footerH - 2
	if innerH < 3 {
		innerH = 3
	}

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

	return body + "\n" + m.footer()
}

// footer renders exactly footerH lines: a status line (or blank) and either the
// help bar or, in filter mode, the live filter input. The build version is
// always right-aligned on the status line.
func (m *model) footer() string {
	st := m.styles
	ver := st.dim.Render(version.Short())
	left := ""
	if m.statusMsg != "" {
		left = st.warnText.Render(m.statusMsg)
	}
	status := joinFooterLine(left, ver, m.width)

	second := m.keys.helpBar(st)
	if m.mode == modeFilter {
		second = st.prompt.Render("/" + m.filterText + "▌")
	}
	return status + "\n" + second
}

func (m *model) promptModalContent() string {
	st := m.styles
	title := st.modalTitle.Render("New worktree")
	body := "branch name: " + st.prompt.Render(m.promptText+"▌") + "\n\n" +
		st.help.Render("enter create  esc cancel")
	return title + "\n\n" + body
}

func (m *model) confirmModalContent() string {
	st := m.styles
	target := ""
	if m.confirmTgt >= 0 && m.confirmTgt < len(m.rows) {
		target = m.rows[m.confirmTgt].label()
	}
	verb := "Remove worktree"
	if m.confirmKind == actRemoveDeleteBranch {
		verb = "Remove worktree AND delete its branch"
	}
	return st.modalDanger.Render(verb) + "\n\n" +
		fmt.Sprintf("Target: %s\n\n", target) +
		st.help.Render("y/d/enter confirm  n/esc cancel")
}

func (m *model) viewPRBody() string {
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
