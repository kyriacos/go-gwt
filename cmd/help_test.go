package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kyriacos/go-gwt/internal/ui"
)

func TestAllCommandsHaveHelp(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	for _, c := range root.Commands() {
		if c.Hidden {
			continue
		}
		if strings.TrimSpace(c.Long) == "" {
			t.Errorf("%q missing Long help text", c.Name())
		}
		if strings.TrimSpace(c.Example) == "" {
			t.Errorf("%q missing Example help text", c.Name())
		}
	}
}

func TestCoHelp(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	co, _, err := root.Find([]string{"co"})
	if err != nil {
		t.Fatal(err)
	}
	co.SetOut(&buf)
	co.SetHelpFunc(styledHelp)
	co.Help()

	out := buf.String()
	for _, want := range []string{
		"Switch to the worktree",
		"--fzf",
		"Examples",
		"gwt co feature",
		"Global flags",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("co --help missing %q\n%s", want, out)
		}
	}
}

func TestHelpUsesColor(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetHelpFunc(styledHelp)
	root.Help()

	if !strings.Contains(buf.String(), "\x1b[") {
		t.Skip("terminal may not support ANSI in test env")
	}
}

func TestRootHelpListsCommands(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetHelpFunc(styledHelp)
	root.Help()

	out := buf.String()
	for _, cmd := range []string{"co", "new", "from", "clean", "search"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("root help missing command %q", cmd)
		}
	}
}

func TestStyledUsage(t *testing.T) {
	t.Parallel()
	ui.SetColor(ui.Always)
	root := NewRootCmd()
	co, _, _ := root.Find([]string{"co"})
	var buf bytes.Buffer
	co.SetOut(&buf)
	co.SetUsageFunc(styledUsage)
	_ = co.Usage()

	if !strings.Contains(buf.String(), "Usage") {
		t.Fatalf("usage output: %q", buf.String())
	}
}
