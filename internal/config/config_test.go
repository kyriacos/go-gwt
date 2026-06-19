package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// isolateEnv clears every env var the loader reads and points HOME /
// XDG_CONFIG_HOME at empty temp dirs so a stray real user config cannot leak
// in. It returns the XDG dir for tests that want to drop a user config there.
func isolateEnv(t *testing.T) (xdgDir string) {
	t.Helper()
	xdgDir = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{
		"GWT_WORKTREE_DIR", "GWT_NAMING", "GWT_RUN_SETUP",
		"GWT_EDITOR", "GWT_NO_COLOR", "NO_COLOR",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	return xdgDir
}

// writeUserConfig writes config.toml under the XDG gwt dir.
func writeUserConfig(t *testing.T, xdgDir, content string) {
	t.Helper()
	dir := filepath.Join(xdgDir, "gwt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeRepoConfig writes a .gwt.toml at repoRoot.
func writeRepoConfig(t *testing.T, repoRoot, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoRoot, ".gwt.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_DefaultsOnly(t *testing.T) {
	isolateEnv(t)
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("got %+v, want defaults %+v", got, Defaults())
	}
}

func TestLoad_MissingFiles_ReturnsDefaults(t *testing.T) {
	isolateEnv(t)
	// repoRoot points at an existing dir with no .gwt.toml.
	repo := t.TempDir()
	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestLoad_UserFileOnly(t *testing.T) {
	xdg := isolateEnv(t)
	writeUserConfig(t, xdg, `
naming      = "{branch}"
auto_setup  = "always"
open_editor = true
editor      = "nvim"
tmux        = true

[gh]
enabled = false

[ui]
color = "never"
`)
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Naming != "{branch}" {
		t.Errorf("Naming = %q, want {branch}", got.Naming)
	}
	if got.AutoSetup != SetupAlways {
		t.Errorf("AutoSetup = %q, want always", got.AutoSetup)
	}
	if !got.OpenEditor || got.Editor != "nvim" || !got.Tmux {
		t.Errorf("editor/tmux not applied: %+v", got)
	}
	if got.GH.Enabled {
		t.Errorf("GH.Enabled = true, want false")
	}
	if got.UI.Color != ColorNever {
		t.Errorf("UI.Color = %q, want never", got.UI.Color)
	}
}

func TestLoad_UserFile_PartialKeepsDefaults(t *testing.T) {
	xdg := isolateEnv(t)
	writeUserConfig(t, xdg, `naming = "{repo}_{branch}"`)
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Naming != "{repo}_{branch}" {
		t.Errorf("Naming = %q", got.Naming)
	}
	// Unspecified keys must keep defaults.
	if got.AutoSetup != SetupPrompt {
		t.Errorf("AutoSetup = %q, want default prompt", got.AutoSetup)
	}
	if !got.GH.Enabled {
		t.Errorf("GH.Enabled should remain default true")
	}
	if got.UI.Color != ColorAuto {
		t.Errorf("UI.Color = %q, want default auto", got.UI.Color)
	}
}

func TestLoad_RepoLocalOverridesUser(t *testing.T) {
	xdg := isolateEnv(t)
	writeUserConfig(t, xdg, `
naming     = "{branch}"
auto_setup = "always"
editor     = "code"
`)
	repo := t.TempDir()
	writeRepoConfig(t, repo, `
naming     = "{repo}-{branch_slug}"
auto_setup = "never"
`)
	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Naming != "{repo}-{branch_slug}" {
		t.Errorf("Naming = %q, want repo-local value", got.Naming)
	}
	if got.AutoSetup != SetupNever {
		t.Errorf("AutoSetup = %q, want never (repo override)", got.AutoSetup)
	}
	// Repo file did not set editor -> user value survives.
	if got.Editor != "code" {
		t.Errorf("Editor = %q, want code (from user file)", got.Editor)
	}
}

func TestLoad_EnvOverridesFiles(t *testing.T) {
	xdg := isolateEnv(t)
	writeUserConfig(t, xdg, `
worktree_dir = "/from/user"
naming       = "{branch}"
auto_setup   = "always"
`)
	repo := t.TempDir()
	writeRepoConfig(t, repo, `
worktree_dir = "/from/repo"
auto_setup   = "prompt"
`)
	t.Setenv("GWT_WORKTREE_DIR", "/from/env")
	t.Setenv("GWT_NAMING", "{repo}/{branch}")
	t.Setenv("GWT_RUN_SETUP", "never")

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.WorktreeDir != "/from/env" {
		t.Errorf("WorktreeDir = %q, want env value", got.WorktreeDir)
	}
	if got.Naming != "{repo}/{branch}" {
		t.Errorf("Naming = %q, want env value", got.Naming)
	}
	if got.AutoSetup != SetupNever {
		t.Errorf("AutoSetup = %q, want never (env)", got.AutoSetup)
	}
}

func TestLoad_EnvRunSetupTriState(t *testing.T) {
	cases := []struct {
		val  string
		want AutoSetup
	}{
		{"1", SetupAlways},
		{"true", SetupAlways},
		{"yes", SetupAlways},
		{"on", SetupAlways},
		{"0", SetupNever},
		{"false", SetupNever},
		{"no", SetupNever},
		{"off", SetupNever},
		{"maybe", SetupPrompt}, // unknown -> leave as configured (default prompt)
	}
	for _, tc := range cases {
		t.Run(tc.val, func(t *testing.T) {
			isolateEnv(t)
			t.Setenv("GWT_RUN_SETUP", tc.val)
			got, err := Load("")
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got.AutoSetup != tc.want {
				t.Errorf("GWT_RUN_SETUP=%q -> AutoSetup %q, want %q", tc.val, got.AutoSetup, tc.want)
			}
		})
	}
}

func TestLoad_EnvEditorSetsOpenEditor(t *testing.T) {
	isolateEnv(t)
	t.Setenv("GWT_EDITOR", "cursor")
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Editor != "cursor" || !got.OpenEditor {
		t.Errorf("got Editor=%q OpenEditor=%v, want cursor/true", got.Editor, got.OpenEditor)
	}
}

func TestLoad_NoColorEnv(t *testing.T) {
	t.Run("NO_COLOR present", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("NO_COLOR", "1")
		got, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.UI.Color != ColorNever {
			t.Errorf("UI.Color = %q, want never", got.UI.Color)
		}
	})
	t.Run("GWT_NO_COLOR truthy", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("GWT_NO_COLOR", "yes")
		got, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.UI.Color != ColorNever {
			t.Errorf("UI.Color = %q, want never", got.UI.Color)
		}
	})
	t.Run("GWT_NO_COLOR falsey leaves color", func(t *testing.T) {
		xdg := isolateEnv(t)
		writeUserConfig(t, xdg, `[ui]
color = "always"`)
		t.Setenv("GWT_NO_COLOR", "0")
		got, err := Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.UI.Color != ColorAlways {
			t.Errorf("UI.Color = %q, want always (GWT_NO_COLOR=0 ignored)", got.UI.Color)
		}
	})
}

func TestLoad_MalformedTOML(t *testing.T) {
	xdg := isolateEnv(t)
	writeUserConfig(t, xdg, "naming = \"unterminated")
	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for malformed TOML")
	}
	if !strings.Contains(err.Error(), "config.toml") {
		t.Errorf("error should name the path, got: %v", err)
	}
}

