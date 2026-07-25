package hook

import (
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Dispatcher struct {
	templates  fs.FS
	logger     *slog.Logger
	index      *knowledge.Index
	router     *Router
	out        io.Writer
	projectDir string
	governance func(repo vcs.Repository, status vcs.Status) (string, int)
}

type Deps struct {
	Templates  fs.FS
	Logger     *slog.Logger
	Index      *knowledge.Index
	Out        io.Writer
	ProjectDir string
	Governance func(repo vcs.Repository, status vcs.Status) (block string, count int)
}

func NewDispatcher(deps Deps) *Dispatcher {
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	governance := deps.Governance
	if governance == nil {
		governance = func(repo vcs.Repository, status vcs.Status) (string, int) {
			return RunGovernanceCheck(repo, status, deps.Index, analysis.ScopeChanged)
		}
	}
	return &Dispatcher{
		templates:  deps.Templates,
		logger:     deps.Logger,
		index:      deps.Index,
		router:     NewRouter(deps.Index),
		out:        out,
		projectDir: deps.ProjectDir,
		governance: governance,
	}
}

func (d *Dispatcher) resolveProjectDir() string {
	if d.projectDir != "" {
		return d.projectDir
	}
	return os.Getenv("CLAUDE_PROJECT_DIR")
}
