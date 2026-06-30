// Package setup runs the commands that prepare a freshly created worktree, and
// the lifecycle hooks fired around create/remove. It ports the bash `gwt`
// run_setup behavior and extends it with configurable hooks.
//
// # Trust model
//
// There are three distinct sources of commands, with different trust:
//
//   - Cursor repo setup commands live in <root>/.cursor/worktrees.json under
//     the "setup-worktree" key. They ship with the repository, so in a clone you
//     did not write they are UNTRUSTED: they never run without consent. Consent
//     precedence (highest first): an explicit per-invocation Decision (from a
//     --cursor-run-setup / --cursor-no-setup flag) > Cfg.CursorWorktreeSetup()
//     (always | never) > an interactive prompt (only when a tty exists;
//     default No when none).
//
//   - Claude Code worktree preparation copies gitignored paths from
//     <root>/.worktreeinclude into the new worktree (same rules as Claude Code).
//     These patterns ship with the repository, so they are UNTRUSTED and
//     consent-gated via Cfg.ClaudeWorktreeSetup() with the same precedence as
//     Cursor setup. WorktreeCreate hooks are not run — go-gwt already created
//     the worktree via git.
//
//   - User-config hooks (Cfg.Hooks.PostCreate / PreRemove) come from the user's
//     own ~/.config/gwt/config.toml, so they are TRUSTED and run without any
//     prompting. The user authored them; asking would be noise.
//
// # Public API
//
// The worktree service wires this in after creating a worktree and before
// removing one:
//
//	r := setup.New(runner, cfg)
//	// after `git worktree add`:
//	_ = r.RunHooks(ctx, setup.PostCreate, newPath, root)
//	_ = r.RunCursorSetup(ctx, newPath, root, cursorDecision)
//	_ = r.RunClaudeSetup(ctx, newPath, root, claudeDecision)
//	// before `git worktree remove`:
//	_ = r.RunHooks(ctx, setup.PreRemove, targetPath, root)
//
// All commands run through the injected exec.Runner via `sh -c "<cmd>"`, with
// cwd set appropriately and ROOT_WORKTREE_PATH exported. Individual command
// failures are reported as warnings and do not abort the sequence, matching the
// bash tool. RunCursorSetup and RunHooks return a non-nil error only for
// setup-level problems (e.g. an unreadable worktrees.json), never for a failing
// user command.
package setup

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/ui"
)

// rootEnvVar is the environment variable exported for every command and the
// literal token substituted in setup-worktree commands.
const rootEnvVar = "ROOT_WORKTREE_PATH"

// rootToken is the literal placeholder replaced with the main worktree path in
// setup-worktree commands.
const rootToken = "$ROOT_WORKTREE_PATH"

// Decision is an explicit per-invocation choice that overrides the configured
// worktree_setup mode for an IDE integration. It models the --cursor-run-setup
// / --cursor-no-setup flags (and $GWT_CURSOR_RUN_SETUP) of the bash tool.
// Default means "no explicit choice; fall through to config / prompt".
type Decision int

const (
	// DecisionDefault defers to config, then to an interactive prompt.
	DecisionDefault Decision = iota
	// DecisionYes runs IDE setup without prompting.
	DecisionYes
	// DecisionNo skips IDE setup without prompting.
	DecisionNo
)

// Runner runs setup commands and lifecycle hooks for worktrees.
type Runner struct {
	Run exec.Runner
	Cfg config.Config
}

// New constructs a Runner.
func New(run exec.Runner, cfg config.Config) *Runner {
	return &Runner{Run: run, Cfg: cfg}
}

func (r *Runner) confirmCursor(cmds []string) bool {
	if !ui.HasTTY() {
		ui.Warn(".cursor/worktrees.json defines setup commands, but there is no terminal to confirm; skipping.")
		ui.Dim("Re-run with --cursor-run-setup to execute them.")
		return false
	}
	ui.Warn("This repo's .cursor/worktrees.json wants to run these setup commands:")
	for _, c := range cmds {
		ui.Dim("  $ %s", c)
	}
	return ui.Confirm("Run Cursor worktree setup?", false)
}

func (r *Runner) consent(decision Decision, configured config.WorktreeSetup, cmds []string, confirmFn func([]string) bool) bool {
	switch decision {
	case DecisionYes:
		return true
	case DecisionNo:
		return false
	}
	switch configured {
	case config.SetupAlways:
		return true
	case config.SetupNever:
		return false
	}
	return confirmFn(cmds)
}

