package arch

import (
	"testing"
)

func TestLoadProjectIndex(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "foo.go", "package foo\n")
	writeFile(t, projectDir, "bar/baz.go", "package bar\nimport \"example.com/mod/foo\"\n")

	idx := loadProjectIndex(projectDir, "example.com/mod")
	if idx == nil {
		t.Fatal("expected a non-nil project index")
	}
	if len(idx.pkgMap) == 0 {
		t.Fatal("expected at least one discovered package")
	}
	if idx.graph == nil {
		t.Fatal("expected a built dependency graph")
	}
}

func TestLoadProjectIndex_RebuildsEveryCall(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "foo.go", "package foo\n")

	firstIndex := loadProjectIndex(projectDir, "example.com/mod")
	secondIndex := loadProjectIndex(projectDir, "example.com/mod")

	// why: there is no cache to reuse across calls (see the file-level why
	// comment on loadProjectIndex), so each call must return its own index.
	if firstIndex == secondIndex {
		t.Error("expected a freshly built index on every call, got the same pointer")
	}
}
