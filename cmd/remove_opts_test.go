package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/config"
)

func TestResolveBranchDeletion_CLIOverridesConfig(t *testing.T) {
	t.Parallel()
	cfg := config.Defaults()
	cfg.Remove.DeleteBranch = true

	c := &cobra.Command{Use: "test"}
	c.Flags().Bool("delete-branch", false, "")
	c.Flags().Bool("force-delete-branch", false, "")
	_ = c.ParseFlags([]string{"--force-delete-branch"})

	del, force := resolveBranchDeletion(c, cfg, false, true)
	if !del || !force {
		t.Fatalf("got delete=%v force=%v, want true/true", del, force)
	}
}

func TestResolveBranchDeletion_ConfigDefault(t *testing.T) {
	t.Parallel()
	cfg := config.Defaults()
	cfg.Remove.DeleteBranch = true

	c := &cobra.Command{Use: "test"}
	c.Flags().Bool("delete-branch", false, "")
	c.Flags().Bool("force-delete-branch", false, "")

	del, force := resolveBranchDeletion(c, cfg, false, false)
	if !del || force {
		t.Fatalf("got delete=%v force=%v, want true/false", del, force)
	}
}

func TestResolveBranchDeletion_ExplicitOff(t *testing.T) {
	t.Parallel()
	cfg := config.Defaults()
	cfg.Remove.DeleteBranch = true

	c := &cobra.Command{Use: "test"}
	c.Flags().Bool("delete-branch", false, "")
	c.Flags().Bool("force-delete-branch", false, "")
	_ = c.ParseFlags([]string{"--delete-branch=false"})

	del, force := resolveBranchDeletion(c, cfg, false, false)
	if del || force {
		t.Fatalf("explicit --delete-branch=false should keep branch, got delete=%v force=%v", del, force)
	}
}
