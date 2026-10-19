package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func hookTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/contract.txt":        {Data: []byte("contract text\n")},
		"hooks/contract-tail.txt":   {Data: []byte("commit tail text\n")},
		"hooks/pre-edit.txt":        {Data: []byte("pre-edit reminder text\n")},
		"hooks/post-edit.txt":       {Data: []byte("post-edit generic reminder\n")},
		"hooks/post-edit-other.txt": {Data: []byte("post-edit non-go reminder\n")},
		"hooks/post-edit-checklist.txt.tmpl": {Data: []byte(
			"checklist for {{.BaseName}}:\n" +
				"{{range .SkillIDs}}  - {{.}}\n{{end -}}" +
				"{{if .IsGo}}  - go doc-comment reminder\n{{end -}}" +
				"verify suite",
		)},
		"hooks/pre-search.txt":  {Data: []byte("pre-search gopls reminder\n")},
		"hooks/user-prompt.txt": {Data: []byte("user-prompt routing reminder\n")},
		"hooks/stop.txt":        {Data: []byte("Governance findings on the working tree.\n")},
		"hooks/stop-checklist.txt.tmpl": {Data: []byte(
			"{{if .Go}}- gofmt, go vet and go test green\n{{end}}{{if .SQL}}- sql bullet\n{{end}}" +
				"{{if .YAML}}- yaml bullet\n{{end}}{{if .Markdown}}- md bullet\n{{end}}",
		)},
	}
}

func dispatchCapture(deps Deps) (*Dispatcher, *bytes.Buffer) {
	var out bytes.Buffer
	deps.Out = &out
	if deps.Templates == (templates.Set{}) {
		deps.Templates = templates.New(hookTemplates())
	}
	if deps.ErrOut == nil {
		deps.ErrOut = io.Discard
	}
	return NewDispatcher(deps), &out
}

func extractContext(t *testing.T, raw string) string {
	t.Helper()
	var out hookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		t.Fatalf("unmarshal hook output: %v (raw=%q)", err, raw)
	}
	return out.HookSpecificOutput.AdditionalContext
}

func uniqueSessionID(t *testing.T) string {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	id := fmt.Sprintf("test-%d-%s", os.Getpid(), name)
	newSessionTracker(id).Cleanup()
	t.Cleanup(func() { newSessionTracker(id).Cleanup() })
	return id
}

func newSessionPayload(t *testing.T) []byte {
	t.Helper()
	return fmt.Appendf(nil, `{"session_id":%q}`, uniqueSessionID(t))
}

func installSkillFixture(t *testing.T, skillsDir, id string) {
	t.Helper()
	dir := filepath.Join(skillsDir, id)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("stub\n"), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

func TestDispatchSessionStart(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{})
	_ = disp.Dispatch("session-start", newSessionPayload(t))
	ctx := extractContext(t, out.String())
	for _, want := range []string{"contract text", "commit tail text"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("missing %q:\n%s", want, ctx)
		}
	}
}

func TestDispatchSessionEnd_Markers(t *testing.T) {
	t.Parallel()

	id := uniqueSessionID(t)
	session := fmt.Appendf(nil, `{"session_id":%q}`, id)
	disp, _ := dispatchCapture(Deps{Index: preEditIndex(t)})

	_ = disp.Dispatch("session-start", session)
	_ = disp.Dispatch(
		"pre-edit",
		fmt.Appendf(nil, `{"tool_input":{"file_path":"a.go"},"session_id":%q}`, id),
	)

	markers := []string{
		"skill-route-session-start." + id,
		"skill-hook-" + preEditFileEvent("a.go") + "." + id,
	}
	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); err != nil {
			t.Errorf("marker %q not created before session-end: %v", name, err)
		}
	}

	_ = disp.Dispatch("session-end", session)

	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("marker %q not removed after session-end (stat err = %v)", name, err)
		}
	}
}

