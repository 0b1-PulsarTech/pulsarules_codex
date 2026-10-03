package source

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

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

func TestFSProvider_WalkSkipsExcludedDirs(t *testing.T) {
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

	provider := NewProvider(dir, NewSkipSet(DefaultSkipDirs()))
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

func TestFSProvider_WalkStopsOnFalse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "a.go", "package a\n")
	writeTestFile(t, dir, "b.go", "package b\n")
	writeTestFile(t, dir, "c.go", "package c\n")

	provider := NewProvider(dir, NewSkipSet(DefaultSkipDirs()))
	var visited []string
	provider.Walk(func(path, _ string) bool {
		visited = append(visited, path)
		return false
	})

	if len(visited) != 1 {
		t.Fatalf("visited = %v, want exactly one entry (walk should stop on false)", visited)
	}
}

func TestFSProvider_WalkLowercasesExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "README.MD", "# hi\n")

	provider := NewProvider(dir, NewSkipSet(DefaultSkipDirs()))
	var gotExt string
	provider.Walk(func(_, ext string) bool {
		gotExt = ext
		return true
	})

	if gotExt != ".md" {
		t.Fatalf("ext = %q, want %q", gotExt, ".md")
	}
}

func TestFSProvider_InMemoryFS(t *testing.T) {
	t.Parallel()

	provider := NewFSProvider(fstest.MapFS{
		"a.go":           &fstest.MapFile{Data: []byte("package a\n")},
		"generated/b.go": &fstest.MapFile{Data: []byte("package b\n")},
	}, "", NewSkipSet(DefaultSkipDirs()))

	content, ok := provider.Read("a.go")
	if !ok || string(content) != "package a\n" {
		t.Fatalf("Read(a.go) = %q, %v, want the file's bytes", content, ok)
	}
	if _, ok = provider.Read("/abs/outside.go"); ok {
		t.Error("Read of an absolute path on a virtual fs must report not found")
	}

	var walked []string
	provider.Walk(func(path, _ string) bool {
		walked = append(walked, path)
		return true
	})
	if len(walked) != 1 || walked[0] != "a.go" {
		t.Errorf("Walk = %v, want only a.go (generated/ is skipped)", walked)
	}
}

func TestFSProvider_ReadAbsolutePaths(t *testing.T) {
	t.Parallel()

	projectDir, outsideDir := t.TempDir(), t.TempDir()
	writeTestFile(t, projectDir, "pkg/a.go", "package a\n")
	writeTestFile(t, outsideDir, "secret.go", "package secret\n")
	provider := NewProvider(projectDir, NewSkipSet(DefaultSkipDirs()))
	insideFile := filepath.Join(projectDir, "pkg", "a.go")
	outsideFile := filepath.Join(outsideDir, "secret.go")
	missingFile := filepath.Join(projectDir, "none.go")

	testCases := []struct {
		name   string
		path   string
		want   string
		wantOK bool
	}{
		{"an absolute path inside the project", insideFile, "package a\n", true},
		{"a relative path inside the project", "pkg/a.go", "package a\n", true},
		{"an absolute path outside the project", outsideFile, "", false},
		{"a missing absolute path inside the project", missingFile, "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := provider.Read(testCase.path)
			if ok != testCase.wantOK || string(got) != testCase.want {
				t.Errorf(
					"Read(%q) = %q, %v, want %q, %v",
					testCase.path, got, ok, testCase.want, testCase.wantOK,
				)
			}
		})
	}
}
