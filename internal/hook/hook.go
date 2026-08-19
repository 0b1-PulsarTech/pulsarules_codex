package hook

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/contract"
)

// Dispatch reads the event mode, executes the matching handler, and records
// the execution outcome. It always returns nil (non-blocking); a handler
// failure surfaces only through the execution log.
func (d *Dispatcher) Dispatch(mode string, payload []byte) error {
	in := decodeHookPayload(payload)
	session := NewSessionTrackerFromID(in.SessionID)
	d.warnMissingProjectDir(session)
	start := time.Now()

	var (
		findings int
		err      error
	)
	switch mode {
	case "session-start":
		err = d.emitSessionStart(session)
	case "pre-edit":
		err = d.emitPreEdit(session, in)
	case "post-edit":
		err = d.emitPostEdit(session, in)
	case "pre-search":
		err = d.emitPreSearch(session, in)
	case "user-prompt":
		err = d.emitUserPrompt()
	case "stop":
		findings, err = d.emitStop("Stop", session)
	case "subagent-start":
		err = d.emitSubagentStart()
	case "subagent-stop":
		// Deliberately silent: a subagent must not commit (git stays in the
		// main session), so a dirty-tree block aimed at it only derails its
		// work. Not wired into settings - Claude Code fires no SubagentStop
		// event, so nothing dispatches this mode today; kept as a defensive
		// no-op against a future/legacy caller.
	case "session-end":
		session.Cleanup()
	}

	d.record(mode, session, start, findings, err)
	return nil
}

func (d *Dispatcher) emitSessionStart(session *SessionTracker) error {
	if !session.OncePerSession("session-start") {
		return nil
	}
	text, err := contract.Session(d.templates)
	if err != nil {
		return fmt.Errorf("session contract: %w", err)
	}
	d.emitOutput("SessionStart", text)
	d.emitKnowledgeDrift()
	return nil
}

// why: no OncePerSession gate here. A subagent inherits its parent session's
// id, so gating on that id would let a marker the parent already burned
// suppress the contract for every subagent it spawns - the exact bug this
// mode exists to fix. Every subagent gets the contract fresh, every time.
func (d *Dispatcher) emitSubagentStart() error {
	text, err := contract.Subagent(d.templates)
	if err != nil {
		return fmt.Errorf("subagent contract: %w", err)
	}
	d.emitOutput("SubagentStart", text)
	return nil
}

func (d *Dispatcher) emitContext(asset, event string) error {
	text, err := fs.ReadFile(d.templates, asset)
	if err != nil {
		return fmt.Errorf("read %s: %w", asset, err)
	}
	d.emitOutput(event, strings.TrimSpace(string(text)))
	return nil
}

func (d *Dispatcher) record(
	mode string,
	session *SessionTracker,
	start time.Time,
	findings int,
	err error,
) {
	if d.logger == nil {
		return
	}
	args := []any{
		slog.String("event", mode),
		slog.String("session_id", session.SessionID()),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.Int("findings", findings),
		slog.Bool("ok", err == nil),
	}
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
	}
	d.logger.Info("hook dispatched", args...)
}

func filterInstalled(ids []string, skillsDir string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		// why: skillsDir is under PULSARULES_SKILLS_DIR, a hook-provided skills root.
		if _, err := os.Stat(filepath.Join(skillsDir, id, "SKILL.md")); err == nil {
			out = append(out, id)
		}
	}
	return out
}
