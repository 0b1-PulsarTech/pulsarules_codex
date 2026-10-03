package source

import (
	"path/filepath"
	"strings"
)

type listedProvider struct {
	base  Provider
	paths []string
	skips SkipSet
}

//nolint:ireturn // factory constructor returns interface
func NewListedProvider(base Provider, paths []string, skips SkipSet) Provider {
	return &listedProvider{base: base, paths: paths, skips: skips}
}

func (p *listedProvider) Read(path string) ([]byte, bool) {
	return p.base.Read(path)
}

func (p *listedProvider) Walk(fn func(path string, ext string) bool) {
	for _, path := range p.paths {
		if p.skips.SkipsPath(path) {
			continue
		}
		if !fn(path, strings.ToLower(filepath.Ext(path))) {
			return
		}
	}
}