func TestLoad_MalformedRepoTOML(t *testing.T) {
	isolateEnv(t)
	repo := t.TempDir()
	writeRepoConfig(t, repo, "this is = = not toml")
	_, err := Load(repo)
	if err == nil {
		t.Fatal("expected error for malformed repo TOML")
	}
	if !strings.Contains(err.Error(), ".gwt.toml") {
		t.Errorf("error should name the repo path, got: %v", err)
	}
}

func TestLoad_InvalidEnum(t *testing.T) {
	t.Run("auto_setup", func(t *testing.T) {
		xdg := isolateEnv(t)
		writeUserConfig(t, xdg, `auto_setup = "sometimes"`)
		_, err := Load("")
		if err == nil {
			t.Fatal("expected error for invalid auto_setup")
		}
		if !strings.Contains(err.Error(), "auto_setup") {
			t.Errorf("error should name auto_setup, got: %v", err)
		}
	})
	t.Run("ui.color", func(t *testing.T) {
		xdg := isolateEnv(t)
		writeUserConfig(t, xdg, `[ui]
color = "rainbow"`)
		_, err := Load("")
		if err == nil {
			t.Fatal("expected error for invalid ui.color")
		}
		if !strings.Contains(err.Error(), "color") {
			t.Errorf("error should name color, got: %v", err)
		}
	})
}

func TestUserConfigPath(t *testing.T) {
	t.Run("XDG present and file exists", func(t *testing.T) {
		xdg := isolateEnv(t)
		writeUserConfig(t, xdg, `naming = "x"`)
		path, ok := UserConfigPath()
		if !ok {
			t.Fatal("expected ok=true")
		}
		want := filepath.Join(xdg, "gwt", "config.toml")
		if path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})
	t.Run("XDG present, no file", func(t *testing.T) {
		isolateEnv(t)
		_, ok := UserConfigPath()
		if ok {
			t.Error("expected ok=false when file missing")
		}
	})
	t.Run("falls back to HOME/.config", func(t *testing.T) {
		isolateEnv(t)
		os.Unsetenv("XDG_CONFIG_HOME")
		home := t.TempDir()
		t.Setenv("HOME", home)
		dir := filepath.Join(home, ".config", "gwt")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`naming="x"`), 0o644); err != nil {
			t.Fatal(err)
		}
		path, ok := UserConfigPath()
		if !ok {
			t.Fatal("expected ok=true via HOME fallback")
		}
		if path != filepath.Join(dir, "config.toml") {
			t.Errorf("path = %q", path)
		}
	})
}
