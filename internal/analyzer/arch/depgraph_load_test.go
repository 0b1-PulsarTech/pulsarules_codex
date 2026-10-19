package arch

import "testing"

func TestLoadDepGraph(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "foo.go", "package foo\n")
	writeFile(t, projectDir, "bar/baz.go", "package bar\nimport \"example.com/mod/foo\"\n")

	graph := loadDepGraph(newSources(projectDir), "example.com/mod")
	if len(graph) == 0 {
		t.Fatal("expected at least one discovered package")
	}
}
