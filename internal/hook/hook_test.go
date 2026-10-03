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

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func hookTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/contract.txt":      {Data: []byte("contract text\n")},
		"hooks/contract-tail.txt": {Data: []byte("commit tail text\n")},
		"hooks/pre-edit.txt":      {Data: []byte("pre-edit reminder text\n")},
		"hooks/post-edit.txt":     {Data: []byte("post-edit generic reminder\n")},
		"hooks/post-edit-checklist.txt.tmpl": {Data: []byte(
			"checklist for {{.BaseName}}:\n" +
				"{{range .SkillIDs}}  - {{.}}\n{{end -}}" +
				"{{if .IsGo}}  - go doc-comment reminder\n{{end -}}" +
				"verify suite",
		)},
		"hooks/pre-search.txt":  {Data: []byte("pre-search gopls reminder\n")},
		"hooks/user-prompt.txt": {Data: []byte("user-prompt routing reminder\n")},
		"hooks/stop.txt":        {Data: []byte("Governance findings on the working tree.\n")},
	}
}

func dispatchCapture(deps Deps) (*Dispatcher, *bytes.Buffer) {
	var out bytes.Buffer
	deps.Out = &out
	if deps.Templates == nil {
		deps.Templates = hookTemplates()
	}
	if deps.ErrOut == nil {
		deps.ErrOut = io.Discard
	}
	return NewDispatcher(deps), &out
}

func extractContext(t *testing.T, raw string) string {
	t.Helper()
	var out Output
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
	disp, out := dispatchCapture(Deps{Logger: logger, Templates: fstest.MapFS{}})

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
