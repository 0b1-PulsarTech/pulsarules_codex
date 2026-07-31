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

// Dispatcher routes hook events to the correct handler, reading contract text
// from embedded templates. A hook must never block, so Dispatch always returns
// nil; internal failures are reported through logger instead.
type Dispatcher struct {
	templates  fs.FS
	logger     *slog.Logger
	index      *knowledge.Index
	router     *Router
	out        io.Writer
	projectDir string
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
	// ProjectDir is the repo root handlers resolve paths against; empty
	// defaults to os.Getenv("CLAUDE_PROJECT_DIR").
	ProjectDir string
	// Governance runs the analyzer pipeline for a dirty worktree; nil
	// defaults to the real pipeline (RunGovernanceCheck against Index).
	Governance func(repo vcs.Repository, status vcs.Status) (block string, count int)
}

// NewDispatcher builds a Dispatcher from deps, defaulting Out to os.Stdout,
// ProjectDir to CLAUDE_PROJECT_DIR, and Governance to the real analyzer
// pipeline, so a production caller can omit whatever it does not override.
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

// resolveProjectDir returns the configured project dir, falling back to
// CLAUDE_PROJECT_DIR so a production Dispatcher built without ProjectDir set
// keeps working.
func (d *Dispatcher) resolveProjectDir() string {
	if d.projectDir != "" {
		return d.projectDir
	}
	return os.Getenv("CLAUDE_PROJECT_DIR")
}
