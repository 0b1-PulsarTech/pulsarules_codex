package hook

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// TestDispatcher_ResolveProjectDir pins the host-neutral fallback: a
// Dispatcher built without ProjectDir set reads PULSARULES_PROJECT_DIR, never
// a host's own variable. It sets an environment variable, so it cannot run
// in parallel.
func TestDispatcher_ResolveProjectDir(t *testing.T) {
	testCases := []struct {
		name       string
		projectDir string
		envDir     string
		want       string
	}{
		{
			name: "explicit dir wins", projectDir: "/explicit",
			envDir: "/from-env", want: "/explicit",
		},
		{
			name: "falls back to the env var", projectDir: "",
			envDir: "/from-env", want: "/from-env",
		},
		{name: "empty when neither is set", projectDir: "", envDir: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_PROJECT_DIR", testCase.envDir)
			d := NewDispatcher(Deps{ProjectDir: testCase.projectDir})
			if got := d.resolveProjectDir(); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestDispatcher_ResolveSkillsDir pins the B5 fix: a Dispatcher built without
// SkillsDir set reads PULSARULES_SKILLS_DIR, never a hardcoded
// ".claude/skills" or ".opencode/skills" default. It sets an environment
// variable, so it cannot run in parallel.
func TestDispatcher_ResolveSkillsDir(t *testing.T) {
	testCases := []struct {
		name      string
		skillsDir string
		envDir    string
		want      string
	}{
		{
			name: "explicit dir wins", skillsDir: "/explicit/skills",
			envDir: "/from-env/skills", want: "/explicit/skills",
		},
		{
			name: "falls back to the env var", skillsDir: "",
			envDir: "/from-env/skills", want: "/from-env/skills",
		},
		{name: "empty when neither is set", skillsDir: "", envDir: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_SKILLS_DIR", testCase.envDir)
			d := NewDispatcher(Deps{SkillsDir: testCase.skillsDir})
			if got := d.resolveSkillsDir(); got != testCase.want {
				t.Errorf("resolveSkillsDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestDispatcher_WarnsOnceOnMissingProjectDir pins the loud diagnostic a
// stale hook install needs: with no project dir resolved, exactly one
// warning goes to stderr naming the required reinstall, however many times
// it dispatches - so stop/pre-search silently vanishing (see emitStop,
// emitPreSearch) is diagnosable, not quiet. Sets an env var, so no t.Parallel.
func TestDispatcher_WarnsOnceOnMissingProjectDir(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "")

	var logBuf, errBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, _ := dispatchCapture(Deps{Logger: logger, ErrOut: &errBuf})
	id := uniqueSessionID(t)
	payload := fmt.Appendf(nil, `{"session_id":%q}`, id)

	_ = disp.Dispatch("session-start", payload)
	_ = disp.Dispatch("user-prompt", payload)

	if got := strings.Count(errBuf.String(), "reinstall"); got != 1 {
		t.Errorf(
			"expected exactly one stderr reinstall warning across two dispatches, got %d in:\n%s",
			got, errBuf.String(),
		)
	}
	if got := strings.Count(logBuf.String(), "reinstall"); got != 1 {
		t.Errorf(
			"expected exactly one logged reinstall warning across two dispatches, got %d in:\n%s",
			got, logBuf.String(),
		)
	}
}

// TestDispatcher_NoWarningWithProjectDir proves the warning is specific to
// the missing-project-dir case, not emitted on every dispatch.
func TestDispatcher_NoWarningWithProjectDir(t *testing.T) {
	t.Parallel()

	var logBuf, errBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, _ := dispatchCapture(
		Deps{Logger: logger, ErrOut: &errBuf, ProjectDir: t.TempDir()},
	)

	_ = disp.Dispatch("session-start", newSessionPayload(t))

	if strings.Contains(errBuf.String(), "reinstall") {
		t.Errorf(
			"unexpected stderr reinstall warning with a resolved project dir:\n%s",
			errBuf.String(),
		)
	}
	if strings.Contains(logBuf.String(), "reinstall") {
		t.Errorf(
			"unexpected logged reinstall warning with a resolved project dir:\n%s",
			logBuf.String(),
		)
	}
}

// TestDispatcher_WarnsToStderrWithoutLogger proves B3: the stderr warning
// fires even when no logger is configured at all (the real-world default,
// since obs.New discards everything absent PULSARULES_LOG_LEVEL) - the bug
// this test exists to catch is the warning depending on the logger being
// enabled. It sets an environment variable, so it cannot run in parallel.
func TestDispatcher_WarnsToStderrWithoutLogger(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "")

	var errBuf bytes.Buffer
	disp, _ := dispatchCapture(Deps{ErrOut: &errBuf})

	_ = disp.Dispatch("session-start", newSessionPayload(t))

	if !strings.Contains(errBuf.String(), "reinstall") {
		t.Errorf("expected a stderr reinstall warning with no logger, got:\n%s", errBuf.String())
	}
}
