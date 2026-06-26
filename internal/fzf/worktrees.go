package fzf

import (
	"sort"

	"github.com/kyriacos/go-gwt/internal/git"
)

// BuildWorktreeLines prepares fzf rows for every worktree in the repo.
func BuildWorktreeLines(repo git.Repo) ([]WorktreeLine, map[string]string, error) {
	wts, err := repo.List()
	if err != nil {
		return nil, nil, err
	}
	states, _ := repo.BranchStates()
	cur, _ := repo.Root()

	class := make([]string, len(wts))
	maxBranch, maxPath := 0, 0
	for i, wt := range wts {
		st := git.ClassifyWorktree(wt, states)
		class[i] = st
		cell := branchCell(wt, st)
		if len(cell) > maxBranch {
			maxBranch = len(cell)
		}
		if len(wt.Path) > maxPath {
			maxPath = len(wt.Path)
		}
	}

	styles := DefaultStyles()
	lines := make([]WorktreeLine, len(wts))
	stateByPath := make(map[string]string, len(wts))
	for i, wt := range wts {
		lines[i] = FormatWorktreeLine(wt, class[i], cur, maxBranch, maxPath, styles)
		stateByPath[wt.Path] = class[i]
	}
	return lines, stateByPath, nil
}

// BuildBranchLines prepares fzf rows for local branches.
func BuildBranchLines(repo git.Repo) ([]BranchLine, error) {
	states, err := repo.BranchStates()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(states))
	for n := range states {
		names = append(names, n)
	}
	sort.Strings(names)

	styles := DefaultStyles()
	lines := make([]BranchLine, len(names))
	for i, n := range names {
		st := states[n]
		tag := branchStateTag(st)
		display := styles.ForState(st).Render(n + tag)
		lines[i] = BranchLine{Display: display, Name: n}
	}
	return lines, nil
}

func branchStateTag(state string) string {
	switch state {
	case git.StateLocal:
		return ""
	case git.StateGone:
		return " (gone)"
	default:
		return ""
	}
}

// FormatBranchHeader is the fzf header for branch pickers.
func FormatBranchHeader() string {
	return "red=upstream gone  blue=local-only  green=active"
}

// FormatCleanHeader is the fzf header for clean multi-select.
func FormatCleanHeader() string {
	return "TAB select / ENTER confirm  |  red=stale (gone/missing)  blue=local-only  green=active"
}

// PreselectStale returns paths of worktrees in gone/missing state for fzf
// --bind to pre-select (not used in basic impl - bash doesn't preselect in fzf multi).
func PreselectStale(stateByPath map[string]string) []string {
	var out []string
	for p, st := range stateByPath {
		if git.IsStale(st) {
			out = append(out, p)
		}
	}
	return out
}

// FormatCleanLines builds fzf rows for removable worktrees (skips main/bare).
func FormatCleanLines(repo git.Repo) ([]WorktreeLine, map[string]string, error) {
	lines, stateByPath, err := BuildWorktreeLines(repo)
	if err != nil {
		return nil, nil, err
	}
	var filtered []WorktreeLine
	filteredStates := make(map[string]string)
	wts, _ := repo.List()
	wtByPath := make(map[string]git.Worktree, len(wts))
	for _, wt := range wts {
		wtByPath[wt.Path] = wt
	}
	for _, ln := range lines {
		wt := wtByPath[ln.Path]
		if wt.IsMain || wt.Bare {
			continue
		}
		filtered = append(filtered, ln)
		filteredStates[ln.Path] = stateByPath[ln.Path]
	}
	if len(filtered) == 0 {
		return nil, nil, nil
	}
	return filtered, filteredStates, nil
}
