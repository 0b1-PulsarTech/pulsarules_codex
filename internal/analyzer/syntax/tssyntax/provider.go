package tssyntax

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/sandbox"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/pathalias"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/budget"
)

const (
	memoryCeiling = 256 * sandbox.MiB
	fileWall      = 10 * time.Second
)

type Provider struct{ runner sandbox.Runner }

func New() *Provider { return newProvider(memoryCeiling, fileWall) }

func newProvider(memory uint64, wall time.Duration) *Provider {
	return &Provider{runner: sandbox.Runner{
		Args:   []string{sandbox.ChildArg},
		Memory: memory,
		Wall:   wall,
	}}
}

func RunChild(args []string, exit func(int)) bool {
	return sandbox.Child(args, handleRequest, exit)
}

func (provider *Provider) Language() language.ID { return language.TypeScript }

func (provider *Provider) Handles(ext string) bool {
	switch strings.ToLower(ext) {
	case ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".svelte":
		return true
	}
	return false
}

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	finder := pathalias.NewFinder(pathalias.DiskReader(corpus.Root, corpus.Read))
	summaries := make(map[string]*syntax.File, len(corpus.Sources))
	session := provider.runner.Open()
	defer session.Close()
	batch := batch{session: session, read: corpus.Read}
	for _, source := range corpus.Sources {
		aliases := finder.For(source.Path)
		summary, err := batch.summarize(ctx, source, pathalias.Specs(aliases))
		if err != nil {
			return summaries, fmt.Errorf("summarize typescript sources: %w", err)
		}
		home, _ := finder.Root(source.Path)
		applyPrefer(summary, importSite{root: home, aliases: aliases})
		summaries[source.Path] = summary
	}
	return summaries, nil
}

type batch struct {
	session *sandbox.Session
	read    func(string) ([]byte, bool)
}

func (batch batch) summarize(
	ctx context.Context, source syntax.Source, local []string,
) (*syntax.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("check context: %w", err)
	}
	summary := newSummary(source.Path)
	content, ok := batch.read(source.Path)
	if !ok {
		summary.ParseError = "file is not readable"
		return summary, nil
	}
	if reason := budget.CheckSize(string(content)); reason != "" {
		summary.ParseError = reason
		return summary, nil
	}
	request := sandbox.Request{
		Path: source.Path, Text: string(content), Local: local, IsTest: source.IsTest,
	}
	outcome, err := batch.session.Do(ctx, request)
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
	ext := strings.ToLower(filepath.Ext(path))
	return &syntax.File{Path: path, Language: language.ForExtension(ext).ID}
}
