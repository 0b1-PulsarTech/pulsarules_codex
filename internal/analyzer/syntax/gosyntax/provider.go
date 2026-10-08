package gosyntax

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/names"
)

type Provider struct {
	cache *astcache.Cache
}

func New(cache *astcache.Cache) *Provider {
	return &Provider{cache: cache}
}

func (provider *Provider) Language() syntax.Language { return syntax.LangGo }

func (provider *Provider) Handles(ext string) bool { return strings.EqualFold(ext, ".go") }

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	// An unreadable go.mod empties the import data; arch-boundary reports it off commit scope.
	modulePath, err := pass.ModulePath(corpus.Root)
	if err != nil {
		modulePath = ""
	}
	summaries := make(map[string]*syntax.File, len(corpus.Sources))
	for _, source := range corpus.Sources {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return summaries, fmt.Errorf("summarize go sources: %w", ctxErr)
		}
		summaries[source.Path] = provider.summarizeSource(source, modulePath, corpus.Read)
	}
	return summaries, nil
}

func (provider *Provider) summarizeSource(
	source syntax.Source,
	modulePath string,
	read func(string) ([]byte, bool),
) *syntax.File {
	file := provider.cache.Get(source.Path)
	if file == nil {
		return &syntax.File{
			Path:       source.Path,
			Language:   syntax.LangGo,
			ParseError: provider.parseError(source.Path, read),
		}
	}
	return summarizeFile(provider.cache.FileSet(), source, file, modulePath)
}

func (provider *Provider) parseError(path string, read func(string) ([]byte, bool)) string {
	if _, ok := read(path); !ok {
		return "file is not readable"
	}
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
		Language:  syntax.LangGo,
		Functions: functions(fset, file),
		Blocks:    blocks(fset, file),
		Names:     names.Collect(fset, file),
		DocSites:  docSites(fset, source, file),
	}
	summary.Imports, summary.ImportGroups = imports(fset, file, modulePath)
	return summary
}
