package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis/projectcfg"
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

type governanceRun struct {
	repo vcs.Repository
	cfg  *config.GovernanceConfig
	idx  *knowledge.Index
}

func loadGovernance(inj remy.Injector, opts *cliopts.Options) (governanceRun, error) {
	repo, err := remy.Get[vcs.Repository](inj)
	if err != nil {
		return governanceRun{}, fmt.Errorf("open repository: %w", err)
	}
	cfg, err := governanceConfig(opts, repo.Root())
	if err != nil {
		return governanceRun{}, err
	}

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return governanceRun{}, fmt.Errorf("get knowledge index: %w", err)
	}
	return governanceRun{repo: repo, cfg: cfg, idx: idx}, nil
}

func runGovernance(inj remy.Injector, opts *cliopts.Options) error {
	if opts.ProjectDir == "" && os.Getenv("PULSARULES_PROJECT_DIR") == "" {
		return errProjectDirRequired
	}
	if err := validateScope(opts.Scope); err != nil {
		return err
	}

	run, err := loadGovernance(inj, opts)
	if err != nil {
		return err
	}

	scope := analysis.ParseScope(opts.Scope)
	files := analysis.FileSetChanged
	if opts.AllFiles {
		files = analysis.FileSetAll
	}

	result := analysis.NewSession(run.repo, "", run.idx, run.cfg).Analyze(scope, nil, files)
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

func projectConfig(opts *cliopts.Options, root string) (*config.GovernanceConfig, error) {
	// ApplyPreset silently ignores an unknown name, so reject it here.
	if opts.Preset != "" && !config.ValidPreset(opts.Preset) {
		return nil, fmt.Errorf(
			"invalid --preset %q (want %s)", opts.Preset, strings.Join(config.Presets(), "|"),
		)
	}
	file, err := projectcfg.Load(root, opts.Config, opts.Preset)
	if err != nil {
		return nil, err
	}
	return projectcfg.Build(file, opts.Preset)
}

func governanceConfig(opts *cliopts.Options, root string) (*config.GovernanceConfig, error) {
	cfg, err := projectConfig(opts, root)
	if err != nil {
		return nil, err
	}
	if opts.IncludeGenerated {
		cfg.IncludeGenerated = true
	}

	if opts.GolangciConfig != "" {
		cfg.SetParam("golangci-lint", "config_path", opts.GolangciConfig)
	}
	if err = core.ValidateSeverityName(opts.TypographicSeverity); err != nil {
		return nil, fmt.Errorf("invalid --typographic-severity: %w", err)
	}
	if opts.TypographicSeverity != "" {
		cfg.SetParam("typographic-markers", "severity", opts.TypographicSeverity)
	}
	if err = branchname.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
		return nil, fmt.Errorf("invalid --branch-extra-types: %w", err)
	}
	if opts.BranchExtraTypes != "" {
		cfg.SetParam("branch-name", "extra_types", opts.BranchExtraTypes)
	}
	return cfg, nil
}
