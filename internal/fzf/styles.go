package fzf

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/kyriacos/go-gwt/internal/git"
)

// FzfStyles holds lipgloss styles for fzf display lines (matches gwt ls).
type FzfStyles struct {
	Green, Blue, Red, Yellow, Cyan, CyanBold, Bold, Dim lipgloss.Style
}

// DefaultStyles returns the standard state-colored styles.
func DefaultStyles() FzfStyles {
	c := func(code string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(code)) }
	return FzfStyles{
		Green:    c("2"),
		Blue:     c("4"),
		Red:      c("1"),
		Yellow:   c("3"),
		Cyan:     c("6"),
		CyanBold: c("6").Bold(true),
		Bold:     lipgloss.NewStyle().Bold(true),
		Dim:      lipgloss.NewStyle().Faint(true),
	}
}

func (s FzfStyles) ForState(state string) lipgloss.Style {
	switch state {
	case git.StateActive:
		return s.Green
	case git.StateLocal:
		return s.Blue
	case git.StateGone, git.StateMissing:
		return s.Red
	case git.StateDetached:
		return s.Yellow
	default:
		return s.Dim
	}
}
