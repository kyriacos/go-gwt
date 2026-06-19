package exec

import (
	"context"
	"fmt"
	"strings"
)

// FakeResult is a canned response for a single command invocation.
type FakeResult struct {
	Stdout string
	Stderr string
	Err    error
}

// Fake is a Runner that returns canned results keyed by the full command line
// ("name arg1 arg2 ..."). It records every call for assertions. Unmatched
// commands return an error unless Default is set. Construct directly:
//
//	f := &exec.Fake{Responses: map[string]exec.FakeResult{
//	    "git worktree list --porcelain": {Stdout: fixture},
//	}}
type Fake struct {
	Responses map[string]FakeResult
	Default   *FakeResult
	Calls     []string
}

// Key builds the map key for a command and its args.
func Key(name string, args ...string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

// Run implements Runner against the canned Responses.
func (f *Fake) Run(_ context.Context, _ , name string, args ...string) ([]byte, []byte, error) {
	key := Key(name, args...)
	f.Calls = append(f.Calls, key)
	if r, ok := f.Responses[key]; ok {
		return []byte(r.Stdout), []byte(r.Stderr), r.Err
	}
	if f.Default != nil {
		return []byte(f.Default.Stdout), []byte(f.Default.Stderr), f.Default.Err
	}
	return nil, nil, fmt.Errorf("exec.Fake: no canned response for %q", key)
}
