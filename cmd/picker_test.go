package cmd

import (
	"testing"

	"github.com/kyriacos/go-gwt/internal/config"
)

func TestUseFzf_DefaultTUI(t *testing.T) {
	forceTUI, forceFzf = false, false
	if useFzf(config.Defaults()) {
		t.Fatal("default config should use TUI")
	}
}

func TestUseFzf_ConfigFzf(t *testing.T) {
	forceTUI, forceFzf = false, false
	cfg := config.Defaults()
	cfg.UI.Picker = config.PickerFzf
	if !useFzf(cfg) {
		t.Fatal("picker=fzf should use fzf")
	}
}

func TestUseFzf_FlagOverrides(t *testing.T) {
	cfg := config.Defaults()
	cfg.UI.Picker = config.PickerFzf

	forceTUI, forceFzf = true, false
	if useFzf(cfg) {
		t.Fatal("--tui should override config fzf")
	}

	forceTUI, forceFzf = false, true
	if !useFzf(cfg) {
		t.Fatal("--fzf should force fzf")
	}
	forceTUI, forceFzf = false, false
}
