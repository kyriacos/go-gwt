package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/kyriacos/go-gwt/internal/git"
)

// This file holds two small full-screen list models that reuse the runtime:
//   - branchModel: single-select branch picker (no-arg `from`/`co`).
//   - cleanModel:  multi-select worktree picker for `clean`.
// Both color rows by worktree/branch state to match `ls`.

// pickerStyles are the lipgloss styles shared by the pickers.
type pickerStyles struct {
	title, sel, help, dim, pane, previewPane, errText lipgloss.Style
	green, blue, red, yellow, mark, gray              lipgloss.Style
}

func newPickerStyles() pickerStyles {
	c := func(code string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(code)) }
	return pickerStyles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		sel:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")),
		help:        lipgloss.NewStyle().Faint(true),
		dim:         lipgloss.NewStyle().Faint(true),
		pane:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")),
		previewPane: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")),
		errText:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		green:       c("2"),
		blue:        c("4"),
		red:         c("1"),
		yellow:      c("3"),
		mark:        c("2").Bold(true),
		gray:        lipgloss.NewStyle().Faint(true),
	}
}

func (s pickerStyles) state(state string) lipgloss.Style {
	switch state {
	case git.StateActive:
		return s.green
	case git.StateLocal:
		return s.blue
	case git.StateGone, git.StateMissing:
		return s.red
	case git.StateDetached:
		return s.yellow
	default:
		return s.dim
	}
}

// stateLabel is the parenthetical shown after a name for non-active states.
func stateLabel(state string) string {
	switch state {
	case git.StateActive:
		return ""
	case git.StateLocal:
		return "local-only"
	default:
		return state
	}
}

// ---- branch picker --------------------------------------------------------

// BranchItem is one selectable local branch.
type BranchItem struct {
	Name  string
	State string // git.State* (active|local|gone)
}

// BranchPreviewFunc returns commit log text for a branch name. A nil func
// disables the preview pane.
type BranchPreviewFunc func(branch string) (string, error)

// PickBranch shows a full-screen branch picker and returns the chosen branch
// name, or "" if the user quit without selecting.
func PickBranch(items []BranchItem, preview BranchPreviewFunc) (string, error) {
	final, err := runModel(newBranchModel(items, preview))
	if err != nil {
		return "", err
	}
	if bm, ok := final.(*branchModel); ok {
		return bm.selected, nil
	}
	return "", nil
}

type branchModel struct {
	items         []BranchItem
	filtered      []int
	cursor        int
	filtering     bool
	filter        string
	selected      string
	quitting      bool
	width, height int
	st            pickerStyles
	preview       BranchPreviewFunc
	previewBranch string
	previewText   string
	previewErr    string
	spinFrame     int
}

func newBranchModel(items []BranchItem, preview BranchPreviewFunc) *branchModel {
	m := &branchModel{items: items, preview: preview, width: 80, height: 24, st: newPickerStyles()}
	m.applyFilter()
	return m
}

// branchPreviewMsg carries async preview text for a branch.
type branchPreviewMsg struct {
	branch string
	text   string
	err    error
}

func loadBranchPreview(fn BranchPreviewFunc, branch string) Cmd {
	if fn == nil {
		return nil
	}
	return func() Msg {
		txt, err := fn(branch)
		return branchPreviewMsg{branch: branch, text: txt, err: err}
	}
}

