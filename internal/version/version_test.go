package version

import (
	"strings"
	"testing"
)

func TestShortCommit(t *testing.T) {
	orig := Commit
	t.Cleanup(func() { Commit = orig })

	Commit = "abcdef1234567890"
	if got := ShortCommit(); got != "abcdef1" {
		t.Fatalf("ShortCommit() = %q, want abcdef1", got)
	}

	Commit = "none"
	if got := ShortCommit(); got != "dev" && !strings.HasPrefix(got, "dev") {
		// vcsRevision may populate from build info in module builds.
		if got == "" {
			t.Fatalf("ShortCommit() with none = empty")
		}
	}
}

func TestStringFormat(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origV, origC, origD
	})
	Set("1.2.3", "deadbeef", "2026-07-13T10:00:00Z")
	got := String()
	for _, want := range []string{"gwt 1.2.3", "deadbee", "2026-07-13T10:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("String() = %q, want substring %q", got, want)
		}
	}
}

func TestShortFormat(t *testing.T) {
	origV, origC := Version, Commit
	t.Cleanup(func() {
		Version, Commit = origV, origC
	})
	Set("1.2.3", "deadbeef", "")
	if got := Short(); got != "gwt 1.2.3 · deadbee" {
		t.Fatalf("Short() = %q", got)
	}
}
