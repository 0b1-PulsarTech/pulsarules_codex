package source

import (
	"path/filepath"
	"strings"
)

type listedSourceProvider struct {
	base  SourceProvider
	paths []string
	skips SkipSet
}

//nolint:ireturn // factory constructor returns interface
func NewListedSourceProvider(base SourceProvider, paths []string, skips SkipSet) SourceProvider {
	return &listedSourceProvider{base: base, paths: paths, skips: skips}
}

func (p *listedSourceProvider) Read(path string) ([]byte, bool) {
	return p.base.Read(path)
}

func (p *listedSourceProvider) Walk(fn func(path string, ext string) bool) {
	for _, path := range p.paths {
		if p.skips.SkipsPath(path) {
			continue
		}
		if !fn(path, strings.ToLower(filepath.Ext(path))) {
			return
		}
	}
}
