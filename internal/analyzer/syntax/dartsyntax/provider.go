package dartsyntax

import (
	"context"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/dartsyntax/helperbin"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Provider struct{ helper *helperbin.Locator }

func New() *Provider {
	const helperRoot = "helper"
	return &Provider{helper: helperbin.NewLocator(helperFS, helperRoot)}
}

func (provider *Provider) Languages() []language.ID { return []language.ID{language.Dart} }

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	requests := collectRequests(corpus.Sources)
	if len(requests) == 0 {
		return map[string]*syntax.File{}, nil
	}
	return summarizeWith(ctx, requests, provider.helper.Exe)
}