func (m *branchModel) previewCmd() Cmd {
	if m.preview == nil || len(m.filtered) == 0 {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	branch := m.items[m.filtered[m.cursor]].Name
	if branch == m.previewBranch && m.previewText != "" {
		return nil
	}
	m.previewBranch = branch
	m.previewText = ""
	m.previewErr = ""
	return loadBranchPreview(m.preview, branch)
}

func (m *branchModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	m.filtered = m.filtered[:0]
	for i, it := range m.items {
		if q == "" || strings.Contains(strings.ToLower(it.Name), q) {
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

func (m *branchModel) Init() Cmd {
	if m.preview == nil {
		return nil
	}
	return Batch(m.previewCmd(), tick())
}

func (m *branchModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.spinFrame++
		if m.preview != nil {
			return m, tick()
		}
		return m, nil
	case branchPreviewMsg:
		if msg.branch == m.previewBranch {
			m.previewText = msg.text
			if msg.err != nil {
				m.previewErr = msg.err.Error()
			} else {
				m.previewErr = ""
			}
		}
		return m, nil
	case windowSizeMsg:
		m.width, m.height = msg.width, msg.height
		return m, nil
	case KeyMsg:
		if m.filtering {
			return m.filterKey(msg)
		}
		return m.listKey(msg)
	}
	return m, nil
}

func (m *branchModel) filterKey(k KeyMsg) (Model, Cmd) {
	switch {
	case k.Type == keyEnter:
		m.filtering = false
	case k.Type == keyEsc:
		m.filtering = false
		m.filter = ""
		m.applyFilter()
	case k.Type == keyBackspace:
		if m.filter != "" {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	case k.Type == keyRunes:
		m.filter += string(k.Runes)
		m.applyFilter()
	}
	return m, m.previewCmd()
}

func (m *branchModel) listKey(k KeyMsg) (Model, Cmd) {
	s := k.String()
	switch {
	case k.Type == keyCtrlC, s == "q", k.Type == keyEsc:
		m.quitting = true
		return m, Quit
	case k.Type == keyUp, s == "k":
		m.move(-1)
		return m, m.previewCmd()
	case k.Type == keyDown, s == "j":
		m.move(1)
		return m, m.previewCmd()
	case s == "/":
		m.filtering = true
	case k.Type == keyEnter:
		if m.cursor >= 0 && m.cursor < len(m.filtered) {
			m.selected = m.items[m.filtered[m.cursor]].Name
			m.quitting = true
			return m, Quit
		}
	}
	return m, nil
}

func (m *branchModel) move(d int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m *branchModel) View() string {
	if m.quitting {
		return ""
	}
	if m.preview != nil {
		return m.viewWithPreview()
	}
	return m.viewListOnly()
}

func (m *branchModel) viewListOnly() string {
	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}
	innerH := m.height - footerH - 2
	if innerH < 3 {
		innerH = 3
	}
	body := m.st.pane.Width(innerW).Height(innerH).
		Render(fitBlock(m.renderList(innerW, innerH), innerW, innerH))
	return body + "\n\n" + m.footer()
}

func (m *branchModel) viewWithPreview() string {
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
	st := m.st
	left := st.pane.Width(listW).Height(innerH).
		Render(fitBlock(m.renderList(listW, innerH), listW, innerH))
	right := st.previewPane.Width(prevW).Height(innerH).
		Render(fitBlock(m.renderPreview(prevW, innerH), prevW, innerH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return body + "\n\n" + m.footer()
}

func (m *branchModel) renderList(innerW, innerH int) string {
	st := m.st
	var lines []string
	lines = append(lines, st.title.Render(fmt.Sprintf("Select a branch (%d)", len(m.items))), "")
	if len(m.filtered) == 0 {
		lines = append(lines, st.dim.Render("(no branches match)"))
	}
	visN := innerH - 2
	start := 0
	if visN > 0 && m.cursor >= visN {
		start = m.cursor - visN + 1
	}
	for vi := start; vi < len(m.filtered) && vi < start+visN; vi++ {
		it := m.items[m.filtered[vi]]
		cursor := "  "
		name := it.Name
		if lbl := stateLabel(it.State); lbl != "" {
			name += " (" + lbl + ")"
		}
		if vi == m.cursor {
			lines = append(lines, st.sel.Render(padVis("> "+name, innerW)))
			continue
		}
		lines = append(lines, cursor+st.state(it.State).Render(name))
	}
	return strings.Join(lines, "\n")
}

func (m *branchModel) renderPreview(width, height int) string {
	st := m.st
	header := st.title.Render("Log") + "\n\n"
	if len(m.filtered) == 0 {
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

func (m *branchModel) footer() string {
	st := m.st
	if m.filtering {
		return st.title.Render("/" + m.filter + "▌")
	}
	return st.help.Render("enter select  / filter  ↑/↓ or j/k move  q quit")
}

// ---- clean multi-select picker --------------------------------------------

// CleanItem is one worktree offered for cleanup.
type CleanItem struct {
	Path   string
	Branch string
	State  string
}

// PickForClean shows a multi-select picker (stale entries pre-marked) and
// returns the paths the user confirmed for removal. Empty slice means cancel /
// nothing selected.
func PickForClean(items []CleanItem) ([]string, error) {
	final, err := runModel(newCleanModel(items))
	if err != nil {
		return nil, err
	}
	if cm, ok := final.(*cleanModel); ok {
		return cm.confirmed, nil
	}
	return nil, nil
}

type cleanModel struct {
	items         []CleanItem
	filtered      []int
	cursor        int
	marked        map[int]bool
	filtering     bool
	filter        string
	confirmed     []string
	quitting      bool
	width, height int
	st            pickerStyles
}

func newCleanModel(items []CleanItem) *cleanModel {
	m := &cleanModel{items: items, marked: map[int]bool{}, width: 80, height: 24, st: newPickerStyles()}
	for i, it := range items {
		if git.IsStale(it.State) {
			m.marked[i] = true // pre-mark stale; one Enter cleans them
		}
	}
	m.applyFilter()
	return m
}

func (m *cleanModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	m.filtered = m.filtered[:0]
	for i, it := range m.items {
		hay := strings.ToLower(it.Branch + " " + it.Path)
		if q == "" || strings.Contains(hay, q) {
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

func (m *cleanModel) Init() Cmd { return nil }

func (m *cleanModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case windowSizeMsg:
		m.width, m.height = msg.width, msg.height
		return m, nil
	case KeyMsg:
		if m.filtering {
			switch {
			case msg.Type == keyEnter:
				m.filtering = false
			case msg.Type == keyEsc:
				m.filtering = false
				m.filter = ""
				m.applyFilter()
			case msg.Type == keyBackspace:
				if m.filter != "" {
					m.filter = m.filter[:len(m.filter)-1]
					m.applyFilter()
				}
			case msg.Type == keyRunes:
				m.filter += string(msg.Runes)
				m.applyFilter()
			}
			return m, nil
		}
		s := msg.String()
		switch {
		case msg.Type == keyCtrlC, s == "q", msg.Type == keyEsc:
			m.quitting = true
			return m, Quit
		case msg.Type == keyUp, s == "k":
			m.move(-1)
		case msg.Type == keyDown, s == "j":
			m.move(1)
		case s == "/":
			m.filtering = true
		case msg.Type == keySpace:
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				idx := m.filtered[m.cursor]
				m.marked[idx] = !m.marked[idx]
			}
		case msg.Type == keyEnter:
			for i, it := range m.items {
				if m.marked[i] {
					m.confirmed = append(m.confirmed, it.Path)
				}
			}
			m.quitting = true
			return m, Quit
		}
	}
	return m, nil
}

func (m *cleanModel) move(d int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m *cleanModel) View() string {
	if m.quitting {
		return ""
	}
	innerW := m.width - 2
	if innerW < 10 {
		innerW = 10
	}
	innerH := m.height - footerH - 2
	if innerH < 3 {
		innerH = 3
	}
	st := m.st

	var lines []string
	marks := 0
	for _, v := range m.marked {
		if v {
			marks++
		}
	}
	lines = append(lines, st.title.Render(fmt.Sprintf("Clean worktrees — %d marked", marks)), "")
	visN := innerH - 2
	start := 0
	if visN > 0 && m.cursor >= visN {
		start = m.cursor - visN + 1
	}
	for vi := start; vi < len(m.filtered) && vi < start+visN; vi++ {
		idx := m.filtered[vi]
		it := m.items[idx]
		box := "[ ]"
		if m.marked[idx] {
			box = st.mark.Render("[x]")
		}
		label := it.Branch
		if label == "" {
			label = "(detached)"
		}
		if lbl := stateLabel(it.State); lbl != "" && it.Branch != "" {
			label += " (" + lbl + ")"
		}
		row := box + " " + st.state(it.State).Render(label) + "  " + st.gray.Render(it.Path)
		cursor := "  "
		if vi == m.cursor {
			cursor = st.title.Render("> ")
		}
		lines = append(lines, cursor+row)
	}

	body := st.pane.Width(innerW).Height(innerH).Render(fitBlock(strings.Join(lines, "\n"), innerW, innerH))
	footer := st.help.Render("space mark  enter remove marked  / filter  ↑/↓ move  q cancel")
	if m.filtering {
		footer = st.title.Render("/" + m.filter + "▌")
	}
	return body + "\n\n" + footer
}
