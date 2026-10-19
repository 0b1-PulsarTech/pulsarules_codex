package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func rowProject(t *testing.T) (dir, skillsDir string) {
	t.Helper()

	dir, skillsDir = t.TempDir(), t.TempDir()
	gitInit(t, dir)
	for name, body := range map[string]string{"go.mod": "module x\n", "dirty.go": "package x\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	installSkillFixture(t, skillsDir, "gopls-navigation")
	return dir, skillsDir
}

func TestEvents_EveryRowDispatches(t *testing.T) {
	t.Parallel()

	seenModes := map[string]bool{}
	idx := preEditIndex(t)
	for _, event := range Events() {
		t.Run(event.Mode, func(t *testing.T) {
			t.Parallel()

			if event.Host == "" || event.Mode == "" || event.handle == nil {
				t.Fatalf("incomplete event row: %+v", event)
			}
			dir, skillsDir := rowProject(t)
			id := uniqueSessionID(t)
			disp, out := dispatchCapture(Deps{
				Index: idx, ProjectDir: dir, SkillsDir: skillsDir,
				Governance: stubGovernance("\nGovernance checks:\n  - stub\n", 1),
			})
			session := newSessionTracker(id)
			session.OncePerSession("probe")
			payload := fmt.Appendf(nil,
				`{"session_id":%q,"tool_name":"Bash","tool_input":`+
					`{"file_path":"dirty.go","command":"grep -r x ."}}`, id)

			if err := disp.Dispatch(event.Mode, payload); err != nil {
				t.Fatalf("Dispatch(%q): %v", event.Mode, err)
			}
			if event.Mode == "session-end" {
				if !session.OncePerSession("probe") {
					t.Error("session-end left the session markers in place")
				}
				return
			}
			var emitted hookOutput
			if err := json.Unmarshal(out.Bytes(), &emitted); err != nil {
				t.Fatalf("mode %q emitted no hook output: %v (%q)", event.Mode, err, out.String())
			}
			if got := emitted.HookSpecificOutput.HookEventName; got != event.Host {
				t.Errorf("hookEventName = %q, want the row host %q", got, event.Host)
			}
		})
		if seenModes[event.Mode] {
			t.Errorf("mode %q appears twice in the table", event.Mode)
		}
		seenModes[event.Mode] = true
	}
}

func TestEvents_StopTimeoutOutlastsGovernance(t *testing.T) {
	t.Parallel()

	const minimumMargin = 15 * time.Second
	for _, event := range Events() {
		if event.Mode != "stop" {
			continue
		}
		hostTimeout := time.Duration(event.Timeout) * time.Second
		if hostTimeout < stopGovernanceDeadline+minimumMargin {
			t.Errorf("Stop timeout %s must exceed the governance deadline %s by %s",
				hostTimeout, stopGovernanceDeadline, minimumMargin)
		}
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
