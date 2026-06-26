package setup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/testutil"
	"github.com/kyriacos/go-gwt/internal/ui"
)

func writeWorktreeInclude(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, worktreeIncludeName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunClaudeSetup_CopiesGitignoredMatches(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	repo.WriteFile(".gitignore", ".env\n.env.local\n")
	repo.WriteFile(".env", "ROOT=1\n")
	repo.WriteFile(".env.local", "LOCAL=1\n")
	repo.WriteFile("tracked.txt", "tracked\n")
	repo.Git("add", ".gitignore", "tracked.txt")
	repo.Git("commit", "-m", "add gitignore")
	writeWorktreeInclude(t, repo.Dir, ".env\n.env.local\n")

	newPath := t.TempDir()
	r := New(exec.New(), config.Config{})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionYes); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}

	for _, name := range []string{".env", ".env.local"} {
		got, err := os.ReadFile(filepath.Join(newPath, name))
		if err != nil {
			t.Fatalf("expected %s copied: %v", name, err)
		}
		want, _ := os.ReadFile(filepath.Join(repo.Dir, name))
		if string(got) != string(want) {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(newPath, "tracked.txt")); !os.IsNotExist(err) {
		t.Error("tracked file should not be copied")
	}
}

func TestRunClaudeSetup_SkipsTrackedEvenIfListed(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	writeWorktreeInclude(t, repo.Dir, "README.md\n")

	newPath := t.TempDir()
	r := New(exec.New(), config.Config{})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionYes); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, "README.md")); !os.IsNotExist(err) {
		t.Error("tracked README.md should not be copied even when listed")
	}
}

func TestRunClaudeSetup_NoClobber(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	repo.WriteFile(".gitignore", ".env\n")
	repo.WriteFile(".env", "ROOT=1\n")
	repo.Git("add", ".gitignore")
	repo.Git("commit", "-m", "gitignore")
	writeWorktreeInclude(t, repo.Dir, ".env\n")

	newPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(newPath, ".env"), []byte("EXISTING=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := New(exec.New(), config.Config{})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionYes); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(newPath, ".env"))
	if string(got) != "EXISTING=1\n" {
		t.Errorf(".env was clobbered: %q", got)
	}
}

func TestRunClaudeSetup_DecisionPrecedence(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		decision Decision
		mode     config.WorktreeSetup
		wantCopy bool
	}{
		{"explicit yes overrides never", DecisionYes, config.SetupNever, true},
		{"explicit no overrides always", DecisionNo, config.SetupAlways, false},
		{"config always", DecisionDefault, config.SetupAlways, true},
		{"config never", DecisionDefault, config.SetupNever, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := testutil.NewRepo(t)
			repo.WriteFile(".gitignore", ".env\n")
			repo.WriteFile(".env", "KEY=1\n")
			repo.Git("add", ".gitignore")
			repo.Git("commit", "-m", "gitignore")
			writeWorktreeInclude(t, repo.Dir, ".env\n")

			newPath := t.TempDir()
			r := New(exec.New(), config.Config{Claude: config.Claude{WorktreeSetup: tc.mode}})
			if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, tc.decision); err != nil {
				t.Fatalf("RunClaudeSetup: %v", err)
			}
			_, err := os.Stat(filepath.Join(newPath, ".env"))
			gotCopy := err == nil
			if gotCopy != tc.wantCopy {
				t.Fatalf("copied=%v want %v", gotCopy, tc.wantCopy)
			}
		})
	}
}

func TestRunClaudeSetup_PromptNoTTY(t *testing.T) {
	if ui.HasTTY() {
		t.Skip("a tty is attached; cannot deterministically test the no-tty default")
	}
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	repo.WriteFile(".gitignore", ".env\n")
	repo.WriteFile(".env", "KEY=1\n")
	repo.Git("add", ".gitignore")
	repo.Git("commit", "-m", "gitignore")
	writeWorktreeInclude(t, repo.Dir, ".env\n")

	newPath := t.TempDir()
	r := New(exec.New(), config.Config{Claude: config.Claude{WorktreeSetup: config.SetupPrompt}})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionDefault); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, ".env")); !os.IsNotExist(err) {
		t.Error("expected nothing copied without a tty")
	}
}

func TestRunClaudeSetup_SkipsNonGitignoredListed(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	repo.WriteFile("local-only.txt", "local\n")
	writeWorktreeInclude(t, repo.Dir, "local-only.txt\n")

	newPath := t.TempDir()
	r := New(exec.New(), config.Config{})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionYes); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, "local-only.txt")); !os.IsNotExist(err) {
		t.Error("untracked non-gitignored file should not be copied")
	}
}

func TestRunClaudeSetup_MissingInclude(t *testing.T) {
	ctx := context.Background()
	repo := testutil.NewRepo(t)
	newPath := t.TempDir()
	r := New(exec.New(), config.Config{})
	if err := r.RunClaudeSetup(ctx, newPath, repo.Dir, DecisionDefault); err != nil {
		t.Fatalf("RunClaudeSetup: %v", err)
	}
}
