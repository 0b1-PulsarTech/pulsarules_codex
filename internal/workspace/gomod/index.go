package gomod

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"sync"
)

type Tree interface {
	Read(path string) ([]byte, bool)
	Walk(fn func(path string, ext string) bool)
}

type File struct {
	Path   string
	Syntax *ast.File
	Err    error
}

type Index struct {
	fset *token.FileSet
	tree Tree

	mutex  sync.Mutex
	paths  map[string][]string
	parsed map[string][]File
}

func NewIndex(tree Tree) *Index {
	index := &Index{
		fset:   token.NewFileSet(),
		tree:   tree,
		paths:  make(map[string][]string),
		parsed: make(map[string][]File),
	}
	if tree == nil {
		return index
	}
	tree.Walk(func(file string, _ string) bool {
		file = filepath.ToSlash(file)
		dir := path.Dir(file)
		// Go never compiles a file whose name starts with "_" or ".".
		if path.Ext(file) != ".go" || IsHiddenName(path.Base(file)) || !IsPackageDir(dir) {
			return true
		}
		index.paths[dir] = append(index.paths[dir], file)
		return true
	})
	return index
}

func (index *Index) FileSet() *token.FileSet { return index.fset }

func (index *Index) Files(dir string) []File {
	// Parsed on the first request: a caller never pays for a directory it does not ask about.
	index.mutex.Lock()
	defer index.mutex.Unlock()
	if files, done := index.parsed[dir]; done {
		return files
	}
	var files []File
	for _, file := range index.paths[dir] {
		src, ok := index.tree.Read(file)
		if !ok {
			continue
		}
		parsed, err := parser.ParseFile(
			index.fset, file, src, parser.ParseComments|parser.SkipObjectResolution,
		)
		if IsBuildIgnored(parsed) {
			continue
		}
		files = append(files, File{Path: file, Syntax: parsed, Err: err})
	}
	index.parsed[dir] = files
	return files
}

func (index *Index) File(file string) (File, bool) {
	files := index.Files(path.Dir(file))
	at := slices.IndexFunc(files, func(entry File) bool { return entry.Path == file })
	if at < 0 {
		return File{}, false
	}
	return files[at], true
}

func (index *Index) Dirs() []string {
	dirs := make([]string, 0, len(index.paths))
	for dir := range index.paths {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	return dirs
}
