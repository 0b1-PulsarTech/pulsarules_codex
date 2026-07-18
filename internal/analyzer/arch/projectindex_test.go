package arch

import (
	"testing"
)

func TestLoadProjectIndex(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "foo.go", "package foo\n")
	writeFile(t, tmp, "bar/baz.go", "package bar\nimport \"example.com/mod/foo\"\n")

	idx := loadProjectIndex(tmp, "example.com/mod")
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

	tmp := t.TempDir()
	writeFile(t, tmp, "foo.go", "package foo\n")

	idx1 := loadProjectIndex(tmp, "example.com/mod")
	idx2 := loadProjectIndex(tmp, "example.com/mod")

	if idx1 == idx2 {
		t.Error("expected a freshly built index on every call, got the same pointer")
	}
}
