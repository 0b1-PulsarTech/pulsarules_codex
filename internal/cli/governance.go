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

// validScopes names the --scope spellings ParseScope recognizes; anything
// else would silently fall through to ScopeFull.
var validScopes = []string{"full", "commit", "changed"}

// validateScope rejects an unrecognized --scope by name, the same way
// governanceConfig rejects an unrecognized --preset, instead of letting
// analysis.ParseScope's unrecognized-value-to-ScopeFull default run the full
// analyzer set on a typo.
func validateScope(scope string) error {
	if scope == "" || slices.Contains(validScopes, scope) {
		return nil
	}
	return fmt.Errorf("invalid --scope %q (want %s)", scope, strings.Join(validScopes, "|"))
}

// runGovernance runs the full analyzer pipeline against the project via
// Session and prints findings to stderr. It returns an *ExitError{Code: 1} if
// any error-severity finding is produced; main is the only caller that turns
// that into os.Exit.
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

// why: a filtered panel must say so - suppressing in silence is worse than
// over-counting - but an unfiltered run should not carry a "0 suppressed" tail.
func suppressedClause(suppressed int) string {
	if suppressed == 0 {
		return ""
	}
	return fmt.Sprintf(", %d suppressed in generated files", suppressed)
}

// governanceConfig builds the run config from the preset and the optional
// golangci-lint config override. It rejects an unrecognized --preset by name
// before doing any work, instead of ApplyPreset silently no-oping and the
// run falling back to the default preset.
func governanceConfig(opts *cliopts.Options) (*config.GovernanceConfig, error) {
	cfg := config.Defaults()
	if opts.Preset != "" {
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
		return nil, err
	}
	if opts.TypographicSeverity != "" {
		cfg.SetParam("typographic-markers", "severity", opts.TypographicSeverity)
	}
	if err := branchname.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
		return nil, err
	}
	if opts.BranchExtraTypes != "" {
		cfg.SetParam("branch-name", "extra_types", opts.BranchExtraTypes)
	}
	return cfg, nil
}
