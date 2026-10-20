package hook

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/throwsite"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type worktree interface {
	Root() string
	WorktreeStatus() (vcs.Status, error)
}

type Dispatcher struct {
	templates  templates.Set
	logger     *slog.Logger
	index      *knowledge.Index
	router     *fileRouter
	throws     *throwsite.Scanner
	out        io.Writer
	errOut     io.Writer
	projectDir string
	skillsDirs []string
	governance func(root string, status vcs.Status) (string, int)
	open       func(dir string) (worktree, error)
}

type Deps struct {
	Templates  templates.Set
	Logger     *slog.Logger
	Index      *knowledge.Index
	Out        io.Writer
	ErrOut     io.Writer
	ProjectDir string
	SkillsDir  string
	Governance func(root string, status vcs.Status) (block string, count int)
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
	governance := deps.Governance
	if governance == nil {
		governance = func(string, vcs.Status) (string, int) { return "", 0 }
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Dispatcher{
		templates:  deps.Templates,
		logger:     logger,
		index:      deps.Index,
		router:     newFileRouter(deps.Index),
		throws:     throwsite.New(),
		out:        out,
		errOut:     errOut,
		projectDir: deps.ProjectDir,
		skillsDirs: filepath.SplitList(deps.SkillsDir),
		governance: governance,
		open:       func(dir string) (worktree, error) { return vcs.Open(dir) },
	}
}

func (disp *Dispatcher) skillInstalled(id string) bool {
	// PULSARULES_SKILLS_DIR is a path list (project dir, then user dir): installed anywhere counts.
	for _, dir := range disp.skillsDirs {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, id, "SKILL.md")); err == nil {
			return true
		}
	}
	return false
}

func (disp *Dispatcher) warnMissingProjectDir(session *sessionTracker) {
	if disp.projectDir != "" {
		return
	}
	if !session.OncePerSession("no-project-dir") {
		return
	}
	msg := "pulsarules_codex: hook ran with no project dir resolved; reinstall the hook " +
		"(pulsarules_cli install) so it exports PULSARULES_PROJECT_DIR"
	// The logger discards output unless the CLI edge saw PULSARULES_LOG_LEVEL, which no wrapper sets.
	_, _ = fmt.Fprintln(disp.errOut, msg)
	disp.logger.Warn(msg)
}

const stopGovernanceDeadline = 60 * time.Second

type governanceResult struct {
	block    string
	count    int
	timedOut bool
}

func (disp *Dispatcher) governanceWithin(
	root string,
	status vcs.Status,
	entered time.Time,
) governanceResult {
	// The host discards output past its timeout, so the budget runs from Stop entry.
	remaining := stopGovernanceDeadline - time.Since(entered)
	if remaining <= 0 {
		return governanceResult{block: governanceTimeoutNotice(), timedOut: true}
	}
	done := make(chan governanceResult, 1)
	// The host/claude test keeps the rendered Stop timeout above this deadline by a margin.
	// No context reaches the analysis: an expired run is abandoned to die with the process.
	go func() {
		block, count := disp.governance(root, status)
		done <- governanceResult{block: block, count: count}
	}()
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case res := <-done:
		return res
	case <-timer.C:
		return governanceResult{block: governanceTimeoutNotice(), timedOut: true}
	}
}

func governanceTimeoutNotice() string {
	return fmt.Sprintf(
		"Governance did not finish within %ds; run `pulsarules_cli governance` "+
			"before committing (the git hooks still check).\n",
		int(stopGovernanceDeadline/time.Second),
	)
}
