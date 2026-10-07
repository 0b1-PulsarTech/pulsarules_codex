package dartsyntax

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

const BuildTimeout = 45 * time.Second

type Provider struct {
	buildBound time.Duration
}

func New() *Provider { return &Provider{buildBound: BuildTimeout} }

func (provider *Provider) Language() syntax.Language { return syntax.LangDart }

func (provider *Provider) Handles(ext string) bool { return ext == ".dart" }

func (provider *Provider) Summarize(
	ctx context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	requests := collectRequests(corpus.Sources, corpus.Read)
	if len(requests) == 0 {
		return map[string]*syntax.File{}, nil
	}
	const runTimeout = 30 * time.Second
	exe, err := provider.prepare(ctx)
	if err != nil {
		return nil, err
	}
	runCtx, cancelRun := context.WithTimeout(ctx, runTimeout)
	defer cancelRun()
	return runHelper(runCtx, exe, "", requests)
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
	bound := provider.buildBound
	if bound <= 0 {
		bound = BuildTimeout
	}
	buildCtx, cancelBuild := context.WithTimeoutCause(ctx, bound, errBuildBound)
	defer cancelBuild()
	return ensureHelper(buildCtx, base, dart)
}
