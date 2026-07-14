package cmd

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/git"
)

// registerCompletions wires dynamic tab completion for worktree and branch names.
func registerCompletions(root *cobra.Command) {
	if c := findSubCmd(root, "rm"); c != nil {
		c.ValidArgsFunction = completeWorktreeTargets
	}
	for _, name := range []string{"co", "from"} {
		if c := findSubCmd(root, name); c != nil {
			c.ValidArgsFunction = completeBranchTargets
		}
	}
	if c := findSubCmd(root, "new"); c != nil {
		c.ValidArgsFunction = completeNewArgs
	}
}

func findSubCmd(root *cobra.Command, name string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func completeNewArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		// Branch name for the new worktree; no fixed list.
		return nil, cobra.ShellCompDirectiveNoFileComp
	case 1:
		return branchNames(toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeBranchTargets(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return branchNames(toComplete)
}

func completeWorktreeTargets(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return worktreeTargetNames(toComplete)
}

func worktreeTargetNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	d, err := build()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	wts, err := d.repo.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	seen := map[string]struct{}{}
	var names []string
	prefix := strings.ToLower(toComplete)
	for _, wt := range wts {
		if wt.IsMain {
			continue
		}
		for _, cand := range worktreeAliases(wt) {
			if cand == "" {
				continue
			}
			if _, ok := seen[cand]; ok {
				continue
			}
			if prefix != "" && !strings.HasPrefix(strings.ToLower(cand), prefix) {
				continue
			}
			seen[cand] = struct{}{}
			names = append(names, cand)
		}
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

func branchNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	d, err := build()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	states, err := d.repo.BranchStates()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	prefix := strings.ToLower(toComplete)
	var names []string
	for name := range states {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

func worktreeAliases(wt git.Worktree) []string {
	base := filepath.Base(wt.Path)
	if wt.Branch != "" {
		return []string{wt.Branch, base}
	}
	return []string{base}
}
