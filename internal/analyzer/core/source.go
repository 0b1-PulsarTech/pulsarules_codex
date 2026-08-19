package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SourceProvider is the consumer-declared port for reading file contents and
// walking the project tree. Analyzers call Read to get file bytes and Walk to
// enumerate files; they never touch os.ReadFile or filepath.WalkDir directly.
type SourceProvider interface {
	// Read returns the contents of the file at the given repo-relative path,
	// or false when the file does not exist or cannot be read.
	Read(path string) ([]byte, bool)
	// Walk enumerates non-test source files in the project, calling fn for
	// each. The path is repo-relative. Returning false from fn stops the walk.
	Walk(fn func(path string, ext string) bool)
}

// fsSourceProvider is the default SourceProvider: any fs.FS with contents
// cached in a map so repeated reads are free.
type fsSourceProvider struct {
	fsys       fs.FS
	projectDir string
	mu         sync.RWMutex
	cache      map[string][]byte
}

// NewSourceProvider creates a SourceProvider rooted at projectDir on disk.
//
//nolint:ireturn // factory constructor returns interface
func NewSourceProvider(projectDir string) SourceProvider {
	return NewFSSourceProvider(os.DirFS(projectDir), projectDir)
}

// NewFSSourceProvider creates a SourceProvider over fsys - os.DirFS in
// production, fstest.MapFS in tests. projectDir names the on-disk root fsys
// was carved from ("" for a virtual fs), used only to relativize abs paths.
//
//nolint:ireturn // factory constructor returns interface
func NewFSSourceProvider(fsys fs.FS, projectDir string) SourceProvider {
	return &fsSourceProvider{
		fsys:       fsys,
		projectDir: projectDir,
		cache:      make(map[string][]byte),
	}
}

func (p *fsSourceProvider) Read(path string) ([]byte, bool) {
	p.mu.RLock()
	content, ok := p.cache[path]
	p.mu.RUnlock()
	if ok {
		return content, true
	}
	rel := path
	if filepath.IsAbs(rel) {
		if p.projectDir == "" {
			return nil, false
		}
		var relErr error
		if rel, relErr = filepath.Rel(p.projectDir, rel); relErr != nil {
			return nil, false
		}
	}
	content, err := fs.ReadFile(p.fsys, filepath.ToSlash(rel))
	if err != nil {
		return nil, false
	}
	p.mu.Lock()
	p.cache[path] = content
	p.mu.Unlock()
	return content, true
}

// walkSkipDirs is the set of directory basenames Walk never descends into:
// git internals, installed skill mirrors, and generated/vendored trees that
// would drown a whole-tree scan (~37 generated skill copies alone).
// "testdata" follows the go tool's own "./..." convention: fixtures under
// it are deliberately non-conforming and must never surface as findings.
var walkSkipDirs = map[string]bool{
	".git":      true,
	".claude":   true,
	".opencode": true,
	"generated": true,
	"build":     true,
	"vendor":    true,
	"testdata":  true,
}

func (p *fsSourceProvider) Walk(fn func(path string, ext string) bool) {
	_ = fs.WalkDir(p.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // WalkDir callback: skip the bad entry, keep walking.
		}
		if d.IsDir() {
			if path != "." && walkSkipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !fn(path, ext) {
			return fs.SkipAll
		}
		return nil
	})
}
