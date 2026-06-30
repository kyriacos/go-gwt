package setup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kyriacos/go-gwt/internal/config"
	"github.com/kyriacos/go-gwt/internal/exec"
	"github.com/kyriacos/go-gwt/internal/ui"
)

// writeWorktreesJSON writes a .cursor/worktrees.json under root with the given
// raw JSON body.
func writeWorktreesJSON(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".cursor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "worktrees.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSetupCommands_StringValue(t *testing.T) {
	root := t.TempDir()
	writeWorktreesJSON(t, root, `{
	  "setup-worktree": "bash .cursor/setup-worktree-unix.sh"
	}`)

	cmds, err := loadCursorSetupCommands(root)
	if err != nil {
		t.Fatalf("loadCursorSetupCommands: %v", err)
	}
	want := []string{"bash .cursor/setup-worktree-unix.sh"}
	if len(cmds) != len(want) {
		t.Fatalf("got %d cmds %v, want %d %v", len(cmds), cmds, len(want), want)
	}
	for i := range want {
		if cmds[i] != want[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, cmds[i], want[i])
		}
	}
}

func TestLoadSetupCommands_UnixPrecedence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix key precedence is not tested on windows")
	}
	root := t.TempDir()
	writeWorktreesJSON(t, root, `{
	  "setup-worktree-unix": ["echo unix"],
	  "setup-worktree": ["echo generic"]
	}`)

	cmds, err := loadCursorSetupCommands(root)
	if err != nil {
		t.Fatalf("loadCursorSetupCommands: %v", err)
	}
	want := []string{"echo unix"}
	if len(cmds) != len(want) || cmds[0] != want[0] {
		t.Fatalf("got %v, want %v", cmds, want)
	}
}

func TestLoadSetupCommands_Substitution(t *testing.T) {
	root := t.TempDir()
	writeWorktreesJSON(t, root, `{
	  "setup-worktree": [
	    "ln -s $ROOT_WORKTREE_PATH/node_modules node_modules",
	    "cp $ROOT_WORKTREE_PATH/.env .env",
	    "",
	    "echo done"
	  ]
	}`)

	cmds, err := loadCursorSetupCommands(root)
	if err != nil {
		t.Fatalf("loadCursorSetupCommands: %v", err)
	}
	want := []string{
		"ln -s " + root + "/node_modules node_modules",
		"cp " + root + "/.env .env",
		"echo done",
	}
	if len(cmds) != len(want) {
		t.Fatalf("got %d cmds %v, want %d %v", len(cmds), cmds, len(want), want)
	}
	for i := range want {
		if cmds[i] != want[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, cmds[i], want[i])
		}
	}
}

func TestLoadSetupCommands_MissingFile(t *testing.T) {
	root := t.TempDir()
	cmds, err := loadCursorSetupCommands(root)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected no commands, got %v", cmds)
	}
}

func TestLoadSetupCommands_Malformed(t *testing.T) {
	root := t.TempDir()
	writeWorktreesJSON(t, root, `{ not valid json `)
	if _, err := loadCursorSetupCommands(root); err == nil {
		t.Fatal("expected error for malformed json, got nil")
	}
}

// fakeWithDefault returns a Fake that succeeds for any sh -c command and
// records calls.
func fakeWithDefault() *exec.Fake {
	return &exec.Fake{Default: &exec.FakeResult{}}
}

func shCalls(f *exec.Fake) []string {
	return append([]string(nil), f.Calls...)
}

func TestRunCursorSetup_DecisionPrecedence(t *testing.T) {
	ctx := context.Background()
	body := `{"setup-worktree": ["echo hi", "echo bye"]}`

	tests := []struct {
		name     string
		decision Decision
		mode     config.WorktreeSetup
		wantRun  bool
	}{
		{"explicit yes overrides never", DecisionYes, config.SetupNever, true},
		{"explicit no overrides always", DecisionNo, config.SetupAlways, false},
		{"config always", DecisionDefault, config.SetupAlways, true},
		{"config never", DecisionDefault, config.SetupNever, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			newPath := t.TempDir()
			writeWorktreesJSON(t, root, body)

			f := fakeWithDefault()
			r := New(f, config.Config{Cursor: config.Cursor{WorktreeSetup: tc.mode}})
			if err := r.RunCursorSetup(ctx, newPath, root, tc.decision); err != nil {
				t.Fatalf("RunCursorSetup: %v", err)
			}

			ran := len(shCalls(f)) > 0
			if ran != tc.wantRun {
				t.Fatalf("ran=%v want %v (calls=%v)", ran, tc.wantRun, f.Calls)
			}
			if tc.wantRun {
				wantFirst := exec.Key("sh", "-c", "echo hi")
				if f.Calls[0] != wantFirst {
					t.Errorf("first call = %q, want %q", f.Calls[0], wantFirst)
				}
				if len(f.Calls) != 2 {
					t.Errorf("expected 2 commands run, got %v", f.Calls)
				}
			}
		})
	}
}

