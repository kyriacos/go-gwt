package git

import "strings"

// shortSHA truncates a full object name to the conventional 7 characters. SHAs
// shorter than 7 chars (or empty) are returned unchanged.
func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

// parseWorktreeList parses the output of `git worktree list --porcelain`.
//
// Records are separated by blank lines. Within a record the recognized lines
// are:
//
//	worktree <path>
//	HEAD <sha>
//	branch refs/heads/<name>
//	bare
//	detached
//
// The first record is the main worktree and is marked IsMain.
func parseWorktreeList(out string) []Worktree {
	var (
		wts     []Worktree
		cur     Worktree
		started bool
	)
	flush := func() {
		if started {
			wts = append(wts, cur)
		}
		cur = Worktree{}
		started = false
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			// A new "worktree" line begins a new record even without a blank
			// line separating it (defensive against trailing output).
			if started {
				flush()
			}
			cur.Path = strings.TrimPrefix(line, "worktree ")
			started = true
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = shortSHA(strings.TrimPrefix(line, "HEAD "))
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "detached":
			cur.Detached = true
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			cur.Prunable = true
		}
	}
	flush()
	if len(wts) > 0 {
		wts[0].IsMain = true
	}
	return wts
}

// parseStatusV2 parses `git status --porcelain=v2 --branch` output into a
// Status. Header lines start with "# "; entry lines start with "1", "2", "u",
// or "?". The XY field of changed-entry lines indicates staged (X) and
// unstaged (Y) state.
func parseStatusV2(out string) Status {
	var s Status
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			a, b := parseAheadBehind(strings.TrimPrefix(line, "# branch.ab "))
			s.Ahead, s.Behind = a, b
		case strings.HasPrefix(line, "# "):
			// other header line, ignore
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			// Ordinary (1) or renamed/copied (2) changed entry. The XY field
			// is the second whitespace-separated token.
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			xy := fields[1]
			if len(xy) == 2 {
				if xy[0] != '.' {
					s.Staged++
				}
				if xy[1] != '.' {
					s.Unstaged++
				}
			}
		case strings.HasPrefix(line, "u "):
			// Unmerged entry: count as both staged and unstaged conflict work.
			s.Unstaged++
		case strings.HasPrefix(line, "? "):
			s.Untracked++
		}
	}
	s.Dirty = s.Staged > 0 || s.Unstaged > 0 || s.Untracked > 0
	return s
}

// parseAheadBehind parses a "+A -B" pair from a `# branch.ab` line.
func parseAheadBehind(s string) (ahead, behind int) {
	for _, tok := range strings.Fields(s) {
		if len(tok) < 2 {
			continue
		}
		n := atoi(tok[1:])
		switch tok[0] {
		case '+':
			ahead = n
		case '-':
			behind = n
		}
	}
	return ahead, behind
}

// atoi parses a non-negative decimal, returning 0 on any malformed input.
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
