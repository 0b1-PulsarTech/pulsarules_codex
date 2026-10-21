package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/character"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/branch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
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
	repo    *vcs.Repository
	cfg     *config.GovernanceConfig
	idx     *knowledge.Index
	catalog *emoji.Catalog
}

func loadGovernance(inj remy.Injector, opts *version.Options) (governanceRun, error) {
	repo, err := remy.Get[*vcs.Repository](inj)
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
	catalog, err := remy.Get[*emoji.Catalog](inj)
	if err != nil {
		return governanceRun{}, fmt.Errorf("get emoji catalog: %w", err)
	}
	return governanceRun{repo: repo, cfg: cfg, idx: idx, catalog: catalog}, nil
}

func runGovernance(inj remy.Injector, opts *version.Options) error {
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

	scope := governance.ParseScope(opts.Scope)
	files := governance.FileSetChanged
	if opts.AllFiles {
		files = governance.FileSetAll
	}

	sess := governance.NewSession(run.repo, "", run.idx, run.catalog, run.cfg)
	result := sess.Analyze(scope, nil, files)
	return reportGovernance(result)
}

func reportGovernance(result governance.Result) error {
	if len(result.Findings) == 0 {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"governance: no findings%s\n",
			suppressedClause(result.SuppressedGenerated),
		)
		return nil
	}

	errCount, warnings, infos := governance.CountBySeverity(result.Findings)
	_, _ = fmt.Fprint(os.Stderr, findings.FormatFindings(result.Findings, findings.StyleCLI, ""))
	_, _ = fmt.Fprintf(
		os.Stderr,
		"\n%d error(s), %d warning(s), %d info(s)%s\n",
		errCount,
		warnings,
		infos,
		suppressedClause(result.SuppressedGenerated),
	)

	if errCount > 0 {
		return ErrFindings
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

func projectConfig(opts *version.Options, root string) (*config.GovernanceConfig, error) {
	// ApplyPreset silently ignores an unknown name, so reject it here.
	if opts.Preset != "" && !config.ValidPreset(opts.Preset) {
		return nil, fmt.Errorf(
			"invalid --preset %q (want %s)", opts.Preset, strings.Join(config.Presets(), "|"),
		)
	}
	file, err := loader.Load(root, opts.Config, opts.Preset)
	if err != nil {
		return nil, err
	}
	return loader.Build(file, opts.Preset)
}

func governanceConfig(opts *version.Options, root string) (*config.GovernanceConfig, error) {
	cfg, err := projectConfig(opts, root)
	if err != nil {
		return nil, err
	}
	if opts.IncludeGenerated {
		cfg.IncludeGenerated = true
	}

	if opts.GolangciConfig != "" {
		cfg.SetParam(golangcilint.ID, golangcilint.ParamConfigPath, opts.GolangciConfig)
	}
	if err = finding.ValidateSeverityName(opts.TypographicSeverity); err != nil {
		return nil, fmt.Errorf("invalid --typographic-severity: %w", err)
	}
	if opts.TypographicSeverity != "" {
		cfg.SetParam(character.TypographicID, pass.ParamSeverity, opts.TypographicSeverity)
	}
	if err = branch.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
		return nil, fmt.Errorf("invalid --branch-extra-types: %w", err)
	}
	if opts.BranchExtraTypes != "" {
		cfg.SetParam(branch.ID, branch.ParamExtraTypes, opts.BranchExtraTypes)
	}
	return cfg, nil
}
