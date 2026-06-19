package git

import (
	"fmt"
	"strings"
)

// git runs a git command in dir and returns trimmed stdout. On failure it
// wraps the error with the given operation label and the command's stderr (when
// any), so callers get actionable context.
func (r *CmdRepo) git(dir, op string, args ...string) (string, error) {
	stdout, stderr, err := r.run.Run(r.ctx, dir, "git", args...)
	if err != nil {
		if msg := strings.TrimSpace(string(stderr)); msg != "" {
			return "", fmt.Errorf("%s: %w: %s", op, err, msg)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return strings.TrimRight(string(stdout), "\n"), nil
}
