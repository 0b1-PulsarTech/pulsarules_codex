package gosyntax

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/names"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Provider struct {
	cache *astcache.Cache
}

func New(cache *astcache.Cache) *Provider {
	return &Provider{cache: cache}
}

func (provider *Provider) Languages() []language.ID { return []language.ID{language.Go} }

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	// An unreadable go.mod empties the import data; arch-boundary reports it off commit scope.
	modulePath := corpus.Module()
	summaries := make(map[string]*syntax.File, len(corpus.Sources))
	for _, source := range corpus.Sources {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return summaries, fmt.Errorf("summarize go sources: %w", ctxErr)
		}
		summaries[source.Path] = provider.summarizeSource(source, modulePath)
	}
	return summaries, nil
}

func (provider *Provider) summarizeSource(
	source syntax.Source,
	modulePath string,
) *syntax.File {
	file := provider.cache.Get(source.Path)
	if file == nil {
		return &syntax.File{
			Path:       source.Path,
			Language:   language.Go,
			ParseError: provider.parseError(source.Path),
		}
	}
	return summarizeFile(provider.cache.FileSet(), source, file, modulePath)
}

func (provider *Provider) parseError(path string) string {
	if err := provider.cache.Err(path); err != nil {
		return err.Error()
	}
	return "file has no syntax tree"
}

func summarizeFile(
	fset *token.FileSet,
	source syntax.Source,
	file *ast.File,
	modulePath string,
) *syntax.File {
	summary := &syntax.File{
		Path:      source.Path,
		Language:  language.Go,
		Functions: functions(fset, file),
		Blocks:    blocks(fset, file),
		Names:     names.Collect(fset, file),
		DocSites:  docSites(fset, source, file),
	}
	summary.Imports, summary.ImportGroups = imports(fset, file, modulePath)
	return summary
}