func TestDispatchSubagentStart_InheritedSession(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{})
	payload := newSessionPayload(t)

	_ = disp.Dispatch("subagent-start", payload)
	first := extractContext(t, out.String())
	if first == "" {
		t.Fatalf("expected subagent-start to emit the contract, got empty output")
	}
	if strings.Contains(first, "commit tail text") {
		t.Errorf("subagent-start must omit the commit tail (a subagent never commits):\n%s", first)
	}

	out.Reset()
	_ = disp.Dispatch("subagent-start", payload)
	second := extractContext(t, out.String())
	if second != first {
		t.Errorf(
			"second dispatch with the same session id = %q, want %q (re-emitted, not suppressed)",
			second, first,
		)
	}
}

func TestDispatchSubagentStart_MissingContract(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, out := dispatchCapture(Deps{Logger: logger, Templates: templates.New(fstest.MapFS{})})

	if err := disp.Dispatch("subagent-start", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read hooks/contract.txt") {
		t.Errorf("log missing contract.txt read failure:\n%s", logBuf.String())
	}
}

func TestDispatchContract_SessionAndSubagent(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	wantSession, err := contract.Session(templates)
	if err != nil {
		t.Fatalf("contract.Session: %v", err)
	}
	wantSubagent, err := contract.Subagent(templates)
	if err != nil {
		t.Fatalf("contract.Subagent: %v", err)
	}

	testCases := []struct {
		name string
		mode string
		want string
	}{
		{name: "session-start", mode: "session-start", want: wantSession},
		{name: "subagent-start", mode: "subagent-start", want: wantSubagent},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			disp, out := dispatchCapture(Deps{Templates: templates})
			_ = disp.Dispatch(testCase.mode, newSessionPayload(t))
			got := extractContext(t, out.String())
			if got != testCase.want {
				t.Errorf("%s emitted %q, want %q", testCase.mode, got, testCase.want)
			}
		})
	}
}

func stubGovernance(block string, count int) func(vcs.Repository, vcs.Status) (string, int) {
	return func(vcs.Repository, vcs.Status) (string, int) {
		return block, count
	}
}

func TestRecord_LogsTheDispatchAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		entry       dispatchRecord
		wantOK      bool
		wantFinding float64
		wantTimeout bool
		wantError   string
	}{
		{
			name:        "clean dispatch",
			entry:       dispatchRecord{mode: "pre-edit"},
			wantOK:      true,
			wantFinding: 0,
		},
		{
			name: "stop with findings that timed out",
			entry: dispatchRecord{
				mode:    "stop",
				outcome: stopOutcome{findings: 3, timedOut: true},
			},
			wantOK:      true,
			wantFinding: 3,
			wantTimeout: true,
		},
		{
			name:      "failed handler",
			entry:     dispatchRecord{mode: "user-prompt", err: errors.New("boom")},
			wantError: "boom",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
			disp, _ := dispatchCapture(Deps{Logger: logger})
			session := newSessionTracker("record-session")
			testCase.entry.start = time.Now()

			disp.record(session, testCase.entry)

			var got map[string]any
			if err := json.Unmarshal(logBuf.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal log line %q: %v", logBuf.String(), err)
			}
			if got["event"] != testCase.entry.mode {
				t.Errorf("event = %v, want %q", got["event"], testCase.entry.mode)
			}
			if got["session_id"] != "record-session" {
				t.Errorf("session_id = %v, want record-session", got["session_id"])
			}
			if got["ok"] != testCase.wantOK {
				t.Errorf("ok = %v, want %v", got["ok"], testCase.wantOK)
			}
			if got["findings"] != testCase.wantFinding {
				t.Errorf("findings = %v, want %v", got["findings"], testCase.wantFinding)
			}
			if timedOut, _ := got["timed_out"].(bool); timedOut != testCase.wantTimeout {
				t.Errorf("timed_out = %v, want %v", timedOut, testCase.wantTimeout)
			}
			if gotError, _ := got["error"].(string); gotError != testCase.wantError {
				t.Errorf("error = %q, want %q", gotError, testCase.wantError)
			}
		})
	}
}

