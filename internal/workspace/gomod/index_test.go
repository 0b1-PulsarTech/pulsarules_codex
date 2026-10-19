package gomod

import (
	"maps"
	"slices"
	"sync"
	"testing"
)

type countingTree struct {
	files map[string]string
	reads map[string]int
}

func (tree *countingTree) Read(file string) ([]byte, bool) {
	tree.reads[file]++
	src, ok := tree.files[file]
	return []byte(src), ok
}

func (tree *countingTree) Walk(visit func(string, string) bool) {
	for _, file := range slices.Sorted(func(yield func(string) bool) {
		for file := range tree.files {
			if !yield(file) {
				return
			}
		}
	}) {
		if !visit(file, "") {
			return
		}
	}
}

func TestNewIndex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		files     map[string]string
		wantDirs  []string
		wantFiles map[string]int
	}{
		{
			name: "package dirs only",
			files: map[string]string{
				"main.go":              "package main\n",
				"a/a.go":               "package a\n",
				"a/a_test.go":          "package a\n",
				"a/note.md":            "# not go\n",
				"vendor/v/v.go":        "package v\n",
				"testdata/t.go":        "package t\n",
				".hidden/h.go":         "package h\n",
				"_skip/s.go":           "package s\n",
				"a/testdata/deep/d.go": "package d\n",
			},
			wantDirs:  []string{".", "a", "vendor/v"},
			wantFiles: map[string]int{".": 1, "a": 2, "vendor/v": 1},
		},
		{
			name: "files Go never compiles are dropped",
			files: map[string]string{
				"a/a.go":      "package a\n",
				"a/_x.go":     "package a\n",
				"a/.y.go":     "package a\n",
				"a/gen.go":    "//go:build ignore\n\npackage main\n",
				"a/legacy.go": "// +build ignore\n\npackage main\n",
				"h/_only.go":  "package h\n",
			},
			wantDirs:  []string{"a"},
			wantFiles: map[string]int{"a": 1},
		},
		{
			name:      "syntax error keeps the file with its error",
			files:     map[string]string{"b/b.go": "package b\nfunc (\n"},
			wantDirs:  []string{"b"},
			wantFiles: map[string]int{"b": 1},
		},
		{name: "empty tree", files: map[string]string{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tree := &countingTree{files: testCase.files, reads: make(map[string]int)}
			index := NewIndex(tree)
			if got := index.Dirs(); !slices.Equal(got, testCase.wantDirs) {
				t.Fatalf("Dirs() = %v, want %v", got, testCase.wantDirs)
			}
			for dir, want := range testCase.wantFiles {
				if got := len(index.Files(dir)); got != want {
					t.Errorf("Files(%q) = %d files, want %d", dir, got, want)
				}
			}
			for file, count := range tree.reads {
				if count != 1 {
					t.Errorf("%s read %d times, want once", file, count)
				}
			}
		})
	}
}

func TestNewIndexFileErrors(t *testing.T) {
	t.Parallel()

	tree := &countingTree{
		files: map[string]string{"b/b.go": "package b\nfunc (\n", "c/c.go": "package c\n"},
		reads: make(map[string]int),
	}
	index := NewIndex(tree)
	if broken := index.Files("b")[0]; broken.Err == nil || broken.Syntax == nil {
		t.Errorf("broken file = %+v, want a partial tree and an error", broken)
	}
	if intact := index.Files("c")[0]; intact.Err != nil || intact.Syntax.Name.Name != "c" {
		t.Errorf("intact file = %+v, want package c and no error", intact)
	}
}

func TestIndexFile(t *testing.T) {
	t.Parallel()

	tree := &countingTree{
		files: map[string]string{"a/a.go": "package a\n", "a/b.go": "package a\n"},
		reads: make(map[string]int),
	}
	index := NewIndex(tree)
	testCases := []struct {
		name   string
		file   string
		wantOK bool
	}{
		{name: "indexed", file: "a/b.go", wantOK: true},
		{name: "unknown file in an indexed dir", file: "a/c.go"},
		{name: "unknown dir", file: "z/z.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := index.File(testCase.file)
			if ok != testCase.wantOK || (ok && got.Path != testCase.file) {
				t.Errorf("File(%q) = %+v, %v, want ok=%v", testCase.file, got, ok, testCase.wantOK)
			}
		})
	}
}

func TestIndexFilesConcurrent(t *testing.T) {
	t.Parallel()

	const callers = 16
	tree := &countingTree{
		files: map[string]string{"a/a.go": "package a\n", "a/b.go": "package a\n"},
		reads: make(map[string]int),
	}
	index := NewIndex(tree)
	start := make(chan struct{})
	var group sync.WaitGroup
	for range callers {
		group.Go(func() {
			<-start
			if files := index.Files("a"); len(files) != 2 {
				t.Errorf("Files(a) = %d files, want 2", len(files))
			}
		})
	}
	close(start)
	group.Wait()
	if tree.reads["a/a.go"] != 1 {
		t.Errorf("a/a.go read %d times, want once", tree.reads["a/a.go"])
	}
}

func TestNewIndexNilTree(t *testing.T) {
	t.Parallel()

	if dirs := NewIndex(nil).Dirs(); len(dirs) != 0 {
		t.Errorf("Dirs() = %v, want none", dirs)
	}
}

func TestIndexParsesLazily(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		ask       func(index *Index)
		wantReads map[string]int
	}{
		{
			name:      "nothing asked reads nothing",
			ask:       func(*Index) {},
			wantReads: map[string]int{},
		},
		{
			name:      "Files reads only its directory, once",
			ask:       func(index *Index) { index.Files("a"); index.Files("a") },
			wantReads: map[string]int{"a/a.go": 1, "a/b.go": 1},
		},
		{
			name:      "File reads only its directory",
			ask:       func(index *Index) { index.File("b/b.go"); index.File("b/b.go") },
			wantReads: map[string]int{"b/b.go": 1},
		},
		{
			name:      "Dirs reads nothing",
			ask:       func(index *Index) { index.Dirs() },
			wantReads: map[string]int{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tree := &countingTree{
				files: map[string]string{
					"a/a.go": "package a\n", "a/b.go": "package a\n", "b/b.go": "package b\n",
				},
				reads: make(map[string]int),
			}
			testCase.ask(NewIndex(tree))
			if !maps.Equal(tree.reads, testCase.wantReads) {
				t.Errorf("reads = %v, want %v", tree.reads, testCase.wantReads)
			}
		})
	}
}
