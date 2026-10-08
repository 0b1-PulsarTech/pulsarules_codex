package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/branchname"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

var errProjectDirRequired = errors.New(
	"governance requires --project DIR or PULSARULES_PROJECT_DIR",
)

func validateScope(scope string) error {
	validScopes := []string{"full", "commit", "changed"}
	// ParseScope falls back to ScopeFull on an unknown spelling, so reject it here.
	if scope == "" || slices.Contains(validScopes, scope) {
		return nil
	}
	return fmt.Errorf("invalid --scope %q (want %s)", scope, strings.Join(validScopes, "|"))
}

func runGovernance(inj remy.Injector, opts *cliopts.Options) error {
	if opts.ProjectDir == "" && os.Getenv("PULSARULES_PROJECT_DIR") == "" {
		return errProjectDirRequired
	}
	if err := validateScope(opts.Scope); err != nil {
		return err
	}

	repo, err := remy.Get[vcs.Repository](inj)
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}
	cfg, err := governanceConfig(opts)
	if err != nil {
		return err
	}

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}

	scope := analysis.ParseScope(opts.Scope)
	files := analysis.FileSetChanged
	if opts.AllFiles {
		files = analysis.FileSetAll
	}

	result := analysis.NewSession(repo, "", idx, cfg).Analyze(scope, nil, files)
	return reportGovernance(result)
}

func reportGovernance(result analysis.Result) error {
	if len(result.Findings) == 0 {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"governance: no findings%s\n",
			suppressedClause(result.SuppressedGenerated),
		)
		return nil
	}

	errCount, warnings, infos := analysis.CountBySeverity(result.Findings)
	_, _ = fmt.Fprint(os.Stderr, analysis.FormatFindings(result.Findings, analysis.StyleCLI, ""))
	_, _ = fmt.Fprintf(
		os.Stderr,
		"\n%d error(s), %d warning(s), %d info(s)%s\n",
		errCount,
		warnings,
		infos,
		suppressedClause(result.SuppressedGenerated),
	)

	if errCount > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}

func suppressedClause(suppressed int) string {
	if suppressed == 0 {
		// An unfiltered run carries no "0 suppressed" tail.
		return ""
	}
	return fmt.Sprintf(", %d suppressed in generated files", suppressed)
}

func governanceConfig(opts *cliopts.Options) (*config.GovernanceConfig, error) {
	cfg := config.Defaults()
	if opts.Preset != "" {
		// ApplyPreset silently ignores an unknown name, so reject it here.
		if !config.ValidPreset(opts.Preset) {
			return nil, fmt.Errorf(
				"invalid --preset %q (want %s)", opts.Preset, strings.Join(config.Presets(), "|"),
			)
		}
		cfg.Preset = opts.Preset
	}
	cfg.ApplyPreset()
	cfg.IncludeGenerated = opts.IncludeGenerated

	if opts.GolangciConfig != "" {
		cfg.SetParam("golangci-lint", "config_path", opts.GolangciConfig)
	}
	if err := core.ValidateSeverityName(opts.TypographicSeverity); err != nil {
		return nil, fmt.Errorf("invalid --typographic-severity: %w", err)
	}
	if opts.TypographicSeverity != "" {
		cfg.SetParam("typographic-markers", "severity", opts.TypographicSeverity)
	}
	if err := branchname.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
		return nil, fmt.Errorf("invalid --branch-extra-types: %w", err)
	}
	if opts.BranchExtraTypes != "" {
		cfg.SetParam("branch-name", "extra_types", opts.BranchExtraTypes)
	}
	return cfg, nil
}
