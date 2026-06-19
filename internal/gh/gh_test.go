package gh

import (
	"errors"
	"testing"

	"github.com/kyriacos/go-gwt/internal/exec"
)

// authOK is the canned successful `gh auth status` response used by most tests.
func authOK() exec.FakeResult {
	return exec.FakeResult{Stderr: "github.com\n  ✓ Logged in to github.com as octocat"}
}

func TestAvailable(t *testing.T) {
	t.Run("authenticated", func(t *testing.T) {
		f := &exec.Fake{Responses: map[string]exec.FakeResult{
			"gh auth status": authOK(),
		}}
		c := New(f)
		if !c.Available() {
			t.Fatal("expected Available() == true")
		}
		// Call again: result must be cached (no second exec).
		if !c.Available() {
			t.Fatal("expected cached Available() == true")
		}
		if got := countCalls(f.Calls, "gh auth status"); got != 1 {
			t.Fatalf("expected gh auth status to run once (cached), ran %d times", got)
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		f := &exec.Fake{Responses: map[string]exec.FakeResult{
			"gh auth status": {Stderr: "You are not logged into any GitHub hosts.", Err: errors.New("exit status 1")},
		}}
		c := New(f)
		if c.Available() {
			t.Fatal("expected Available() == false when not authenticated")
		}
	})

	t.Run("gh missing", func(t *testing.T) {
		// No canned response and no default -> Fake returns an error, mimicking
		// the binary not being found.
		f := &exec.Fake{}
		c := New(f)
		if c.Available() {
			t.Fatal("expected Available() == false when gh is missing")
		}
	})
}

func TestListPRs(t *testing.T) {
	const fixture = `[
  {"number":42,"title":"Add widgets","author":{"login":"octocat"},"headRefName":"feature/widgets","state":"OPEN","isDraft":false},
  {"number":7,"title":"WIP refactor","author":{"login":"hubot"},"headRefName":"refactor","state":"OPEN","isDraft":true}
]`
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"gh auth status": authOK(),
		"gh pr list --json number,title,author,headRefName,state,isDraft --limit 50": {Stdout: fixture},
	}}
	c := New(f)

	prs, err := c.ListPRs()
	if err != nil {
		t.Fatalf("ListPRs: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	want := PR{Number: 42, Title: "Add widgets", Author: "octocat", Branch: "feature/widgets", State: "OPEN", Draft: false}
	if prs[0] != want {
		t.Fatalf("prs[0] = %+v, want %+v", prs[0], want)
	}
	if !prs[1].Draft || prs[1].Author != "hubot" {
		t.Fatalf("prs[1] = %+v, want draft author hubot", prs[1])
	}
}

func TestListPRsUnavailable(t *testing.T) {
	f := &exec.Fake{} // gh auth status fails
	c := New(f)
	_, err := c.ListPRs()
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestCheckout(t *testing.T) {
	f := &exec.Fake{Responses: map[string]exec.FakeResult{
		"gh auth status": authOK(),
		"gh pr view 42 --json headRefName,headRefOid": {Stdout: `{"headRefName":"feature/widgets","headRefOid":"abc123"}`},
		"gh pr checkout 42":                           {Stdout: ""},
	}}
	c := New(f)

	branch, err := c.Checkout(42)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if branch != "feature/widgets" {
		t.Fatalf("branch = %q, want feature/widgets", branch)
	}
	if countCalls(f.Calls, "gh pr checkout 42") != 1 {
		t.Fatal("expected gh pr checkout 42 to run")
	}
}

func TestChecks(t *testing.T) {
	cases := []struct {
		name    string
		stdout  string
		err     error
		want    CIStatus
		wantErr bool
	}{
		{
			name:   "all passing",
			stdout: `[{"bucket":"pass","state":"SUCCESS"},{"bucket":"pass","state":"SUCCESS"}]`,
			want:   CIStatus{State: CIPassing, Passed: 2, Failed: 0, Total: 2},
		},
		{
			name:   "one failing",
			stdout: `[{"bucket":"pass","state":"SUCCESS"},{"bucket":"fail","state":"FAILURE"}]`,
			// gh exits non-zero when a check fails but still prints JSON.
			err:  errors.New("exit status 1"),
			want: CIStatus{State: CIFailing, Passed: 1, Failed: 1, Total: 2},
		},
		{
			name:   "pending",
			stdout: `[{"bucket":"pass","state":"SUCCESS"},{"bucket":"pending","state":"IN_PROGRESS"}]`,
			err:    errors.New("exit status 8"),
			want:   CIStatus{State: CIPending, Passed: 1, Failed: 0, Total: 2},
		},
		{
			name:   "skipped ignored",
			stdout: `[{"bucket":"pass","state":"SUCCESS"},{"bucket":"skipping","state":"SKIPPED"}]`,
			want:   CIStatus{State: CIPassing, Passed: 1, Failed: 0, Total: 1},
		},
		{
			name:   "state fallback when bucket empty",
			stdout: `[{"bucket":"","state":"SUCCESS"},{"bucket":"","state":"FAILURE"}]`,
			err:    errors.New("exit status 1"),
			want:   CIStatus{State: CIFailing, Passed: 1, Failed: 1, Total: 2},
		},
		{
			name:   "no checks",
			stdout: `[]`,
			want:   CIStatus{State: CIUnknown, Passed: 0, Failed: 0, Total: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &exec.Fake{Responses: map[string]exec.FakeResult{
				"gh auth status": authOK(),
				"gh pr checks somebranch --json bucket,state": {Stdout: tc.stdout, Err: tc.err},
			}}
			c := New(f)
			got, err := c.Checks("somebranch")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Checks: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Checks = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestChecksUnavailable(t *testing.T) {
	f := &exec.Fake{}
	c := New(f)
	_, err := c.Checks("branch")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func countCalls(calls []string, key string) int {
	n := 0
	for _, c := range calls {
		if c == key {
			n++
		}
	}
	return n
}
