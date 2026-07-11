package core

import (
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

// fileSourceProvider is the default SourceProvider. It reads files from disk
// via os.ReadFile, caching contents in a map so repeated reads are free.
type fileSourceProvider struct {
	projectDir string
	mu         sync.RWMutex
	cache      map[string][]byte
}

// NewSourceProvider creates a SourceProvider rooted at projectDir. Files read
// via Read are cached for the lifetime of the provider.
//
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

// walkSkipDirs is the set of directory basenames Walk never descends into:
// git internals, installed skill mirrors, and generated/vendored trees that
// would otherwise drown a whole-tree scan in duplicated content (there are
// ~37 generated skill copies alone).
var walkSkipDirs = map[string]bool{
	".git":      true,
	".claude":   true,
	".opencode": true,
	"generated": true,
	"build":     true,
	"vendor":    true,
}

func (p *fileSourceProvider) Walk(fn func(path string, ext string) bool) {
	_ = filepath.WalkDir(p.projectDir, func(full string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if full != p.projectDir && walkSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(p.projectDir, full)
		if relErr != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(full))
		if !fn(rel, ext) {
			return filepath.SkipAll
		}
		return nil
	})
}
