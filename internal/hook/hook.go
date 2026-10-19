package hook

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
)

func (disp *Dispatcher) Dispatch(mode string, payload []byte) error {
	in := decodeHookPayload(payload)
	session := newSessionTracker(in.SessionID)
	disp.warnMissingProjectDir(session)
	start := time.Now()

	var (
		outcome stopOutcome
		err     error
	)
	switch mode {
	case "session-start":
		err = disp.emitSessionStart(session)
	case "pre-edit":
		err = disp.emitPreEdit(session, in)
	case "post-edit":
		err = disp.emitPostEdit(session, in)
	case "pre-search":
		err = disp.emitPreSearch(session, in)
	case "user-prompt":
		err = disp.emitUserPrompt()
	case "stop":
		outcome, err = disp.emitStop("Stop", session)
	case "subagent-start":
		err = disp.emitSubagentStart()
	case "session-end":
		session.Cleanup()
	}

	disp.record(session, dispatchRecord{mode: mode, start: start, outcome: outcome, err: err})
	// A hook must never block: a handler failure surfaces only through the log.
	return nil
}

func (disp *Dispatcher) emitSessionStart(session *sessionTracker) error {
	if !session.OncePerSession("session-start") {
		return nil
	}
	text, err := contract.Session(disp.templates)
	if err != nil {
		return fmt.Errorf("session contract: %w", err)
	}
	disp.emitOutput("SessionStart", text)
	disp.emitKnowledgeDrift()
	return nil
}

func (disp *Dispatcher) emitSubagentStart() error {
	text, err := contract.Subagent(disp.templates)
	if err != nil {
		return fmt.Errorf("subagent contract: %w", err)
	}
	// No OncePerSession gate: a subagent shares its parent's session id and would be muted.
	disp.emitOutput("SubagentStart", text)
	return nil
}

func (disp *Dispatcher) emitContext(asset, event string) error {
	return disp.emitContextWithNotice(asset, event, "")
}

func (disp *Dispatcher) emitContextWithNotice(asset, event, notice string) error {
	text, err := disp.templates.Text(asset)
	if err != nil {
		return err
	}
	disp.emitOutput(event, joinNotice(notice, text))
	return nil
}

type stopOutcome struct {
	findings int
	timedOut bool
}

type dispatchRecord struct {
	mode    string
	start   time.Time
	outcome stopOutcome
	err     error
}

func (disp *Dispatcher) record(session *sessionTracker, entry dispatchRecord) {
	if disp.logger == nil {
		return
	}
	args := []any{
		slog.String("event", entry.mode),
		slog.String("session_id", session.SessionID()),
		slog.Int64("duration_ms", time.Since(entry.start).Milliseconds()),
		slog.Int("findings", entry.outcome.findings),
		slog.Bool("ok", entry.err == nil),
	}
	if entry.outcome.timedOut {
		args = append(args, slog.Bool("timed_out", true))
	}
	if entry.err != nil {
		args = append(args, slog.String("error", entry.err.Error()))
	}
	disp.logger.Info("hook dispatched", args...)
}

func filterInstalled(ids []string, skillsDir string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		// The skillsDir is under PULSARULES_SKILLS_DIR, a hook-provided skills root.
		if _, err := os.Stat(filepath.Join(skillsDir, id, "SKILL.md")); err == nil {
			out = append(out, id)
		}
	}
	return out
}
