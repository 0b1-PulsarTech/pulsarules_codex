package hook

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (d *Dispatcher) Dispatch(mode string, payload []byte) error {
	in := decodeHookPayload(payload)
	session := NewSessionTrackerFromID(in.SessionID)
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
	case "session-end":
		session.Cleanup()
	}

	d.record(mode, session, start, findings, err)
	// A hook must never block: a handler failure surfaces only through the log.
	return nil
}

func (d *Dispatcher) emitSessionStart(session *SessionTracker) error {
	if !session.OncePerSession("session-start") {
		return nil
	}
	return d.emitContext("hooks/contract.txt", "SessionStart")
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
		// The skillsDir is under PULSARULES_SKILLS_DIR, a hook-provided skills root.
		if _, err := os.Stat(filepath.Join(skillsDir, id, "SKILL.md")); err == nil {
			out = append(out, id)
		}
	}
	return out
}
