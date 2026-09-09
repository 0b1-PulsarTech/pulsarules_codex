package pass

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SourceProvider interface {
	Read(path string) ([]byte, bool)
	Walk(fn func(path string, ext string) bool)
}

type fsSourceProvider struct {
	fsys       fs.FS
	projectDir string
	mu         sync.RWMutex
	cache      map[string][]byte
}

//nolint:ireturn // factory constructor returns interface
func NewSourceProvider(projectDir string) SourceProvider {
	return NewFSSourceProvider(os.DirFS(projectDir), projectDir)
}

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

func (p *fsSourceProvider) Walk(fn func(path string, ext string) bool) {
	skipDirs := map[string]bool{
		".git":      true,
		".claude":   true,
		".opencode": true,
		"generated": true,
		"build":     true,
		"vendor":    true,
		"testdata":  true,
	}
	_ = fs.WalkDir(p.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // WalkDir callback: skip the bad entry, keep walking.
		}
		if d.IsDir() {
			if path != "." && skipDirs[d.Name()] {
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
