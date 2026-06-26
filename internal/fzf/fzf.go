// Package fzf wraps the fzf binary for fast interactive pickers that match the
// legacy bash gwt UX. When fzf is not installed, callers fall back to the
// built-in TUI pickers.
package fzf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kyriacos/go-gwt/internal/git"
)

// ErrUnavailable is returned when the fzf binary is not on PATH.
var ErrUnavailable = errors.New("fzf: not available")

// Available reports whether the fzf binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// pick runs fzf over stdin lines and returns the selected line, or "" when the
// user cancels. preview is passed to fzf's --preview flag ({1} = whole line).
func pick(prompt, preview string, multi bool, stdin string) (string, error) {
	if !Available() {
		return "", ErrUnavailable
	}
	args := []string{
		"--ansi",
		"--reverse",
		"--delimiter=\t",
		"--with-nth=1",
		"--height=50%",
		"--prompt=" + prompt,
		"--preview-window=right:50%:wrap",
	}
	if preview != "" {
		args = append(args, "--preview="+preview)
	}
	if multi {
		args = append(args, "--multi")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(stdin)

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		cmd.Stderr = os.Stderr
	} else {
		defer tty.Close()
		cmd.Stderr = tty
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return "", nil // user cancelled
		}
		return "", fmt.Errorf("fzf: %w", err)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// WorktreeLine is one row for the worktree picker: colored display + raw path.
type WorktreeLine struct {
	Display string
	Path    string
}

// PickWorktree shows a single-select fzf picker. Returns the chosen path or ""
// when cancelled.
func PickWorktree(lines []WorktreeLine) (string, error) {
	if len(lines) == 0 {
		return "", nil
	}
	var b strings.Builder
	for _, ln := range lines {
		fmt.Fprintf(&b, "%s\t%s\n", ln.Display, ln.Path)
	}
	out, err := pick("worktree> ", "git -C {2} log --oneline --graph --decorate --color=always -n 10 2>/dev/null", false, b.String())
	if err != nil || out == "" {
		return "", err
	}
	// Recover path from the tab-separated selection.
	if i := strings.LastIndexByte(out, '\t'); i >= 0 {
		return out[i+1:], nil
	}
	return out, nil
}

// BranchLine is one row for the branch picker.
type BranchLine struct {
	Display string
	Name    string
}

// PickBranch shows a single-select branch picker. When populateVerb is non-empty
// and the user selects a branch, Populate is set to "gwt <verb> <branch>" so
// the shell wrapper can park the command in the line buffer (bash gwt parity).
// Returns the branch name, populate string (if any), and an error.
func PickBranch(populateVerb string, lines []BranchLine) (branch, populate string, err error) {
	if len(lines) == 0 {
		return "", "", nil
	}
	var b strings.Builder
	for _, ln := range lines {
		fmt.Fprintf(&b, "%s\t%s\n", ln.Display, ln.Name)
	}
	out, err := pick(populateVerb+"> ",
		"git log --oneline --graph --decorate --color=always -n 10 {2}",
		false, b.String())
	if err != nil || out == "" {
		return "", "", err
	}
	name := out
	if i := strings.LastIndexByte(out, '\t'); i >= 0 {
		name = out[i+1:]
	}
	if populateVerb != "" {
		return "", fmt.Sprintf("gwt %s %s", populateVerb, name), nil
	}
	return name, "", nil
}

// PickWorktreesMulti shows a multi-select fzf picker. Returns selected paths.
func PickWorktreesMulti(lines []WorktreeLine, header string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	var b strings.Builder
	for _, ln := range lines {
		fmt.Fprintf(&b, "%s\t%s\n", ln.Display, ln.Path)
	}
	args := []string{
		"--ansi", "--reverse", "--delimiter=\t", "--with-nth=1",
		"--height=60%", "--multi", "--prompt=clean> ",
		"--preview-window=right:50%:wrap",
		"--preview=git -C {2} log --oneline --graph --decorate --color=always -n 10 2>/dev/null",
	}
	if header != "" {
		args = append(args, "--header="+header)
	}
	if !Available() {
		return nil, ErrUnavailable
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(b.String())
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		cmd.Stderr = os.Stderr
	} else {
		defer tty.Close()
		cmd.Stderr = tty
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return nil, nil
		}
		return nil, fmt.Errorf("fzf: %w", err)
	}
	var paths []string
	for line := range strings.SplitSeq(strings.TrimRight(stdout.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		if i := strings.LastIndexByte(line, '\t'); i >= 0 {
			paths = append(paths, line[i+1:])
		}
	}
	return paths, nil
}

// FormatWorktreeLine builds the colored fzf display column for one worktree,
// mirroring `gwt ls` / the bash gwt WT_AWK fzf mode.
func FormatWorktreeLine(wt git.Worktree, state, cur string, maxBranch, maxPath int, styles FzfStyles) WorktreeLine {
	isCur := filepath.Clean(wt.Path) == filepath.Clean(cur)
	marker := "  "
	if isCur {
		marker = styles.Cyan.Render("* ")
	}
	cell := branchCell(wt, state)
	branch := styles.ForState(state).Render(fmt.Sprintf("%-*s", maxBranch, cell))
	pathStyle := styles.Bold
	if isCur {
		pathStyle = styles.CyanBold
	}
	path := pathStyle.Render(fmt.Sprintf("%-*s", maxPath, wt.Path))
	head := styles.Dim.Render(wt.Head)
	display := fmt.Sprintf("%s%s  %s  %s", marker, branch, path, head)
	return WorktreeLine{Display: display, Path: wt.Path}
}

func branchCell(wt git.Worktree, state string) string {
	if wt.Branch == "" {
		return "(" + state + ")"
	}
	if state == "gone" || state == "missing" {
		return "[" + wt.Branch + "] (" + state + ")"
	}
	return "[" + wt.Branch + "]"
}
