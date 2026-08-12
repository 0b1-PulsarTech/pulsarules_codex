package hook

import (
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Dispatcher struct {
	templates  fs.FS
	logger     *slog.Logger
	index      *knowledge.Index
	router     *fileRouter
	out        io.Writer
	projectDir string
	skillsDir  string
	governance func(repo vcs.Repository, status vcs.Status) (string, int)
}

type Deps struct {
	Templates  fs.FS
	Logger     *slog.Logger
	Index      *knowledge.Index
	Out        io.Writer
	ProjectDir string
	SkillsDir  string
	Governance func(repo vcs.Repository, status vcs.Status) (block string, count int)
}

func NewDispatcher(deps Deps) *Dispatcher {
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	governanceItem := deps.Governance
	if governanceItem == nil {
		governanceItem = func(repo vcs.Repository, status vcs.Status) (string, int) {
			return RunGovernanceCheck(repo, status, deps.Index, governance.ScopeChanged)
		}
	}
	return &Dispatcher{
		templates:  deps.Templates,
		logger:     deps.Logger,
		index:      deps.Index,
		router:     newFileRouter(deps.Index),
		out:        out,
		projectDir: deps.ProjectDir,
		skillsDir:  deps.SkillsDir,
		governance: governanceItem,
	}
}

func (d *Dispatcher) resolveProjectDir() string {
	if d.projectDir != "" {
		return d.projectDir
	}
	return os.Getenv("PULSARULES_PROJECT_DIR")
}

func (d *Dispatcher) resolveSkillsDir() string {
	if d.skillsDir != "" {
		return d.skillsDir
	}
	// The layout is host-specific (.claude/skills vs .opencode/skills): no hardcoded default.
	return os.Getenv("PULSARULES_SKILLS_DIR")
}
