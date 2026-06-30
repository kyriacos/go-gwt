package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kyriacos/go-gwt/internal/ui"
)

// helpStyles are used for colored CLI help (always on; see styledHelp).
type helpStyles struct {
	title, cmd, short, section, flag, shorthand, desc, example, alias, dim lipgloss.Style
}

var helpSt = helpStyles{
	title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
	cmd:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
	short:     lipgloss.Style{},
	section:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
	flag:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	shorthand: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	desc:      lipgloss.Style{},
	example:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	alias:     lipgloss.NewStyle().Faint(true),
	dim:       lipgloss.NewStyle().Faint(true),
}

func initHelp(root *cobra.Command) {
	root.SetHelpFunc(styledHelp)
	root.SetUsageFunc(styledUsage)
	for _, c := range root.Commands() {
		initHelp(c)
	}
}

func styledHelp(cmd *cobra.Command, _ []string) {
	ui.SetColor(ui.Always)
	out := cmd.OutOrStdout()
	st := helpSt

	// Header: gwt co — Switch to a worktree…
	path := cmd.CommandPath()
	fmt.Fprintln(out, st.title.Render(path)+st.dim.Render(" — ")+cmd.Short)
	fmt.Fprintln(out)

	if cmd.Long != "" {
		for _, para := range strings.Split(strings.TrimSpace(cmd.Long), "\n\n") {
			fmt.Fprintln(out, para)
			fmt.Fprintln(out)
		}
	}

	if cmd.Runnable() && cmd.UsageString() != "" {
		fmt.Fprintln(out, st.section.Render("Usage"))
		fmt.Fprintln(out, "  "+st.cmd.Render(cmd.UseLine()))
		fmt.Fprintln(out)
	}

	if len(cmd.Aliases) > 0 {
		fmt.Fprintln(out, st.section.Render("Aliases"))
		fmt.Fprintln(out, "  "+st.alias.Render(strings.Join(cmd.Aliases, ", ")))
		fmt.Fprintln(out)
	}

	if cmd.Example != "" {
		fmt.Fprintln(out, st.section.Render("Examples"))
		for line := range strings.SplitSeq(strings.TrimSpace(cmd.Example), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Fprintln(out, "  "+st.example.Render(line))
		}
		fmt.Fprintln(out)
	}

	if subs := visibleSubcommands(cmd); len(subs) > 0 {
		fmt.Fprintln(out, st.section.Render("Commands"))
		max := 0
		for _, s := range subs {
			if l := len(s.Name()); l > max {
				max = l
			}
		}
		for _, s := range subs {
			name := fmt.Sprintf("  %-*s", max, s.Name())
			fmt.Fprintln(out, st.cmd.Render(name)+"  "+st.dim.Render(s.Short))
		}
		fmt.Fprintln(out)
	}

	printFlagSection(out, st, "Flags", cmd.LocalFlags())
	printFlagSection(out, st, "Global flags", cmd.InheritedFlags())

	if cmd.Root().HasAvailableSubCommands() && cmd != cmd.Root() {
		fmt.Fprintln(out, st.dim.Render("Tip: gwt <command> --help for more on a subcommand."))
	}
}

func styledUsage(cmd *cobra.Command) error {
	ui.SetColor(ui.Always)
	out := cmd.OutOrStdout()
	st := helpSt
	fmt.Fprintf(out, "%s\n\n", st.section.Render("Usage"))
	fmt.Fprintf(out, "  %s\n\n", st.cmd.Render(cmd.UseLine()))
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintln(out, st.dim.Render("Run 'gwt <command> --help' for details on a command."))
	}
	return nil
}

func visibleSubcommands(cmd *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.Hidden {
			continue
		}
		out = append(out, c)
	}
	return out
}

func printFlagSection(out io.Writer, st helpStyles, heading string, flags *pflag.FlagSet) {
	if flags == nil || !flags.HasAvailableFlags() {
		return
	}
	fmt.Fprintln(out, st.section.Render(heading))
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		var name strings.Builder
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			name.WriteString(st.shorthand.Render("-" + f.Shorthand))
			name.WriteString(", ")
		}
		name.WriteString(st.flag.Render("--" + f.Name))
		if f.Value.Type() != "bool" {
			name.WriteString(st.dim.Render(" " + f.Value.Type()))
		}
		if f.DefValue != "" && f.DefValue != "false" {
			name.WriteString(st.dim.Render(" (default " + f.DefValue + ")"))
		}
		fmt.Fprintf(out, "  %-28s %s\n", name.String(), f.Usage)
	})
	fmt.Fprintln(out)
}