// TestRunCursorSetup_PromptNoTTY verifies the DecisionDefault + SetupPrompt path:
// when no tty is available, consent defaults to No and nothing runs. Test
// environments have no controlling /dev/tty, so this exercises the no-tty
// branch deterministically.
func TestRunCursorSetup_PromptNoTTY(t *testing.T) {
	if ui.HasTTY() {
		t.Skip("a tty is attached; cannot deterministically test the no-tty default")
	}
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	writeWorktreesJSON(t, root, `{"setup-worktree": ["echo hi"]}`)

	f := fakeWithDefault()
	r := New(f, config.Config{Cursor: config.Cursor{WorktreeSetup: config.SetupPrompt}})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionDefault); err != nil {
		t.Fatalf("RunCursorSetup: %v", err)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected nothing run without a tty, got %v", f.Calls)
	}
}

func TestRunCursorSetup_RunsInNewPathWithEnv(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	writeWorktreesJSON(t, root, `{"setup-worktree": ["echo $ROOT_WORKTREE_PATH"]}`)

	// Ensure the var is exported while the command runs.
	_ = os.Unsetenv(rootEnvVar)

	f := &exec.Fake{Default: &exec.FakeResult{}}
	r := New(f, config.Config{})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionYes); err != nil {
		t.Fatalf("RunCursorSetup: %v", err)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("expected 1 call, got %v", f.Calls)
	}
	// The env var is restored (unset) after the run.
	if _, ok := os.LookupEnv(rootEnvVar); ok {
		t.Errorf("%s should be unset after RunCursorSetup, but is set", rootEnvVar)
	}
}

func TestRunCursorSetup_ContinuesOnFailure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	writeWorktreesJSON(t, root, `{"setup-worktree": ["fail-cmd", "ok-cmd"]}`)

	f := &exec.Fake{
		Responses: map[string]exec.FakeResult{
			exec.Key("sh", "-c", "fail-cmd"): {Err: os.ErrPermission},
		},
		Default: &exec.FakeResult{},
	}
	r := New(f, config.Config{})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionYes); err != nil {
		t.Fatalf("RunCursorSetup should not return command failures, got %v", err)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("expected both commands attempted, got %v", f.Calls)
	}
}

func TestRunCursorSetup_EnvFallback(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	// No worktrees.json; provide a top-level .env.
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("KEY=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := fakeWithDefault()
	r := New(f, config.Config{})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionDefault); err != nil {
		t.Fatalf("RunCursorSetup: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(newPath, ".env"))
	if err != nil {
		t.Fatalf("expected .env copied: %v", err)
	}
	if string(got) != "KEY=value\n" {
		t.Errorf(".env content = %q", string(got))
	}
	if len(f.Calls) != 0 {
		t.Errorf("fallback should not run any commands, got %v", f.Calls)
	}
}

func TestRunCursorSetup_EnvFallback_NoClobber(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("ROOT=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newPath, ".env"), []byte("EXISTING=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := New(fakeWithDefault(), config.Config{})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionDefault); err != nil {
		t.Fatalf("RunCursorSetup: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(newPath, ".env"))
	if string(got) != "EXISTING=1\n" {
		t.Errorf("existing .env was clobbered: %q", string(got))
	}
}

func TestRunCursorSetup_EnvFallback_NoSource(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	newPath := t.TempDir()
	r := New(fakeWithDefault(), config.Config{})
	if err := r.RunCursorSetup(ctx, newPath, root, DecisionDefault); err != nil {
		t.Fatalf("RunCursorSetup with nothing to do should be a no-op, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, ".env")); !os.IsNotExist(err) {
		t.Errorf("did not expect a .env to be created")
	}
}

// TestRunHooks_TrustedNoConsent verifies user-config hooks run without any
// consent gate, even when cursor.worktree_setup is "never" (which only governs
// Cursor setup commands). This is the trusted-vs-untrusted distinction.
func TestRunHooks_TrustedNoConsent(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cwd := t.TempDir()

	cfg := config.Config{
		Cursor: config.Cursor{WorktreeSetup: config.SetupNever}, // must NOT block hooks
		Hooks: config.Hooks{
			PostCreate: []string{"echo created", "make deps"},
			PreRemove:  []string{"echo removing"},
		},
	}

	f := fakeWithDefault()
	r := New(f, cfg)

	if err := r.RunHooks(ctx, PostCreate, cwd, root); err != nil {
		t.Fatalf("RunHooks post_create: %v", err)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("expected post_create hooks to run untrusted, got %v", f.Calls)
	}
	if f.Calls[0] != exec.Key("sh", "-c", "echo created") {
		t.Errorf("first hook = %q", f.Calls[0])
	}

	f.Calls = nil
	if err := r.RunHooks(ctx, PreRemove, cwd, root); err != nil {
		t.Fatalf("RunHooks pre_remove: %v", err)
	}
	if len(f.Calls) != 1 || f.Calls[0] != exec.Key("sh", "-c", "echo removing") {
		t.Fatalf("expected pre_remove hook to run, got %v", f.Calls)
	}
}

func TestRunHooks_Empty(t *testing.T) {
	ctx := context.Background()
	f := fakeWithDefault()
	r := New(f, config.Config{})
	if err := r.RunHooks(ctx, PostCreate, t.TempDir(), t.TempDir()); err != nil {
		t.Fatalf("RunHooks with no hooks: %v", err)
	}
	if len(f.Calls) != 0 {
		t.Fatalf("expected no calls, got %v", f.Calls)
	}
}

func TestRunHooks_UnknownPhase(t *testing.T) {
	r := New(fakeWithDefault(), config.Config{})
	if err := r.RunHooks(context.Background(), Phase("bogus"), t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("expected error for unknown phase")
	}
}
