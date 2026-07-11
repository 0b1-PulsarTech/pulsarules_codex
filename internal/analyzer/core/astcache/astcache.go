package astcache

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
)

// Cache holds parsed ASTs keyed by file path. It is safe for concurrent use.
type Cache struct {
	fset *token.FileSet
	mu   sync.RWMutex
	file map[string]*ast.File
	err  map[string]error
}

// New creates an empty cache with its own FileSet.
func New() *Cache {
	return &Cache{
		fset: token.NewFileSet(),
		file: make(map[string]*ast.File),
		err:  make(map[string]error),
	}
}

// FileSet returns the token.FileSet used by this cache. All parsed files
// share this FileSet so position information is consistent across analyzers.
func (c *Cache) FileSet() *token.FileSet { return c.fset }

// Parse parses a Go source file and caches the result. Subsequent calls for
// the same path return the cached parse. A parse error is cached too, so a
// broken file is only parsed once per invocation.
func (c *Cache) Parse(path string, src []byte) (*ast.File, error) {
	c.mu.RLock()
	if f, ok := c.file[path]; ok {
		c.mu.RUnlock()
		return f, c.err[path]
	}
	c.mu.RUnlock()

	f, err := parser.ParseFile(c.fset, path, src, parser.ParseComments)
	if err != nil {
		c.mu.Lock()
		c.err[path] = err
		c.mu.Unlock()
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.file[path] = f
	return f, nil
}

// Get returns a previously parsed file, or nil if not yet parsed.
func (c *Cache) Get(path string) *ast.File {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.file[path]
}
