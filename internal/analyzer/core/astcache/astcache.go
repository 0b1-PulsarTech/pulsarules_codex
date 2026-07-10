package astcache

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
)

type Cache struct {
	fset *token.FileSet
	mu   sync.RWMutex
	file map[string]*ast.File
}

func New() *Cache {
	return &Cache{
		fset: token.NewFileSet(),
		file: make(map[string]*ast.File),
	}
}

func (c *Cache) FileSet() *token.FileSet { return c.fset }

func (c *Cache) Parse(path string, src []byte) (*ast.File, error) {
	c.mu.RLock()
	if f, ok := c.file[path]; ok {
		c.mu.RUnlock()
		return f, nil
	}
	c.mu.RUnlock()

	f, err := parser.ParseFile(c.fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.file[path] = f
	return f, nil
}

func (c *Cache) Get(path string) *ast.File {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.file[path]
}
