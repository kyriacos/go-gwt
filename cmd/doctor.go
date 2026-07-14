package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyriacos/go-gwt/internal/ui"
	"github.com/kyriacos/go-gwt/internal/version"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Short:   "Check shell integration and conflicting gwt binaries",
		Long:    doctorLong,
		Example: doctorExample,
		Args:    cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			runDoctor()
		},
	}
}

func runDoctor() {
	fmt.Println(version.String())

	bin := version.Binary
	if bin == "" {
		bin, _ = os.Executable()
	}
	if bin != "" {
		bin, _ = filepath.EvalSymlinks(bin)
	}

	path := os.Getenv("PATH")
	seen := map[string]bool{}
	var others []string
	for _, dir := range filepath.SplitList(path) {
		candidate := filepath.Join(dir, "gwt")
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if bin != "" && sameFile(candidate, bin) {
			continue
		}
		others = append(others, candidate)
	}

	issues := 0
	if len(others) > 0 {
		issues++
		ui.Warn("Other gwt binaries on PATH (can break shell-init if they run first):")
		for _, o := range others {
			fmt.Printf("  %s\n", o)
		}
		fmt.Println()
	}

	if shell, ok := os.LookupEnv("SHELL"); ok {
		name := filepath.Base(shell)
		if name == "zsh" || name == "bash" || name == "fish" {
			fmt.Println("Refresh shell integration with this exact command (do not use bare `gwt`):")
			fmt.Printf("  eval $(%q shell-init %s)\n", bin, name)
		}
	}

	if issues == 0 {
		fmt.Println()
		ui.OK("No conflicting gwt binaries found.")
		fmt.Println("If setup still hangs, re-run the shell-init line above in this terminal.")
		return
	}

	fmt.Println()
	ui.Warn("Fix: move stale binaries aside, then re-run shell-init:")
	for _, o := range others {
		if strings.Contains(o, ".local/bin/gwt") {
			fmt.Printf("  mv %q %q\n", o, o+".bak")
		}
	}
}

func sameFile(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if a == b {
		return true
	}
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}
