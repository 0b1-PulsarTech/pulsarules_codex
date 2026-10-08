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
	err  map[string]error
}

func New() *Cache {
	return &Cache{
		fset: token.NewFileSet(),
		file: make(map[string]*ast.File),
		err:  make(map[string]error),
	}
}

func (cache *Cache) FileSet() *token.FileSet { return cache.fset }

func (cache *Cache) Parse(path string, src []byte) (*ast.File, error) {
	cache.mu.RLock()
	if file, ok := cache.file[path]; ok {
		cache.mu.RUnlock()
		return file, nil
	}
	if err, ok := cache.err[path]; ok {
		cache.mu.RUnlock()
		return nil, err
	}
	cache.mu.RUnlock()

	file, err := parser.ParseFile(cache.fset, path, src, parser.ParseComments)
	if err != nil {
		err = fmt.Errorf("parse %s: %w", path, err)
		cache.mu.Lock()
		cache.err[path] = err
		cache.mu.Unlock()
		return nil, err
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.file[path] = file
	return file, nil
}

func (cache *Cache) Get(path string) *ast.File {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.file[path]
}

func (cache *Cache) Err(path string) error {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.err[path]
}
