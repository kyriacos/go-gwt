package setup

import "context"

// Phase identifies a lifecycle point at which user-config hooks run.
type Phase string

const (
	// PostCreate fires after a worktree is created; cwd = the new worktree.
	PostCreate Phase = "post_create"
	// PreRemove fires before a worktree is removed; cwd = the worktree about
	// to be removed.
	PreRemove Phase = "pre_remove"
)

// RunHooks executes the user-configured lifecycle hooks for phase. cwd is the
// worktree the hooks operate in (the new worktree for PostCreate, the target
// worktree for PreRemove); root is the main worktree path, exported as
// ROOT_WORKTREE_PATH.
//
// Unlike repo setup commands, hooks come from the user's own config, so they
// are TRUSTED and run without any consent prompt. Individual failures are
// warnings and do not abort the sequence. RunHooks returns an error only for a
// caller mistake (an unknown phase).
func (r *Runner) RunHooks(ctx context.Context, phase Phase, cwd, root string) error {
	var cmds []string
	switch phase {
	case PostCreate:
		cmds = r.Cfg.Hooks.PostCreate
	case PreRemove:
		cmds = r.Cfg.Hooks.PreRemove
	default:
		return errUnknownPhase(phase)
	}

	if len(cmds) == 0 {
		return nil
	}
	// Trusted: no consent check. Reuse the same execution path as setup so
	// echoing, env, cwd, and continue-on-failure behavior are identical.
	r.runCommands(ctx, cmds, cwd, root)
	return nil
}

type errUnknownPhase Phase

func (e errUnknownPhase) Error() string { return "setup: unknown hook phase " + string(e) }
