package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

var errProjectDirRequired = errors.New(
	"governance requires --project DIR or CLAUDE_PROJECT_DIR",
)

func runGovernance(inj remy.Injector, opts *version.Options) error {
	if opts.ProjectDir == "" && os.Getenv("CLAUDE_PROJECT_DIR") == "" {
		return errProjectDirRequired
	}

	repo, err := remy.Get[vcs.Repository](inj)
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}
	cfg := governanceConfig(opts)

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}

	scope := governance.ParseScope(opts.Scope)
	files := governance.FileSetChanged
	if opts.AllFiles {
		files = governance.FileSetAll
	}

	findingsItem := governance.NewSession(repo, "", idx, cfg).Analyze(scope, nil, files)
	if len(findingsItem) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "governance: no findings")
		return nil
	}

	errCount, warnings, infos := governance.CountBySeverity(findingsItem)
	_, _ = fmt.Fprint(os.Stderr, findings.FormatFindings(findingsItem, findings.StyleCLI, ""))
	_, _ = fmt.Fprintf(
		os.Stderr,
		"\n%d error(s), %d warning(s), %d info(s)\n",
		errCount,
		warnings,
		infos,
	)

	if errCount > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}

func governanceConfig(opts *version.Options) *config.GovernanceConfig {
	cfg := config.Defaults()
	if opts.Preset != "" {
		cfg.Preset = opts.Preset
	}
	cfg.ApplyPreset()

	if opts.GolangciConfig != "" {
		cfg.SetParam("golangci-lint", "config_path", opts.GolangciConfig)
	}
	return cfg
}
