package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

var errProjectDirRequired = errors.New(
	"governance requires --project DIR or PULSARULES_PROJECT_DIR",
)

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
	scope, err := governance.ParseScope(opts.Scope)
	if err != nil {
		return err
	}

	run, err := loadGovernance(inj, opts)
	if err != nil {
		return err
	}

	files := governance.FileSetChanged
	if opts.AllFiles {
		files = governance.FileSetAll
	}

	sess := governance.NewSession(run.repo, run.idx, run.catalog, run.cfg)
	result := sess.Analyze(governance.Request{Scope: scope, Files: files})
	return reportGovernance(os.Stderr, result)
}

func reportGovernance(out io.Writer, report findings.Report) error {
	if len(report.Findings) == 0 {
		_, _ = fmt.Fprintf(out, "governance: %s\n", report.Summary())
		return nil
	}

	_, _ = fmt.Fprint(out, report.Text(findings.CLI(), ""))
	_, _ = fmt.Fprintf(out, "\n%s\n", report.Summary())

	if report.HasErrors() {
		return ErrFindings
	}
	return nil
}

func projectConfig(opts *version.Options, root string) (*config.GovernanceConfig, error) {
	// ApplyPreset silently ignores an unknown name, so reject it here.
	if opts.Preset != "" && !config.ValidPreset(opts.Preset) {
		return nil, fmt.Errorf(
			"invalid --preset %q (want %s)", opts.Preset, strings.Join(config.Presets(), "|"),
		)
	}
	file, err := loader.NewLoader(registry.ConfigRules()).Load(
		root, opts.Config, opts.Preset,
	)
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
	if err = opts.Tuning.Validate(); err != nil {
		return nil, err
	}
	opts.Tuning.Apply(cfg)
	return cfg, nil
}
