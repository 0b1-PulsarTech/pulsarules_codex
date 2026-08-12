package hook

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Dispatcher routes hook events to the correct handler, reading contract text
// from embedded templates. A hook must never block, so Dispatch always returns
// nil; internal failures are reported through logger instead.
type Dispatcher struct {
	templates  fs.FS
	logger     *slog.Logger
	index      *knowledge.Index
	router     *Router
	out        io.Writer
	errOut     io.Writer
	projectDir string
	skillsDir  string
	governance func(repo vcs.Repository, status vcs.Status) (string, int)
}

// Deps carries what the Dispatcher needs instead of constructing it, so a
// test can observe output and stub governance without touching global state.
type Deps struct {
	// Templates provides the embedded hook contract text (hooks/*.txt).
	Templates fs.FS
	// Logger records one execution-summary line per dispatch; nil skips logging.
	Logger *slog.Logger
	// Index feeds rule-body injection in commit checks and post-edit skill
	// routing; nil skips both.
	Index *knowledge.Index
	// Out receives the emitted hook JSON; nil defaults to os.Stdout.
	Out io.Writer
	// ErrOut receives diagnostics both hosts surface (e.g. the stale-install
	// warning); nil defaults to os.Stderr.
	ErrOut io.Writer
	// ProjectDir is the repo root handlers resolve paths against; empty
	// defaults to os.Getenv("PULSARULES_PROJECT_DIR").
	ProjectDir string
	// SkillsDir is the installed-skills directory post-edit and pre-search
	// filter against; empty defaults to os.Getenv("PULSARULES_SKILLS_DIR").
	// Its layout is host-specific (.claude/skills vs .opencode/skills), so
	// generic Go never hardcodes either - the host's install wiring sets it.
	SkillsDir string
	// Governance runs the analyzer pipeline for a dirty worktree; nil
	// defaults to the real pipeline (RunGovernanceCheck against Index).
	Governance func(repo vcs.Repository, status vcs.Status) (block string, count int)
}

// NewDispatcher builds a Dispatcher from deps, defaulting Out to os.Stdout,
// ErrOut to os.Stderr, ProjectDir to PULSARULES_PROJECT_DIR, SkillsDir to
// PULSARULES_SKILLS_DIR, and Governance to the real analyzer pipeline, so a
// production caller can omit whatever it does not override.
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
		errOut:     errOut,
		projectDir: deps.ProjectDir,
		skillsDir:  deps.SkillsDir,
		governance: governance,
	}
}

// resolveProjectDir returns the configured project dir, falling back to
// PULSARULES_PROJECT_DIR so a production Dispatcher built without ProjectDir
// set keeps working.
func (d *Dispatcher) resolveProjectDir() string {
	if d.projectDir != "" {
		return d.projectDir
	}
	return os.Getenv("PULSARULES_PROJECT_DIR")
}

// resolveSkillsDir returns the configured skills dir, falling back to
// PULSARULES_SKILLS_DIR so a production Dispatcher built without SkillsDir
// set keeps working. Its layout is host-specific, so this never falls back
// to a hardcoded ".claude/skills" or ".opencode/skills" default.
func (d *Dispatcher) resolveSkillsDir() string {
	if d.skillsDir != "" {
		return d.skillsDir
	}
	return os.Getenv("PULSARULES_SKILLS_DIR")
}

// why: stderr, not just the logger - the logger discards everything unless
// PULSARULES_LOG_LEVEL is set, which no installed hook wrapper sets.
func (d *Dispatcher) warnMissingProjectDir(session *SessionTracker) {
	if d.resolveProjectDir() != "" {
		return
	}
	if !session.OncePerSession("no-project-dir") {
		return
	}
	msg := "pulsarules_codex: hook ran with no project dir resolved; reinstall the hook " +
		"(pulsarules_cli install) so it exports PULSARULES_PROJECT_DIR"
	_, _ = fmt.Fprintln(d.errOut, msg)
	if d.logger != nil {
		d.logger.Warn(msg)
	}
}
