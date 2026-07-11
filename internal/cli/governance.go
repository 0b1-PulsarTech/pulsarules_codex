package cli

import (
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// runGovernance runs the full analyzer pipeline against the project via
// Session and prints findings to stderr. It returns an *ExitError{Code: 1} if
// any error-severity finding is produced; main is the only caller that turns
// that into os.Exit.
func runGovernance(inj remy.Injector, opts *cliopts.Options) error {
	if opts.ProjectDir == "" && os.Getenv("CLAUDE_PROJECT_DIR") == "" {
		return fmt.Errorf("governance requires --project DIR or CLAUDE_PROJECT_DIR")
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

	scope := analysis.ParseScope(opts.Scope)
	files := analysis.FileSetChanged
	if opts.AllFiles {
		files = analysis.FileSetAll
	}

	findings := analysis.NewSession(repo, "", idx, cfg).Analyze(scope, nil, files)
	if len(findings) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "governance: no findings")
		return nil
	}

	errCount, warnings, infos := analysis.CountBySeverity(findings)
	_, _ = fmt.Fprint(os.Stderr, analysis.FormatFindings(findings, analysis.StyleCLI, ""))
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

// governanceConfig builds the run config from the preset and the optional
// golangci-lint config override.
func governanceConfig(opts *cliopts.Options) *config.GovernanceConfig {
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
