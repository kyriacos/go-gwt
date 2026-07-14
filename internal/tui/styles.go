// Package tui implements the gwt dashboard: a single-screen, filterable manager
// of git worktrees with concurrent status loading, a preview pane, destructive
// action confirmation, and a gh-backed PR view.
//
// Dependency note: the project's go.mod pins only lipgloss (not bubbletea or
// bubbles), and this package is built under a hard constraint that forbids
// adding dependencies. The dashboard therefore ships a tiny self-contained
// runtime in runtime.go that follows the Bubble Tea Model/Update/View pattern
// (Msg, Cmd, batched commands, key events) so the structure matches the plan
// and the integration agent's mental model, without importing bubbletea or
// bubbles. If those modules are later added to go.mod, the Model/Update/View
// functions and message types map onto tea.Model directly.
package tui

import "github.com/charmbracelet/lipgloss"

// styles holds every lipgloss style the dashboard renders with. Colors honor
// the global lipgloss color profile, which the ui package sets from the color
// policy (NO_COLOR / [ui].color); when color is disabled the profile is Ascii
// and Render is a no-op passthrough.
type styles struct {
	title       lipgloss.Style
	help        lipgloss.Style
	spinner     lipgloss.Style
	pane        lipgloss.Style
	previewPane lipgloss.Style

	// list rows
	rowNormal   lipgloss.Style
	rowSelected lipgloss.Style
	marker      lipgloss.Style

	// worktree-state coloring (mirrors the legacy tool)
	stateActive    lipgloss.Style // current worktree
	stateLocalOnly lipgloss.Style // branch with no upstream
	stateDetached  lipgloss.Style // detached HEAD
	stateBare      lipgloss.Style // bare repo entry

	// status decorations
	ahead    lipgloss.Style
	behind   lipgloss.Style
	dirty    lipgloss.Style
	clean    lipgloss.Style
	sha      lipgloss.Style
	size     lipgloss.Style
	dim      lipgloss.Style
	errText  lipgloss.Style
	okText   lipgloss.Style
	warnText lipgloss.Style

	// modal / prompt
	modal        lipgloss.Style
	modalTitle   lipgloss.Style
	modalDanger  lipgloss.Style
	prompt       lipgloss.Style
	scrim        lipgloss.Style // dimmed full-screen backdrop
	overlayModal lipgloss.Style // centered modal panel

	// pr view
	prDraft   lipgloss.Style
	ciPassing lipgloss.Style
	ciFailing lipgloss.Style
	ciPending lipgloss.Style
}

func newStyles() styles {
	c := func(s string) lipgloss.Color { return lipgloss.Color(s) }
	return styles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(c("6")),
		help:        lipgloss.NewStyle().Faint(true),
		spinner:     lipgloss.NewStyle().Foreground(c("5")),
		pane:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c("8")),
		previewPane: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c("8")),

		rowNormal:   lipgloss.NewStyle(),
		rowSelected: lipgloss.NewStyle().Bold(true).Foreground(c("0")).Background(c("6")),
		marker:      lipgloss.NewStyle().Bold(true).Foreground(c("2")),

		stateActive:    lipgloss.NewStyle().Foreground(c("2")).Bold(true),
		stateLocalOnly: lipgloss.NewStyle().Foreground(c("3")),
		stateDetached:  lipgloss.NewStyle().Foreground(c("5")),
		stateBare:      lipgloss.NewStyle().Faint(true),

		ahead:    lipgloss.NewStyle().Foreground(c("2")),
		behind:   lipgloss.NewStyle().Foreground(c("1")),
		dirty:    lipgloss.NewStyle().Foreground(c("3")),
		clean:    lipgloss.NewStyle().Foreground(c("8")),
		sha:      lipgloss.NewStyle().Foreground(c("4")),
		size:     lipgloss.NewStyle().Faint(true),
		dim:      lipgloss.NewStyle().Faint(true),
		errText:  lipgloss.NewStyle().Foreground(c("1")).Bold(true),
		okText:   lipgloss.NewStyle().Foreground(c("2")),
		warnText: lipgloss.NewStyle().Foreground(c("3")),

		modal:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(c("3")).Padding(1, 2),
		modalTitle:  lipgloss.NewStyle().Bold(true),
		modalDanger: lipgloss.NewStyle().Foreground(c("1")).Bold(true),
		prompt:      lipgloss.NewStyle().Foreground(c("6")),
		scrim:       lipgloss.NewStyle().Faint(true),
		overlayModal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c("6")).
			Padding(1, 2),

		prDraft:   lipgloss.NewStyle().Faint(true),
		ciPassing: lipgloss.NewStyle().Foreground(c("2")),
		ciFailing: lipgloss.NewStyle().Foreground(c("1")),
		ciPending: lipgloss.NewStyle().Foreground(c("3")),
	}
}
