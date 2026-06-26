package config

import (
	"fmt"
	"os"
	"strings"
)

// Environment variables, applied last among the non-flag layers (env beats
// files). GWT_WORKTREE_DIR and GWT_CURSOR_RUN_SETUP mirror the bash gwt tool
// for parity; GWT_RUN_SETUP is a deprecated alias for GWT_CURSOR_RUN_SETUP.
//
//	GWT_WORKTREE_DIR        parent directory for new worktrees (string)
//	GWT_NAMING              directory naming template (string)
//	GWT_CURSOR_RUN_SETUP    sets [cursor].worktree_setup:
//	                          always/prompt/never -> that mode
//	                          1/true/yes/on       -> always
//	                          0/false/no/off      -> never
//	                          anything else       -> leave as configured
//	GWT_RUN_SETUP             deprecated alias for GWT_CURSOR_RUN_SETUP
//	GWT_CLAUDE_RUN_SETUP      sets [claude].worktree_setup (same values)
//	GWT_EDITOR                editor command (string); also sets OpenEditor=true
//	GWT_NO_COLOR              if truthy, force ColorNever
//	NO_COLOR                  standard convention; if set (any value), force ColorNever
func applyEnv(cfg *Config) {
	if v, ok := os.LookupEnv("GWT_WORKTREE_DIR"); ok {
		cfg.WorktreeDir = v
	}
	if v, ok := os.LookupEnv("GWT_NAMING"); ok && v != "" {
		cfg.Naming = v
	}
	if v, ok := envWorktreeSetup("GWT_CURSOR_RUN_SETUP", "GWT_RUN_SETUP"); ok {
		cfg.Cursor.WorktreeSetup = v
	}
	if v, ok := envWorktreeSetup("GWT_CLAUDE_RUN_SETUP"); ok {
		cfg.Claude.WorktreeSetup = v
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

// envWorktreeSetup reads the first set env var from keys and parses it as a
// WorktreeSetup mode.
func envWorktreeSetup(keys ...string) (WorktreeSetup, bool) {
	var raw string
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			raw = v
			break
		}
	}
	if raw == "" {
		return "", false
	}
	switch WorktreeSetup(strings.ToLower(strings.TrimSpace(raw))) {
	case SetupAlways, SetupPrompt, SetupNever:
		return WorktreeSetup(strings.ToLower(strings.TrimSpace(raw))), true
	default:
		switch parseBoolish(raw) {
		case boolTrue:
			return SetupAlways, true
		case boolFalse:
			return SetupNever, true
		case boolUnknown:
			return "", false
		}
	}
	return "", false
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

func validateWorktreeSetup(field string, mode WorktreeSetup) error {
	if mode == "" {
		return nil
	}
	switch mode {
	case SetupPrompt, SetupAlways, SetupNever:
		return nil
	default:
		return fmt.Errorf("config: invalid %s %q (want one of: %s, %s, %s)",
			field, mode, SetupPrompt, SetupAlways, SetupNever)
	}
}

// validate checks enum-valued fields. It names the offending key on failure.
func validate(cfg Config) error {
	if err := validateWorktreeSetup("auto_setup", cfg.AutoSetup); err != nil {
		return err
	}
	if err := validateWorktreeSetup("cursor.worktree_setup", cfg.Cursor.WorktreeSetup); err != nil {
		return err
	}
	if err := validateWorktreeSetup("claude.worktree_setup", cfg.Claude.WorktreeSetup); err != nil {
		return err
	}
	switch cfg.UI.Color {
	case ColorAuto, ColorAlways, ColorNever:
	default:
		return fmt.Errorf("config: invalid ui.color %q (want one of: %s, %s, %s)",
			cfg.UI.Color, ColorAuto, ColorAlways, ColorNever)
	}
	return nil
}