func TestDispatchUnknownMode_WritesNothing(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{})

	if err := disp.Dispatch("bogus", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("unknown mode wrote %q, want no output", out.String())
	}
}

func TestDispatchPostEdit_ExplicitSkillsDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		filePath     string
		installSkill string
		wantContains string
	}{
		{
			name:         "go file emits the skill when it is installed",
			filePath:     "pkg/foo.go",
			installSkill: "go-style",
			wantContains: "go-style",
		},
		{
			name:         "go file falls back to generic when no skills installed",
			filePath:     "pkg/foo.go",
			wantContains: "post-edit generic reminder",
		},
		{
			name:         "markdown extension produces no output",
			filePath:     "README.md",
			wantContains: "",
		},
		{
			name:         "sql file emits the skill when it is installed",
			filePath:     "query.sql",
			installSkill: "database-persistence",
			wantContains: "database-persistence",
		},
		{
			name:         "test file emits the skill when it is installed",
			filePath:     "internal/service/service_test.go",
			installSkill: "integration-tests",
			wantContains: "integration-tests",
		},
		{
			name:         "proto file emits the skill when it is installed",
			filePath:     "api.proto",
			installSkill: "grpc-adapter",
			wantContains: "grpc-adapter",
		},
	}
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			skillsDir := t.TempDir()
			if testCase.installSkill != "" {
				installSkillFixture(t, skillsDir, testCase.installSkill)
			}
			disp, out := dispatchCapture(Deps{Index: idx, SkillsDir: skillsDir})
			payload := postEditPayload(t, testCase.filePath, uniqueSessionID(t))
			_ = disp.Dispatch("post-edit", payload)
			if testCase.wantContains == "" {
				if strings.TrimSpace(out.String()) != "" {
					t.Errorf("expected no output for %q, got %q", testCase.filePath, out.String())
				}
				return
			}
			ctx := extractContext(t, out.String())
			if !strings.Contains(ctx, testCase.wantContains) {
				t.Errorf("output missing %q:\n%s", testCase.wantContains, ctx)
			}
		})
	}
}

func TestDispatchPostEdit_NonGoFallback(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	disp, out := dispatchCapture(Deps{Index: idx, SkillsDir: t.TempDir()})
	_ = disp.Dispatch("post-edit", postEditPayload(t, "query.sql", uniqueSessionID(t)))
	ctx := extractContext(t, out.String())
	if !strings.Contains(ctx, "post-edit non-go reminder") {
		t.Errorf("expected the non-Go reminder, got %q", ctx)
	}
}

func TestDispatchPostEdit_SilentForTSAndJS(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	for _, id := range []string{"dart-style", "ts-errors", "domain-hygiene", "code-minimalism"} {
		installSkillFixture(t, skillsDir, id)
	}
	for _, filePath := range []string{"src/x.ts", "src/X.tsx", "src/x.js"} {
		t.Run(filePath, func(t *testing.T) {
			t.Parallel()
			disp, out := dispatchCapture(Deps{Index: preEditIndex(t), SkillsDir: skillsDir})
			_ = disp.Dispatch("post-edit", postEditPayload(t, filePath, uniqueSessionID(t)))
			if strings.TrimSpace(out.String()) != "" {
				t.Errorf("post-edit %s: output %q, want silence", filePath, out.String())
			}
		})
	}
}

func TestDispatchPostEdit_SkillsDirUnset(t *testing.T) {
	t.Setenv("PULSARULES_SKILLS_DIR", "")

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	disp, out := dispatchCapture(Deps{Index: idx})
	_ = disp.Dispatch("post-edit", postEditPayload(t, "pkg/foo.go", uniqueSessionID(t)))
	ctx := extractContext(t, out.String())
	if !strings.Contains(ctx, "post-edit generic reminder") {
		t.Errorf("expected the generic reminder, got %q", ctx)
	}
}
