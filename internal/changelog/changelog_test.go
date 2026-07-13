package changelog

import (
	"os"
	"strings"
	"testing"
)

func TestMarkdownEmbedded(t *testing.T) {
	t.Parallel()
	if strings.TrimSpace(Markdown) == "" {
		t.Fatal("expected embedded changelog markdown")
	}
	if !strings.Contains(Markdown, "# Changelog") {
		t.Fatalf("unexpected changelog header: %q", Markdown[:min(80, len(Markdown))])
	}
}

func TestMatchesRepositoryChangelog(t *testing.T) {
	t.Parallel()
	root, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(root) != Markdown {
		t.Fatal("internal/changelog/CHANGELOG.md is stale; run: cp CHANGELOG.md internal/changelog/CHANGELOG.md")
	}
}