// RunCursorSetup prepares a newly created worktree at newPath. root is the main
// worktree path. decision overrides [cursor].worktree_setup for this invocation.
//
// If <root>/.cursor/worktrees.json defines setup-worktree commands, they are
// run (subject to consent) in newPath with ROOT_WORKTREE_PATH exported and the
// $ROOT_WORKTREE_PATH token substituted. With no such config, RunCursorSetup
// falls back to copying a top-level .env from root into newPath when present and
// not already there (this fallback is not consent-gated).
func (r *Runner) RunCursorSetup(ctx context.Context, newPath, root string, decision Decision) error {
	cmds, err := loadCursorSetupCommands(root)
	if err != nil {
		return err
	}

	if len(cmds) > 0 {
		if !r.consent(decision, r.Cfg.CursorWorktreeSetup(), cmds, r.confirmCursor) {
			ui.Dim("Skipped Cursor setup-worktree commands.")
			return nil
		}
		ui.Dim("Running setup-worktree from .cursor/worktrees.json ...")
		r.runCommands(ctx, cmds, newPath, root)
		return nil
	}

	// Fallback: copy a top-level .env if the repo has one and the new worktree
	// does not already have it.
	return copyEnvFallback(root, newPath)
}

// loadCursorSetupCommands reads <root>/.cursor/worktrees.json and returns setup
// commands with $ROOT_WORKTREE_PATH substituted for root. Cursor accepts each
// setup key as either a string (one command or script path) or an array of
// commands; on Unix, setup-worktree-unix takes precedence over setup-worktree.
// A missing file yields no commands and no error. A malformed file is an error.
func loadCursorSetupCommands(root string) ([]string, error) {
	cfgPath := filepath.Join(root, ".cursor", "worktrees.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var doc struct {
		SetupWorktreeUnix    json.RawMessage `json:"setup-worktree-unix"`
		SetupWorktreeWindows json.RawMessage `json:"setup-worktree-windows"`
		SetupWorktree        json.RawMessage `json:"setup-worktree"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	raw := pickCursorSetupRaw(doc.SetupWorktreeUnix, doc.SetupWorktreeWindows, doc.SetupWorktree)
	rawCmds, err := normalizeSetupCommands(raw)
	if err != nil {
		return nil, err
	}

	cmds := make([]string, 0, len(rawCmds))
	for _, c := range rawCmds {
		if strings.TrimSpace(c) == "" {
			continue
		}
		cmds = append(cmds, strings.ReplaceAll(c, rootToken, root))
	}
	return cmds, nil
}

func pickCursorSetupRaw(unix, windows, fallback json.RawMessage) json.RawMessage {
	if runtime.GOOS == "windows" {
		if len(windows) > 0 {
			return windows
		}
	} else if len(unix) > 0 {
		return unix
	}
	return fallback
}

func normalizeSetupCommands(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		if strings.TrimSpace(one) == "" {
			return nil, nil
		}
		return []string{one}, nil
	}

	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		out := make([]string, 0, len(many))
		for _, c := range many {
			if strings.TrimSpace(c) == "" {
				continue
			}
			out = append(out, c)
		}
		return out, nil
	}

	return nil, errors.New("setup-worktree value must be a string or array of strings")
}

// runCommands executes each command in cwd via `sh -c`, with ROOT_WORKTREE_PATH
// set to root. Each command is echoed; individual failures are warnings and do
// not stop the sequence.
func (r *Runner) runCommands(ctx context.Context, cmds []string, cwd, root string) {
	prevRoot, hadRoot := os.LookupEnv(rootEnvVar)
	_ = os.Setenv(rootEnvVar, root)
	defer func() {
		if hadRoot {
			_ = os.Setenv(rootEnvVar, prevRoot)
		} else {
			_ = os.Unsetenv(rootEnvVar)
		}
	}()

	for _, cmd := range cmds {
		ui.Dim("  $ %s", cmd)
		_, stderr, err := r.Run.Run(ctx, cwd, "sh", "-c", cmd)
		if len(stderr) > 0 {
			ui.Dim("%s", string(stderr))
		}
		if err != nil {
			ui.Warn("  (step failed, continuing): %s", cmd)
		}
	}
}

// copyEnvFallback copies <root>/.env to <newPath>/.env when the source exists
// and the destination does not. Absence of a source .env is a no-op.
func copyEnvFallback(root, newPath string) error {
	src := filepath.Join(root, ".env")
	dst := filepath.Join(newPath, ".env")

	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil // already present; do not clobber
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return err
	}
	ui.Dim("Copied .env from main worktree.")
	return nil
}
