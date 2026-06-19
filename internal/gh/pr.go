package gh

import (
	"encoding/json"
	"fmt"
	"strings"
)

// prJSON mirrors the shape of `gh pr list --json ...` / `gh pr view --json ...`.
type prJSON struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
	HeadRefOid  string `json:"headRefOid"`
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}

func (p prJSON) toPR() PR {
	return PR{
		Number: p.Number,
		Title:  p.Title,
		Author: p.Author.Login,
		Branch: p.HeadRefName,
		State:  p.State,
		Draft:  p.IsDraft,
	}
}

// ListPRs returns up to 50 open PRs for the current repo. It returns
// ErrUnavailable when gh is missing or unauthenticated.
func (c *CmdClient) ListPRs() ([]PR, error) {
	if !c.Available() {
		return nil, ErrUnavailable
	}
	out, err := c.runGH("pr", "list", "--json", "number,title,author,headRefName,state,isDraft", "--limit", "50")
	if err != nil {
		return nil, err
	}
	var raw []prJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("gh pr list: parse json: %w", err)
	}
	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		prs = append(prs, r.toPR())
	}
	return prs, nil
}

// Checkout ensures a local branch for the given PR exists and returns its name.
// The caller is responsible for creating the worktree from that branch;
// Checkout itself only guarantees the local branch exists.
//
// Branch discovery without disrupting the user's checkout:
//
// `gh pr checkout <pr>` is the canonical way to obtain a PR's branch, but it
// switches the current working tree to that branch, which is undesirable
// mid-flow (the worktree helper wants to create a *new* worktree from the
// branch, not move the user's HEAD). gh offers no flag to fetch-without-switch:
// `--detach` still moves HEAD (to a detached state), and there is no gh
// equivalent of `git fetch origin pull/<n>/head:<branch>`.
//
// The approach here: resolve the branch name first via `gh pr view <pr> --json
// headRefName` (read-only, no checkout), then run `gh pr checkout <pr>` to
// fetch/create the local branch. Tradeoff: `gh pr checkout` does switch the
// current HEAD as a side effect. We accept this because it is the only gh
// command that reliably materializes the PR branch locally (including
// cross-fork PRs), and the integration layer (cmd/worktree) immediately creates
// a worktree from the returned branch and the wrapper cd's the shell elsewhere,
// so the transient switch is not user-visible. If gh ever exposes a
// non-switching fetch, prefer it. The returned branch name is taken from the
// read-only view call, so it is correct regardless of checkout side effects.
func (c *CmdClient) Checkout(pr int) (string, error) {
	if !c.Available() {
		return "", ErrUnavailable
	}
	prArg := fmt.Sprintf("%d", pr)

	// Resolve the branch name first (read-only, does not touch HEAD).
	out, err := c.runGH("pr", "view", prArg, "--json", "headRefName,headRefOid")
	if err != nil {
		return "", err
	}
	var view prJSON
	if err := json.Unmarshal(out, &view); err != nil {
		return "", fmt.Errorf("gh pr view %d: parse json: %w", pr, err)
	}
	branch := strings.TrimSpace(view.HeadRefName)
	if branch == "" {
		return "", fmt.Errorf("gh pr view %d: empty headRefName", pr)
	}

	// Materialize the local branch. This switches the current HEAD as a side
	// effect (see doc comment); the caller does not rely on the current
	// checkout afterwards.
	if _, err := c.runGH("pr", "checkout", prArg); err != nil {
		return "", err
	}
	return branch, nil
}
