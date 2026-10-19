package dartsyntax

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const BuildTimeout = 45 * time.Second

type Provider struct{}

func New() *Provider { return &Provider{} }

func (provider *Provider) Languages() []language.ID { return []language.ID{language.Dart} }

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	requests := collectRequests(corpus.Sources)
	if len(requests) == 0 {
		return map[string]*syntax.File{}, nil
	}
	return summarizeWith(ctx, requests, provider.prepare)
}

func (provider *Provider) prepare(ctx context.Context) (string, error) {
	dart, lookErr := exec.LookPath("dart")
	if lookErr != nil {
		return "", fmt.Errorf(
			"locate dart: no dart on PATH (fvm users need one, e.g. fvm global): %w", lookErr,
		)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache dir: %w", err)
	}
	buildCtx, cancelBuild := context.WithTimeoutCause(ctx, BuildTimeout, errBuildBound)
	defer cancelBuild()
	return ensureHelper(buildCtx, base, dart)
}
