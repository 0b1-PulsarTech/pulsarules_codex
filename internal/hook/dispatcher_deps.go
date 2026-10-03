package hook

import (
	"fmt"
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
	errOut     io.Writer
	projectDir string
	skillsDir  string
	governance func(repo vcs.Repository, status vcs.Status) (string, int)
}

type Deps struct {
	Templates  fs.FS
	Logger     *slog.Logger
	Index      *knowledge.Index
	Out        io.Writer
	ErrOut     io.Writer
	ProjectDir string
	SkillsDir  string
	Governance func(repo vcs.Repository, status vcs.Status) (block string, count int)
}

func NewDispatcher(deps Deps) *Dispatcher {
	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.ErrOut
	if errOut == nil {
		errOut = os.Stderr
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
		errOut:     errOut,
		projectDir: deps.ProjectDir,
		skillsDir:  deps.SkillsDir,
		governance: governanceItem,
	}
}

func (disp *Dispatcher) resolveProjectDir() string {
	if disp.projectDir != "" {
		return disp.projectDir
	}
	return os.Getenv("PULSARULES_PROJECT_DIR")
}

func (disp *Dispatcher) resolveSkillsDir() string {
	if disp.skillsDir != "" {
		return disp.skillsDir
	}
	// The layout is host-specific (.claude/skills vs .opencode/skills): no hardcoded default.
	return os.Getenv("PULSARULES_SKILLS_DIR")
}

func (disp *Dispatcher) warnMissingProjectDir(session *sessionTracker) {
	if disp.resolveProjectDir() != "" {
		return
	}
	if !session.OncePerSession("no-project-dir") {
		return
	}
	msg := "pulsarules_codex: hook ran with no project dir resolved; reinstall the hook " +
		"(pulsarules_cli install) so it exports PULSARULES_PROJECT_DIR"
	// The logger discards output unless PULSARULES_LOG_LEVEL is set, which no hook wrapper sets.
	_, _ = fmt.Fprintln(disp.errOut, msg)
	if disp.logger != nil {
		disp.logger.Warn(msg)
	}
}
