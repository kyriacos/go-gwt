package config

import (
	"fmt"
	"os"
	"strings"
)

// Environment variables, applied last among the non-flag layers (env beats
// files). GWT_WORKTREE_DIR and GWT_RUN_SETUP mirror the bash gwt tool for
// parity.
//
//	GWT_WORKTREE_DIR  parent directory for new worktrees (string)
//	GWT_NAMING        directory naming template (string)
//	GWT_RUN_SETUP     sets AutoSetup:
//	                    always/prompt/never -> that mode
//	                    1/true/yes/on       -> always
//	                    0/false/no/off      -> never
//	                    anything else       -> leave as configured
//	GWT_EDITOR        editor command (string); also sets OpenEditor=true
//	GWT_NO_COLOR      if truthy, force ColorNever
//	NO_COLOR          standard convention; if set (any value), force ColorNever
func applyEnv(cfg *Config) {
	if v, ok := os.LookupEnv("GWT_WORKTREE_DIR"); ok {
		cfg.WorktreeDir = v
	}
	if v, ok := os.LookupEnv("GWT_NAMING"); ok && v != "" {
		cfg.Naming = v
	}
	if v, ok := os.LookupEnv("GWT_RUN_SETUP"); ok {
		switch AutoSetup(strings.ToLower(strings.TrimSpace(v))) {
		case SetupAlways, SetupPrompt, SetupNever:
			cfg.AutoSetup = AutoSetup(strings.ToLower(strings.TrimSpace(v)))
		default:
			switch parseBoolish(v) {
			case boolTrue:
				cfg.AutoSetup = SetupAlways
			case boolFalse:
				cfg.AutoSetup = SetupNever
			case boolUnknown:
				// leave as configured
			}
		}
	}
	if v, ok := os.LookupEnv("GWT_EDITOR"); ok && v != "" {
		cfg.Editor = v
		cfg.OpenEditor = true
	}
	// NO_COLOR: per the convention, presence (regardless of value) disables
	// color. GWT_NO_COLOR is the gwt-scoped equivalent and is treated as a
	// boolean for friendliness.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		cfg.UI.Color = ColorNever
	}
	if v, ok := os.LookupEnv("GWT_NO_COLOR"); ok {
		if parseBoolish(v) != boolFalse {
			cfg.UI.Color = ColorNever
		}
	}
}

type boolish int

const (
	boolUnknown boolish = iota
	boolTrue
	boolFalse
)

func parseBoolish(v string) boolish {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return boolTrue
	case "0", "false", "no", "off":
		return boolFalse
	default:
		return boolUnknown
	}
}

// validate checks enum-valued fields. It names the offending key on failure.
func validate(cfg Config) error {
	switch cfg.AutoSetup {
	case SetupPrompt, SetupAlways, SetupNever:
	default:
		return fmt.Errorf("config: invalid auto_setup %q (want one of: %s, %s, %s)",
			cfg.AutoSetup, SetupPrompt, SetupAlways, SetupNever)
	}
	switch cfg.UI.Color {
	case ColorAuto, ColorAlways, ColorNever:
	default:
		return fmt.Errorf("config: invalid ui.color %q (want one of: %s, %s, %s)",
			cfg.UI.Color, ColorAuto, ColorAlways, ColorNever)
	}
	return nil
}
