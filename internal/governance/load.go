package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func populateASTCache(
	changedFiles []pass.FileChange,
	sources pass.SourceProvider,
) *astcache.Cache {
	if len(changedFiles) == 0 {
		return nil
	}
	cache := astcache.New()
	for _, fc := range changedFiles {
		if fc.Extension == ".go" && !fc.IsTest {
			src, ok := sources.Read(fc.Path)
			if !ok {
				continue
			}
			if _, err := cache.Parse(fc.Path, src); err != nil {
				continue
			}
		}
	}
	return cache
}
