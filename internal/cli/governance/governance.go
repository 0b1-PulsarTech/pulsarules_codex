package governance

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
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

func (cmd *Command) loadGovernance(
	openRepository func() (*vcs.Repository, error),
	index *knowledge.Index,
	loadCatalog func() (*emoji.Catalog, error),
	loader func() *loader.Loader,
) (governanceRun, error) {
	repo, err := openRepository()
	if err != nil {
		return governanceRun{}, fmt.Errorf("open repository: %w", err)
	}
	cfg, err := cmd.governanceConfig(loader(), repo.Root())
	if err != nil {
		return governanceRun{}, err
	}

	catalog, err := loadCatalog()
	if err != nil {
		return governanceRun{}, fmt.Errorf("get emoji catalog: %w", err)
	}
	return governanceRun{repo: repo, cfg: cfg, idx: index, catalog: catalog}, nil
}

func (cmd *Command) Run(
	openRepository func() (*vcs.Repository, error),
	index *knowledge.Index,
	loadCatalog func() (*emoji.Catalog, error),
	loader func() *loader.Loader,
) error {
	if cmd.ProjectDir() == "" {
		return errProjectDirRequired
	}
	scope, err := parseScope(cmd.scope)
	if err != nil {
		return err
	}

	run, err := cmd.loadGovernance(openRepository, index, loadCatalog, loader)
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

func parseScope(raw string) (governance.Scope, error) {
	scope, err := governance.ParseScope(raw)
	if err != nil {
		return scope, fmt.Errorf("invalid --scope %w", err)
	}
	return scope, nil
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

func (cmd *Command) projectConfig(
	loaderItem *loader.Loader,
	root string,
) (*config.GovernanceConfig, error) {
	// ApplyPreset silently ignores an unknown name, so reject it here.
	if cmd.preset != "" && !config.ValidPreset(cmd.preset) {
		return nil, errors.New(config.UnknownPreset("invalid --preset", cmd.preset))
	}
	file, err := loaderItem.Load(root, cmd.config, cmd.preset)
	if err != nil {
		return nil, err
	}
	return loader.Build(file, cmd.preset)
}

func (cmd *Command) governanceConfig(
	loader *loader.Loader,
	root string,
) (*config.GovernanceConfig, error) {
	cfg, err := cmd.projectConfig(loader, root)
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
