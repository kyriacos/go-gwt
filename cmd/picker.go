package cmd

import (
	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/fzf"
)

// forceTUI and forceFzf override [ui].picker for one invocation.
var forceTUI, forceFzf bool

// useFzf reports whether interactive pickers should use fzf instead of the
// built-in TUI. Precedence: --fzf > --tui > config (default: tui).
func useFzf(cfg config.Config) bool {
	if forceFzf {
		return true
	}
	if forceTUI {
		return false
	}
	return cfg.UsePickerFzf()
}

// fzfReady is useFzf plus an installed fzf binary.
func fzfReady(cfg config.Config) bool {
	return useFzf(cfg) && fzf.Available()
}
