package core

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestFile writes content at dir/relPath, creating parent directories as
// needed.
func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", full, err)
	}
}

func TestFileSourceProvider_WalkSkipsExcludedDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", "package main\n")
	writeTestFile(t, dir, ".git/HEAD", "ref: refs/heads/main\n")
	writeTestFile(t, dir, ".claude/skills/foo/SKILL.md", "# foo\n")
	writeTestFile(t, dir, ".opencode/plugins/x.js", "// x\n")
	writeTestFile(t, dir, "generated/claude/skills/foo/SKILL.md", "# foo\n")
	writeTestFile(t, dir, "build/bin/tool", "binary\n")
	writeTestFile(t, dir, "vendor/pkg/pkg.go", "package pkg\n")
	writeTestFile(t, dir, "testdata/golden/case/fixture.go", "package fixture\n")
	writeTestFile(t, dir, "internal/real.go", "package internal\n")

	provider := NewSourceProvider(dir)
	var visited []string
	provider.Walk(func(path, _ string) bool {
		visited = append(visited, path)
		return true
	})

	want := map[string]bool{
		filepath.Join("main.go"):          true,
		filepath.Join("internal/real.go"): true,
	}
	if len(visited) != len(want) {
		t.Fatalf("visited = %v, want exactly %v", visited, want)
	}
	for _, path := range visited {
		if !want[path] {
			t.Errorf("Walk visited excluded path %q", path)
		}
	}
}

func TestFileSourceProvider_WalkStopsOnFalse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "a.go", "package a\n")
	writeTestFile(t, dir, "b.go", "package b\n")
	writeTestFile(t, dir, "c.go", "package c\n")

	provider := NewSourceProvider(dir)
	var visited []string
	provider.Walk(func(path, _ string) bool {
		visited = append(visited, path)
		return false
	})

	if len(visited) != 1 {
		t.Fatalf("visited = %v, want exactly one entry (walk should stop on false)", visited)
	}
}

func TestFileSourceProvider_WalkLowercasesExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "README.MD", "# hi\n")

	provider := NewSourceProvider(dir)
	var gotExt string
	provider.Walk(func(_, ext string) bool {
		gotExt = ext
		return true
	})

	if gotExt != ".md" {
		t.Fatalf("ext = %q, want %q", gotExt, ".md")
	}
}
