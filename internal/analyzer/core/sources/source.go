package sources

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Provider interface {
	Read(path string) ([]byte, bool)
	Walk(fn func(path string, ext string) bool)
}

type fsProvider struct {
	fsys       fs.FS
	projectDir string
	skips      SkipSet
	mu         sync.RWMutex
	cache      map[string][]byte
}

//nolint:ireturn // factory constructor returns interface
func NewProvider(projectDir string, skips SkipSet) Provider {
	return NewFSProvider(os.DirFS(projectDir), projectDir, skips)
}

//nolint:ireturn // factory constructor returns interface
func NewFSProvider(fsys fs.FS, projectDir string, skips SkipSet) Provider {
	return &fsProvider{
		fsys:       fsys,
		projectDir: projectDir,
		skips:      skips,
		cache:      make(map[string][]byte),
	}
}

func (provider *fsProvider) Read(path string) ([]byte, bool) {
	provider.mu.RLock()
	content, ok := provider.cache[path]
	provider.mu.RUnlock()
	if ok {
		return content, true
	}
	rel := path
	if filepath.IsAbs(rel) {
		if provider.projectDir == "" {
			return nil, false
		}
		var relErr error
		if rel, relErr = filepath.Rel(provider.projectDir, rel); relErr != nil {
			return nil, false
		}
	}
	content, err := fs.ReadFile(provider.fsys, filepath.ToSlash(rel))
	if err != nil {
		return nil, false
	}
	provider.mu.Lock()
	provider.cache[path] = content
	provider.mu.Unlock()
	return content, true
}

func (provider *fsProvider) Walk(fn func(path string, ext string) bool) {
	_ = fs.WalkDir(provider.fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // WalkDir callback: skip the bad entry, keep walking.
		}
		if entry.IsDir() {
			if path != "." && provider.skips.SkipsDir(entry.Name()) {
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
