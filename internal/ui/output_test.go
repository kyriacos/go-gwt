package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPath_WritesFileWhenGWT_PATH_OUTSet(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "path")
	t.Setenv(pathOutEnv, out)

	Path("/worktrees/feature")

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "/worktrees/feature\n" {
		t.Fatalf("file = %q", string(data))
	}
}

func TestPopulate_WritesFileWhenGWT_PATH_OUTSet(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "path")
	t.Setenv(pathOutEnv, out)

	Populate("gwt from my-branch")

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "GWT_POPULATE:gwt from my-branch\n" {
		t.Fatalf("file = %q", string(data))
	}
}
