package worktree

import (
	"path/filepath"
	"strings"
)

// defaultTemplate is used when Cfg.Naming is empty.
const defaultTemplate = "{repo}-{branch}"

// applyTemplate renders the naming template for a branch, given the main
// worktree path (used to derive {repo}). Recognised tokens:
//
//	{repo}        basename of the main worktree
//	{branch}      the branch name with '/' replaced by '-'
//	{branch_slug} an aggressively sanitized branch name (see slugify)
//
// An empty template falls back to defaultTemplate.
func applyTemplate(template, mainWorktree, branch string) string {
	if strings.TrimSpace(template) == "" {
		template = defaultTemplate
	}
	repo := filepath.Base(mainWorktree)
	r := strings.NewReplacer(
		"{repo}", repo,
		"{branch}", branchDir(branch),
		"{branch_slug}", slugify(branch),
	)
	return r.Replace(template)
}

// branchDir maps a branch name to a directory-friendly form by replacing path
// separators ('/') with '-'. It keeps the rest of the name intact so that e.g.
// "feature/foo" becomes "feature-foo".
func branchDir(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// slugify produces an aggressively sanitized, lowercase variant of s suitable
// for filesystem names: every run of non-alphanumeric characters collapses to a
// single '-', leading/trailing '-' are trimmed.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		// Non-alphanumeric: emit a single '-' for any run.
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
