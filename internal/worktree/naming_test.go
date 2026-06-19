package worktree

import "testing"

func TestApplyTemplate(t *testing.T) {
	t.Parallel()
	const main = "/home/u/projects/myrepo"
	tests := []struct {
		name     string
		template string
		branch   string
		want     string
	}{
		{"default empty template", "", "feature", "myrepo-feature"},
		{"explicit default", "{repo}-{branch}", "feature", "myrepo-feature"},
		{"slash in branch", "{repo}-{branch}", "feature/foo", "myrepo-feature-foo"},
		{"branch_slug token", "{branch_slug}", "Feature/Foo Bar!", "feature-foo-bar"},
		{"repo only", "{repo}", "x", "myrepo"},
		{"custom join", "wt_{branch}", "topic/x", "wt_topic-x"},
		{"slug collapses repeats", "{branch_slug}", "a///b___c", "a-b-c"},
		{"whitespace template falls back", "   ", "bug", "myrepo-bug"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := applyTemplate(tt.template, main, tt.branch); got != tt.want {
				t.Fatalf("applyTemplate(%q, %q, %q) = %q, want %q", tt.template, main, tt.branch, got, tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"Simple":        "simple",
		"feature/foo":   "feature-foo",
		"FOO_BAR":       "foo-bar",
		"a  b":          "a-b",
		"--leading--":   "leading",
		"123-abc":       "123-abc",
		"!!!":           "",
		"café":          "caf", // non-ascii dropped
		"x/y/z":         "x-y-z",
		"Mixed.Case-99": "mixed-case-99",
	}
	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got := slugify(in); got != want {
				t.Fatalf("slugify(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestBranchDir(t *testing.T) {
	t.Parallel()
	if got := branchDir("a/b/c"); got != "a-b-c" {
		t.Fatalf("branchDir = %q", got)
	}
	if got := branchDir("plain"); got != "plain" {
		t.Fatalf("branchDir = %q", got)
	}
}
