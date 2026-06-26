// Package config loads gwt configuration and resolves it across the precedence
// chain: flag > env > repo-local .gwt.toml > user config > built-in defaults.
// The config-layer agent implements Load; the struct shape is frozen here.
package config

// WorktreeSetup controls whether IDE-specific repo worktree preparation runs.
type WorktreeSetup string

const (
	SetupPrompt WorktreeSetup = "prompt"
	SetupAlways WorktreeSetup = "always"
	SetupNever  WorktreeSetup = "never"
)

// ColorMode controls colored output.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// Cursor holds settings for Cursor's .cursor/worktrees.json integration.
type Cursor struct {
	// WorktreeSetup controls consent for setup-worktree commands in
	// .cursor/worktrees.json. Empty means fall through to the deprecated
	// top-level auto_setup, then the default (prompt).
	WorktreeSetup WorktreeSetup `toml:"worktree_setup"`
}

// Claude holds settings for Claude Code worktree preparation (.worktreeinclude).
type Claude struct {
	// WorktreeSetup controls consent for copying gitignored paths listed in
	// .worktreeinclude. Empty defaults to prompt.
	WorktreeSetup WorktreeSetup `toml:"worktree_setup"`
}

// Hooks are user-configured lifecycle commands (trusted; always run).
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
	WorktreeDir string `toml:"worktree_dir"`
	Naming      string `toml:"naming"` // tokens: {repo} {branch} {branch_slug}

	// AutoSetup is deprecated; use [cursor].worktree_setup instead.
	AutoSetup WorktreeSetup `toml:"auto_setup"`

	Cursor Cursor `toml:"cursor"`
	Claude Claude `toml:"claude"`

	OpenEditor bool   `toml:"open_editor"`
	Editor     string `toml:"editor"`
	Tmux       bool   `toml:"tmux"`
	Hooks      Hooks  `toml:"hooks"`
	GH         GH     `toml:"gh"`
	UI         UI     `toml:"ui"`
}

// CursorWorktreeSetup returns the effective consent mode for Cursor worktree
// setup (.cursor/worktrees.json). Precedence: [cursor].worktree_setup >
// deprecated auto_setup > prompt.
func (c Config) CursorWorktreeSetup() WorktreeSetup {
	if c.Cursor.WorktreeSetup != "" {
		return c.Cursor.WorktreeSetup
	}
	if c.AutoSetup != "" {
		return c.AutoSetup
	}
	return SetupPrompt
}

// ClaudeWorktreeSetup returns the effective consent mode for Claude Code
// worktree preparation. Precedence: [claude].worktree_setup > prompt.
func (c Config) ClaudeWorktreeSetup() WorktreeSetup {
	if c.Claude.WorktreeSetup != "" {
		return c.Claude.WorktreeSetup
	}
	return SetupPrompt
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	return Config{
		WorktreeDir: "",
		Naming:      "{repo}-{branch}",
		OpenEditor:  false,
		Editor:      "",
		Tmux:        false,
		GH:          GH{Enabled: true},
		UI:          UI{Color: ColorAuto},
	}
}

// Load is implemented in load.go.
