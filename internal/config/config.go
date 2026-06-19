// Package config loads gwt configuration and resolves it across the precedence
// chain: flag > env > repo-local .gwt.toml > user config > built-in defaults.
// The config-layer agent implements Load; the struct shape is frozen here.
package config

// AutoSetup controls whether repo-provided setup commands run.
type AutoSetup string

const (
	SetupPrompt AutoSetup = "prompt"
	SetupAlways AutoSetup = "always"
	SetupNever  AutoSetup = "never"
)

// ColorMode controls colored output.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// Hooks are repo-provided lifecycle commands (untrusted; consent applies).
type Hooks struct {
	PostCreate []string `toml:"post_create"`
	PreRemove  []string `toml:"pre_remove"`
}

// GH toggles gh integration.
type GH struct {
	Enabled bool `toml:"enabled"`
}

// UI holds presentation options.
type UI struct {
	Color ColorMode `toml:"color"`
}

// Config is the fully-resolved configuration.
type Config struct {
	// WorktreeDir is the parent directory for new worktrees. Empty means the
	// default: the parent of the main worktree (sibling of the repo).
	WorktreeDir string    `toml:"worktree_dir"`
	Naming      string    `toml:"naming"` // tokens: {repo} {branch} {branch_slug}
	AutoSetup   AutoSetup `toml:"auto_setup"`
	OpenEditor  bool      `toml:"open_editor"`
	Editor      string    `toml:"editor"`
	Tmux        bool      `toml:"tmux"`
	Hooks       Hooks     `toml:"hooks"`
	GH          GH        `toml:"gh"`
	UI          UI        `toml:"ui"`
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	return Config{
		WorktreeDir: "",
		Naming:      "{repo}-{branch}",
		AutoSetup:   SetupPrompt,
		OpenEditor:  false,
		Editor:      "",
		Tmux:        false,
		GH:          GH{Enabled: true},
		UI:          UI{Color: ColorAuto},
	}
}

// Load is implemented in load.go.
