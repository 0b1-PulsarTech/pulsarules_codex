package governance

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
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

type Command struct {
	project          string
	preset           string
	config           string
	scope            string
	golangciConfig   string
	tuning           flags.Tuning
	allFiles         bool
	includeGenerated bool
	env              flags.Environment
}

func New(env flags.Environment) *Command {
	return &Command{env: env}
}

func (cmd *Command) ProjectDir() string {
	if cmd.project == "" {
		return cmd.env.ProjectDir
	}
	return cmd.project
}

type governanceRun struct {
	repo    *vcs.Repository
	cfg     *config.GovernanceConfig
	idx     *knowledge.Index
	catalog *emoji.Catalog
}

func (cmd *Command) loadGovernance(inj remy.Injector) (governanceRun, error) {
	repo, err := remy.Get[*vcs.Repository](inj)
	if err != nil {
		return governanceRun{}, fmt.Errorf("open repository: %w", err)
	}
	cfg, err := cmd.governanceConfig(repo.Root())
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

func (cmd *Command) Run(inj remy.Injector, _ []string) error {
	if cmd.ProjectDir() == "" {
		return errProjectDirRequired
	}
	scope, err := governance.ParseScope(cmd.scope)
	if err != nil {
		return err
	}

	run, err := cmd.loadGovernance(inj)
	if err != nil {
		return err
	}

	files := governance.FileSetChanged
	if cmd.allFiles {
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
		return command.ErrFindings
	}
	return nil
}

func (cmd *Command) projectConfig(root string) (*config.GovernanceConfig, error) {
	// ApplyPreset silently ignores an unknown name, so reject it here.
	if cmd.preset != "" && !config.ValidPreset(cmd.preset) {
		return nil, errors.New(config.UnknownPreset("invalid --preset", cmd.preset))
	}
	file, err := loader.NewLoader(registry.ConfigRules()).Load(
		root, cmd.config, cmd.preset,
	)
	if err != nil {
		return nil, err
	}
	return loader.Build(file, cmd.preset)
}

func (cmd *Command) governanceConfig(root string) (*config.GovernanceConfig, error) {
	cfg, err := cmd.projectConfig(root)
	if err != nil {
		return nil, err
	}
	if cmd.includeGenerated {
		cfg.IncludeGenerated = true
	}

	if cmd.golangciConfig != "" {
		cfg.SetParam(golangcilint.ID, golangcilint.ParamConfigPath, cmd.golangciConfig)
	}
	if err = cmd.tuning.Validate(); err != nil {
		return nil, err
	}
	cmd.tuning.Apply(cfg)
	return cfg, nil
}
