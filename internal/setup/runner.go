// Package setup runs the commands that prepare a freshly created worktree, and
// the lifecycle hooks fired around create/remove. It ports the bash `gwt`
// run_setup behavior and extends it with configurable hooks.
//
// # Trust model
//
// There are two distinct sources of commands, with different trust:
//
//   - Repo-provided setup commands live in <root>/.cursor/worktrees.json under
//     the "setup-worktree" key. They ship with the repository, so in a clone you
//     did not write they are UNTRUSTED: they never run without consent. Consent
//     precedence (highest first): an explicit per-invocation Decision (from a
//     --run-setup / --no-setup flag) > Cfg.AutoSetup (always | never) >
//     an interactive prompt (only when a tty exists; default No when none).
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
//	_ = r.RunSetup(ctx, newPath, root, decision)
//	// before `git worktree remove`:
//	_ = r.RunHooks(ctx, setup.PreRemove, targetPath, root)
//
// All commands run through the injected exec.Runner via `sh -c "<cmd>"`, with
// cwd set appropriately and ROOT_WORKTREE_PATH exported. Individual command
// failures are reported as warnings and do not abort the sequence, matching the
// bash tool. RunSetup and RunHooks return a non-nil error only for setup-level
// problems (e.g. an unreadable worktrees.json), never for a failing user
// command.
package setup

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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

// Decision is an explicit per-invocation choice that overrides Cfg.AutoSetup.
// It models the --run-setup / --no-setup flags (and $GWT_RUN_SETUP) of the bash
// tool. Default means "no explicit choice; fall through to config / prompt".
type Decision int

const (
	// DecisionDefault defers to Cfg.AutoSetup, then to an interactive prompt.
	DecisionDefault Decision = iota
	// DecisionYes runs repo setup commands without prompting.
	DecisionYes
	// DecisionNo skips repo setup commands without prompting.
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

// confirm reports the user's consent to run repo setup commands. It honors the
// no-tty case by defaulting to No.
func (r *Runner) confirm(cmds []string) bool {
	if !ui.HasTTY() {
		ui.Warn(".cursor/worktrees.json defines setup commands, but there is no terminal to confirm; skipping.")
		ui.Dim("Re-run with --run-setup to execute them.")
		return false
	}
	ui.Warn("This repo's .cursor/worktrees.json wants to run these setup commands:")
	for _, c := range cmds {
		ui.Dim("  $ %s", c)
	}
	return ui.Confirm("Run them?", false)
}

// consent resolves whether repo setup commands may run, applying the precedence:
// explicit Decision > Cfg.AutoSetup > interactive prompt (default No w/o tty).
func (r *Runner) consent(decision Decision, cmds []string) bool {
	switch decision {
	case DecisionYes:
		return true
	case DecisionNo:
		return false
	}
	switch r.Cfg.AutoSetup {
	case config.SetupAlways:
		return true
	case config.SetupNever:
		return false
	}
	return r.confirm(cmds)
}

// RunSetup prepares a newly created worktree at newPath. root is the main
// worktree path. decision overrides config for this invocation.
//
// If <root>/.cursor/worktrees.json defines setup-worktree commands, they are
// run (subject to consent) in newPath with ROOT_WORKTREE_PATH exported and the
// $ROOT_WORKTREE_PATH token substituted. With no such config, RunSetup falls
// back to copying a top-level .env from root into newPath when present and not
// already there.
func (r *Runner) RunSetup(ctx context.Context, newPath, root string, decision Decision) error {
	cmds, err := loadSetupCommands(root)
	if err != nil {
		return err
	}

	if len(cmds) > 0 {
		if !r.consent(decision, cmds) {
			ui.Dim("Skipped setup-worktree commands.")
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

// loadSetupCommands reads <root>/.cursor/worktrees.json and returns the
// "setup-worktree" commands with $ROOT_WORKTREE_PATH substituted for root.
// A missing file yields no commands and no error. A malformed file is an error.
func loadSetupCommands(root string) ([]string, error) {
	cfgPath := filepath.Join(root, ".cursor", "worktrees.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var doc struct {
		SetupWorktree []string `json:"setup-worktree"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	cmds := make([]string, 0, len(doc.SetupWorktree))
	for _, c := range doc.SetupWorktree {
		if strings.TrimSpace(c) == "" {
			continue
		}
		cmds = append(cmds, strings.ReplaceAll(c, rootToken, root))
	}
	return cmds, nil
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
