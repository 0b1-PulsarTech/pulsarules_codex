package core

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SourceProvider interface {
	Read(path string) ([]byte, bool)
	Walk(fn func(path string, ext string) bool)
}

type fileSourceProvider struct {
	projectDir string
	mu         sync.RWMutex
	cache      map[string][]byte
}

//nolint:ireturn // factory constructor returns interface
func NewSourceProvider(projectDir string) SourceProvider {
	return &fileSourceProvider{
		projectDir: projectDir,
		cache:      make(map[string][]byte),
	}
}

func (p *fileSourceProvider) Read(path string) ([]byte, bool) {
	p.mu.RLock()
	content, ok := p.cache[path]
	p.mu.RUnlock()
	if ok {
		return content, true
	}
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(p.projectDir, path)
	}
	content, err := os.ReadFile(abs) //nolint:gosec // project file path.
	if err != nil {
		return nil, false
	}
	p.mu.Lock()
	p.cache[path] = content
	p.mu.Unlock()
	return content, true
}

func (p *fileSourceProvider) Walk(fn func(path string, ext string) bool) {
	skipDirs := map[string]bool{
		".git":      true,
		".claude":   true,
		"generated": true,
		"build":     true,
		"vendor":    true,
	}
	_ = filepath.WalkDir(p.projectDir, func(full string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // WalkDir callback: skip the bad entry, keep walking.
		}
		if d.IsDir() {
			if full != p.projectDir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(p.projectDir, full)
		if relErr != nil {
			return nil //nolint:nilerr // unrelatable entry: skip it, keep walking.
		}
		ext := strings.ToLower(filepath.Ext(full))
		if !fn(rel, ext) {
			return filepath.SkipAll
		}
		return nil
	})
}
