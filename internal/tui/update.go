package tui

// Init kicks off the worktree list load and the spinner tick.
func (m *model) Init() Cmd {
	return Batch(loadWorktrees(m.repo), tick())
}

// Update is the central message handler.
func (m *model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.spinFrame++
		// keep ticking while anything is loading or animating
		return m, tick()

	case worktreesMsg:
		if msg.err != nil {
			m.statusMsg = "list failed: " + msg.err.Error()
			return m, nil
		}
		m.rows = rowsFromWorktrees(msg.wts)
		m.loading = len(m.rows)
		m.cursor = 0
		m.applyFilter()
		cmds := []Cmd{loadStatuses(m.repo, msg.wts)}
		if c := m.previewCmd(); c != nil {
			cmds = append(cmds, c)
		}
		return m, Batch(cmds...)

	case statusMsg:
		for i := range m.rows {
			if m.rows[i].wt.Path == msg.path {
				m.rows[i].loaded = true
				if msg.err != nil {
					m.rows[i].loadErr = msg.err.Error()
				} else {
					m.rows[i].status = msg.status
					m.rows[i].size = msg.size
				}
				break
			}
		}
		if m.loading > 0 {
			m.loading--
		}
		return m, nil

	case previewMsg:
		if msg.path == m.previewPath {
			m.previewText = msg.text
			if msg.err != nil {
				m.previewErr = msg.err.Error()
			} else {
				m.previewErr = ""
			}
		}
		return m, nil

	case prListMsg:
		m.prLoaded = true
		if msg.err != nil {
			m.prErr = msg.err.Error()
		} else {
			m.prRows = msg.prs
			m.prErr = ""
			m.prCursor = 0
		}
		return m, nil

	case actionDoneMsg:
		return m.handleActionDone(msg)

	case windowSizeMsg:
		m.width = msg.width
		m.height = msg.height
		return m, nil

	case KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleActionDone updates state after a create/remove/clean/PR-checkout.
func (m *model) handleActionDone(msg actionDoneMsg) (Model, Cmd) {
	if msg.err != nil {
		m.statusMsg = msg.verb + " failed: " + msg.err.Error()
		m.mode = modeList
		return m, nil
	}
	switch msg.verb {
	case "created", "checkout":
		// Selecting the freshly created/checked-out worktree quits with its path.
		if msg.path != "" {
			m.selectedPath = msg.path
			m.quitting = true
			return m, Quit
		}
		fallthrough
	default:
		m.statusMsg = msg.msg
		m.mode = modeList
		// refresh the list to reflect the change
		return m, loadWorktrees(m.repo)
	}
}

// handleKey dispatches a key based on the current mode.
func (m *model) handleKey(k KeyMsg) (Model, Cmd) {
	switch m.mode {
	case modeFilter:
		return m.handleFilterKey(k)
	case modePrompt:
		return m.handlePromptKey(k)
	case modeConfirm:
		return m.handleConfirmKey(k)
	case modePR:
		return m.handlePRKey(k)
	default:
		return m.handleListKey(k)
	}
}

func (m *model) handleListKey(k KeyMsg) (Model, Cmd) {
	s := k.String()
	switch {
	case k.Type == keyCtrlC:
		m.quitting = true
		return m, Quit

	case m.keys.up.matches(s) || k.Type == keyUp:
		m.moveCursor(-1)
		return m, m.previewCmd()
	case m.keys.down.matches(s) || k.Type == keyDown:
		m.moveCursor(1)
		return m, m.previewCmd()

	case m.keys.enter.matches(s):
		if ri := m.currentRow(); ri >= 0 {
			m.selectedPath = m.rows[ri].wt.Path
			m.quitting = true
			return m, Quit
		}
		return m, nil

	case m.keys.filter.matches(s):
		m.mode = modeFilter
		return m, nil

	case m.keys.refresh.matches(s):
		m.statusMsg = ""
		return m, loadWorktrees(m.repo)

	case m.keys.newWT.matches(s):
		m.mode = modePrompt
		m.promptText = ""
		return m, nil

	case m.keys.remove.matches(s):
		if ri := m.currentRow(); ri >= 0 {
			m.confirmKind = actRemove
			m.confirmTgt = ri
			m.mode = modeConfirm
		}
		return m, nil

	case m.keys.removeD.matches(s):
		if ri := m.currentRow(); ri >= 0 {
			m.confirmKind = actRemoveDeleteBranch
			m.confirmTgt = ri
			m.mode = modeConfirm
		}
		return m, nil

	case m.keys.pr.matches(s):
		m.mode = modePR
		m.prLoaded = false
		m.prErr = ""
		return m, loadPRs(m.ghc)

	case m.keys.open.matches(s):
		// Open in editor is delegated to the caller via an Action only if one
		// exists; with the minimal Actions set we surface the path on the status
		// line. The integration agent may wire an Open(path) action if desired.
		if ri := m.currentRow(); ri >= 0 {
			m.statusMsg = "open: " + m.rows[ri].wt.Path
		}
		return m, nil

	case m.keys.quit.matches(s) || k.Type == keyEsc:
		m.quitting = true
		return m, Quit
	}
	return m, nil
}

func (m *model) handleFilterKey(k KeyMsg) (Model, Cmd) {
	switch {
	case k.Type == keyEnter:
		m.mode = modeList
		return m, m.previewCmd()
	case k.Type == keyEsc:
		m.filterText = ""
		m.applyFilter()
		m.mode = modeList
		return m, m.previewCmd()
	case k.Type == keyBackspace:
		if n := len(m.filterText); n > 0 {
			m.filterText = m.filterText[:n-1]
		}
		m.applyFilter()
		return m, nil
	case k.Type == keyRunes:
		m.filterText += string(k.Runes)
		m.applyFilter()
		return m, nil
	}
	return m, nil
}

func (m *model) handlePromptKey(k KeyMsg) (Model, Cmd) {
	switch {
	case k.Type == keyEnter:
		name := m.promptText
		m.mode = modeList
		if name == "" {
			return m, nil
		}
		m.statusMsg = "creating " + name + "…"
		return m, doCreate(m.acts, name)
	case k.Type == keyEsc:
		m.mode = modeList
		m.promptText = ""
		return m, nil
	case k.Type == keyBackspace:
		if n := len(m.promptText); n > 0 {
			m.promptText = m.promptText[:n-1]
		}
		return m, nil
	case k.Type == keySpace:
		m.promptText += " "
		return m, nil
	case k.Type == keyRunes:
		m.promptText += string(k.Runes)
		return m, nil
	}
	return m, nil
}

func (m *model) handleConfirmKey(k KeyMsg) (Model, Cmd) {
	s := k.String()
	switch {
	case m.keys.confirm.matches(s):
		ri := m.confirmTgt
		kind := m.confirmKind
		m.mode = modeList
		m.confirmKind = actNone
		if ri < 0 || ri >= len(m.rows) {
			return m, nil
		}
		target := m.rows[ri].wt.Path
		deleteBranch := kind == actRemoveDeleteBranch
		m.statusMsg = "removing…"
		return m, doRemove(m.acts, target, deleteBranch)
	case m.keys.cancel.matches(s) || k.Type == keyEsc:
		m.mode = modeList
		m.confirmKind = actNone
		return m, nil
	}
	return m, nil
}

func (m *model) handlePRKey(k KeyMsg) (Model, Cmd) {
	s := k.String()
	switch {
	case k.Type == keyUp || m.keys.up.matches(s):
		if m.prCursor > 0 {
			m.prCursor--
		}
		return m, nil
	case k.Type == keyDown || m.keys.down.matches(s):
		if m.prCursor < len(m.prRows)-1 {
			m.prCursor++
		}
		return m, nil
	case k.Type == keyEnter:
		if pr := m.currentPR(); pr != nil {
			m.statusMsg = "checking out PR…"
			return m, doCheckoutPR(m.acts, pr.Number)
		}
		return m, nil
	case k.Type == keyEsc:
		m.mode = modeList
		return m, nil
	case m.keys.quit.matches(s):
		m.quitting = true
		return m, Quit
	}
	return m, nil
}

// previewCmd returns a command to (re)load the preview for the highlighted row,
// updating previewPath. Returns nil when preview is disabled or the list is
// empty.
func (m *model) previewCmd() Cmd {
	if m.preview == nil {
		return nil
	}
	ri := m.currentRow()
	if ri < 0 {
		return nil
	}
	path := m.rows[ri].wt.Path
	if path == m.previewPath && m.previewText != "" {
		return nil
	}
	m.previewPath = path
	m.previewText = ""
	m.previewErr = ""
	return loadPreview(m.preview, path)
}
