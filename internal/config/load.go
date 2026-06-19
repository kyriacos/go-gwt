package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load resolves configuration. repoRoot is the main worktree path (used to find
// a repo-local .gwt.toml); pass "" if unknown.
//
// Precedence, highest first:
//  1. Command-line flags (applied by the caller AFTER Load returns).
//  2. Environment variables (see applyEnv for the full set).
//  3. Repo-local .gwt.toml at repoRoot.
//  4. User config at ${XDG_CONFIG_HOME:-$HOME/.config}/gwt/config.toml.
//  5. Built-in Defaults().
//
// A missing file is not an error. A malformed file IS an error (wrapped with
// its path). Invalid enum values are reported with the offending key.
func Load(repoRoot string) (Config, error) {
	cfg := Defaults()

	// 4. User config (lowest file precedence).
	if path, ok := UserConfigPath(); ok {
		if err := mergeFile(&cfg, path); err != nil {
			return Config{}, err
		}
	}

	// 3. Repo-local config (overrides the user config).
	if repoRoot != "" {
		repoPath := filepath.Join(repoRoot, ".gwt.toml")
		if err := mergeFile(&cfg, repoPath); err != nil {
			return Config{}, err
		}
	}

	// 2. Environment variables (override files).
	applyEnv(&cfg)

	// Validate the final, fully-resolved config.
	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// UserConfigPath returns the path to the user config file and whether it
// exists. It honors $XDG_CONFIG_HOME, falling back to $HOME/.config. It is
// exported so tests can exercise path resolution via env overrides. The second
// return is false when the file does not exist or no base directory can be
// determined.
func UserConfigPath() (string, bool) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", false
		}
		base = filepath.Join(home, ".config")
	}
	path := filepath.Join(base, "gwt", "config.toml")
	if !fileExists(path) {
		return path, false
	}
	return path, true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// mergeFile decodes the TOML file at path over cfg. A missing file is a no-op.
// A malformed file returns an error wrapped with the path.
func mergeFile(cfg *Config, path string) error {
	if !fileExists(path) {
		return nil
	}
	// Decode into the existing Config so unspecified keys keep their prior
	// (defaults or user-file) values. BurntSushi only writes keys present in
	// the document, giving natural overlay semantics.
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return fmt.Errorf("config: parsing %s: %w", path, err)
	}
	return nil
}
