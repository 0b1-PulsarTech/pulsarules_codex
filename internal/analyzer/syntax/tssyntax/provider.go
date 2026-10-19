package tssyntax

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/sandbox"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/pathalias"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/budget"
)

const (
	memoryCeiling = 256 << 20
	fileWall      = 10 * time.Second
)

type Provider struct{ runner sandbox.Runner }

func New() *Provider { return newProvider(memoryCeiling, fileWall) }

func newProvider(memory uint64, wall time.Duration) *Provider {
	return &Provider{runner: sandbox.NewRunner(memory, wall)}
}

func RunChild(args []string, exit func(int)) bool {
	return sandbox.Child(args, handleRequest, exit)
}

func (provider *Provider) Languages() []language.ID {
	return []language.ID{language.TypeScript, language.JavaScript, language.Svelte}
}

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	finder := pathalias.NewFinder(pathalias.DiskReader(corpus.Root, corpus.Read))
	summaries := make(map[string]*syntax.File, len(corpus.Sources))
	session := provider.runner.Open()
	defer session.Close()
	for _, source := range corpus.Sources {
		home, aliases := finder.Lookup(source.Path)
		summary, err := summarize(ctx, session, source, pathalias.Specs(aliases))
		if err != nil {
			return summaries, fmt.Errorf("summarize typescript sources: %w", err)
		}
		applyPrefer(summary, newImportSite(source.Path, home, aliases))
		summaries[source.Path] = summary
	}
	return summaries, nil
}

func summarize(
	ctx context.Context, session *sandbox.Session, source syntax.Source, local []string,
) (*syntax.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("check context: %w", err)
	}
	summary := &syntax.File{}
	text := string(source.Content)
	if reason := budget.CheckSize(text); reason != "" {
		summary.ParseError = reason
		return summary, nil
	}
	request := sandbox.Request{Source: source, Local: local}
	outcome, err := session.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("parse in child: %w", err)
	}
	if outcome.File == nil {
		summary.ParseError = outcome.Skip
		return summary, nil
	}
	return outcome.File, nil
}

func newSummary(path string) *syntax.File {
	return &syntax.File{Path: path, Language: language.ForExtension(filepath.Ext(path)).ID}
}
