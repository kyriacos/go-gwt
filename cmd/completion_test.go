package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/git"
	"github.com/kyriacos/go-gwt/internal/shell"
)

func TestWorktreeAliases(t *testing.T) {
	t.Parallel()
	got := worktreeAliases(git.Worktree{Branch: "feature/foo", Path: "/code/backend-feature-foo"})
	if len(got) != 2 || got[0] != "feature/foo" || got[1] != "backend-feature-foo" {
		t.Fatalf("aliases = %v", got)
	}
}

func TestCompleteNewArgsArity(t *testing.T) {
	t.Parallel()
	if _, d := completeNewArgs(nil, []string{"name", "base"}, ""); d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("third arg should not complete")
	}
}

func TestShellInitIncludesCompletion(t *testing.T) {
	t.Parallel()
	script, err := shell.Init("zsh", "gwt", "/opt/homebrew/bin/gwt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "_GWT_BIN=") {
		t.Fatalf("shell-init missing _GWT_BIN:\n%s", script)
	}
	if strings.Contains(script, `"'`) {
		t.Fatalf("shell-init has broken nested quotes in completion:\n%s", script)
	}
}
