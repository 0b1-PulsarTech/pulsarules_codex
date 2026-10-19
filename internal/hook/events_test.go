package hook

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestEvents_EveryRowDispatches(t *testing.T) {
	t.Parallel()

	seenModes := map[string]bool{}
	for _, event := range Events() {
		t.Run(event.Mode, func(t *testing.T) {
			t.Parallel()

			if event.Host == "" || event.Mode == "" || event.handle == nil {
				t.Fatalf("incomplete event row: %+v", event)
			}
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))
			disp, _ := dispatchCapture(Deps{Logger: logger, ProjectDir: t.TempDir()})

			if err := disp.Dispatch(event.Mode, newSessionPayload(t)); err != nil {
				t.Fatalf("Dispatch(%q): %v", event.Mode, err)
			}
			if !strings.Contains(logBuf.String(), "event="+event.Mode+" ") ||
				!strings.Contains(logBuf.String(), "ok=true") {
				t.Errorf("mode %q was not dispatched cleanly:\n%s", event.Mode, logBuf.String())
			}
		})
		if seenModes[event.Mode] {
			t.Errorf("mode %q appears twice in the table", event.Mode)
		}
		seenModes[event.Mode] = true
	}
}

func TestEvents_UnknownModeIsSilent(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{ProjectDir: t.TempDir()})
	if err := disp.Dispatch("no-such-mode", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("unknown mode wrote %q, want nothing", out.String())
	}
}
