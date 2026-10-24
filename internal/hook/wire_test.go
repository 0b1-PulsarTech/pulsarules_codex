package hook

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/dialect"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type hookOutput struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

func TestDecodeHookPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		payload      string
		wantSession  string
		wantFilePath string
	}{
		{
			name:         "well formed payload",
			payload:      `{"session_id":"s1","tool_input":{"file_path":"main.go"}}`,
			wantSession:  "s1",
			wantFilePath: "main.go",
		},
		{
			name:        "session without tool input",
			payload:     `{"session_id":"s2"}`,
			wantSession: "s2",
		},
		{
			name:    "malformed payload degrades to the zero value",
			payload: `{not json`,
		},
		{
			name:    "empty payload degrades to the zero value",
			payload: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			decoded := decodeHookPayload([]byte(testCase.payload))
			if decoded.SessionID != testCase.wantSession {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, testCase.wantSession)
			}
			if decoded.ToolInput.FilePath != testCase.wantFilePath {
				t.Errorf(
					"FilePath = %q, want %q",
					decoded.ToolInput.FilePath,
					testCase.wantFilePath,
				)
			}
		})
	}
}

func TestDecodeHookPayloads(t *testing.T) {
	t.Parallel()

	const patch = `*** Begin Patch\n*** Update File: a.go\n@@\n-x\n+y\n*** Add File: /abs/b.go\n+z\n*** End Patch`
	testCases := []struct {
		name      string
		dialect   string
		payload   string
		wantPaths []string
	}{
		{
			"codex two file patch gives two absolute payloads",
			dialect.Codex,
			`{"session_id":"s","cwd":"/work","tool_name":"apply_patch","tool_input":{"command":"` + patch + `"}}`,
			[]string{"/work/a.go", "/abs/b.go"},
		},
		{
			"codex Edit tool name is treated as an edit", dialect.Codex,
			`{"cwd":"/work","tool_name":"Edit","tool_input":{"command":"*** Update File: c.go\n"}}`,
			[]string{"/work/c.go"},
		},
		{
			"codex Write tool name is treated as an edit", dialect.Codex,
			`{"cwd":"/work","tool_name":"Write","tool_input":{"command":"*** Add File: w.go\n"}}`,
			[]string{"/work/w.go"},
		},
		{
			"codex MultiEdit tool name is treated as an edit",
			dialect.Codex,
			`{"cwd":"/work","tool_name":"MultiEdit","tool_input":{"command":"*** Update File: m.go\n"}}`,
			[]string{"/work/m.go"},
		},
		{
			"codex bash passes through", dialect.Codex,
			`{"cwd":"/work","tool_name":"Bash","tool_input":{"command":"grep -r foo ."}}`,
			[]string{""},
		},
		{
			"codex patch without headers passes through", dialect.Codex,
			`{"tool_name":"apply_patch","tool_input":{"command":"junk"}}`,
			[]string{""},
		},
		{
			"claude ignores patch text", "",
			`{"tool_name":"apply_patch","tool_input":{"command":"*** Update File: /x.go\n"}}`,
			[]string{""},
		},
		{
			"claude file_path is kept",
			dialect.Codex,
			`{"tool_name":"Write","tool_input":{"file_path":"/w/x.go","command":"*** Update File: /y.go\n"}}`,
			[]string{"/w/x.go"},
		},
		{"malformed payload", dialect.Codex, `{not json`, []string{""}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, in := range decodeHookPayloads([]byte(testCase.payload), testCase.dialect) {
				got = append(got, in.ToolInput.FilePath)
			}
			if !slices.Equal(got, testCase.wantPaths) {
				t.Errorf("file paths = %q, want %q", got, testCase.wantPaths)
			}
		})
	}
}

func decodeOne(t *testing.T, raw string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("not exactly one JSON object: %v (%q)", err, raw)
	}
	return decoded
}

func TestDispatch_CodexPostEditFansOutToOneObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skills := t.TempDir()
	installSkillFixture(t, skills, "go-style")
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	disp, out := dispatchCapture(
		Deps{Index: idx, ProjectDir: dir, SkillsDir: skills, Dialect: dialect.Codex},
	)
	payload := `{"session_id":"` + uniqueSessionID(
		t,
	) + `","cwd":"` + dir + `","tool_name":"apply_patch",` +
		`"tool_input":{"command":"*** Update File: a.go\n*** Update File: b.go\n"}}`

	_ = disp.Dispatch("post-edit", []byte(payload))

	decodeOne(t, out.String())
	if lines := strings.Count(strings.TrimSpace(out.String()), "\n"); lines != 0 {
		t.Errorf("output has %d extra lines: %q", lines, out.String())
	}
	reminder := extractContext(t, out.String())
	for _, want := range []string{"checklist for a.go", "checklist for b.go"} {
		if !strings.Contains(reminder, want) {
			t.Errorf("missing %q in %q", want, reminder)
		}
	}
}

func patchPayload(t *testing.T, session, cwd, patch string) []byte {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{
		"session_id": session, "cwd": cwd, "tool_name": "apply_patch",
		"tool_input": map[string]string{"command": patch},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return encoded
}

func codexPatchDispatcher(t *testing.T, dir string) (*Dispatcher, *bytes.Buffer) {
	t.Helper()
	skills := t.TempDir()
	installSkillFixture(t, skills, "go-style")
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return dispatchCapture(
		Deps{Index: idx, ProjectDir: dir, SkillsDir: skills, Dialect: dialect.Codex},
	)
}

func TestDispatch_CodexPatchOps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		mode  string
		patch string
		want  []string
		deny  []string
	}{
		{
			"pre-edit routes only file 2", "pre-edit",
			"*** Update File: a.txt\n*** Update File: b.go\n",
			[]string{"pre-edit reminder text"},
			nil,
		},
		{
			"pre-edit skips a delete", "pre-edit",
			"*** Delete File: a.go\n", nil,
			[]string{"pre-edit reminder text"},
		},
		{
			"pre-edit keeps a move source", "pre-edit",
			"*** Update File: a.go\n*** Move to: b.txt\n",
			[]string{"pre-edit reminder text"},
			nil,
		},
		{
			"post-edit skips a delete", "post-edit",
			"*** Delete File: a.go\n*** Update File: b.go\n",
			[]string{"checklist for b.go"},
			[]string{"checklist for a.go"},
		},
		{
			"post-edit skips a move source and keeps the destination", "post-edit",
			"*** Update File: a.go\n*** Move to: b.go\n",
			[]string{"checklist for b.go"},
			[]string{"checklist for a.go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			disp, out := codexPatchDispatcher(t, dir)

			_ = disp.Dispatch(
				testCase.mode,
				patchPayload(t, uniqueSessionID(t), dir, testCase.patch),
			)

			got := ""
			if out.Len() != 0 {
				got = extractContext(t, out.String())
			}
			for _, want := range testCase.want {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %q", want, got)
				}
			}
			for _, deny := range testCase.deny {
				if strings.Contains(got, deny) {
					t.Errorf("unexpected %q in %q", deny, got)
				}
			}
		})
	}
}

func TestDispatch_CodexDeleteKeepsThePreEditMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	disp, out := codexPatchDispatcher(t, dir)
	session := uniqueSessionID(t)
	send := func(patch string) {
		_ = disp.Dispatch("pre-edit", patchPayload(t, session, dir, patch))
	}

	send("*** Delete File: a.go\n")
	send("*** Update File: a.go\n")

	if !strings.Contains(extractContext(t, out.String()), "pre-edit reminder text") {
		t.Errorf("the update after a delete must still be reminded: %q", out.String())
	}
}

func TestDispatch_CodexFanOutKeepsEveryFilesError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skills := t.TempDir()
	installSkillFixture(t, skills, "go-style")
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	assets := hookTemplates()
	delete(assets, "hooks/post-edit-checklist.txt.tmpl")
	var logBuf bytes.Buffer
	disp, _ := dispatchCapture(Deps{
		Index: idx, ProjectDir: dir, SkillsDir: skills, Dialect: dialect.Codex,
		Logger:    slog.New(slog.NewTextHandler(&logBuf, nil)),
		Templates: templates.New(assets),
	})
	patch := "*** Update File: a.go\n*** Update File: b.txt\n*** Update File: c.go\n"

	_ = disp.Dispatch("post-edit", patchPayload(t, uniqueSessionID(t), dir, patch))

	if got := strings.Count(logBuf.String(), "file does not exist"); got != 2 {
		t.Errorf(
			"logged %d checklist errors, want one per failing file (2):\n%s",
			got,
			logBuf.String(),
		)
	}
}

func TestDispatch_CodexPreSearchExecCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	skills := t.TempDir()
	installSkillFixture(t, skills, "gopls-navigation")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	disp, out := dispatchCapture(Deps{ProjectDir: dir, SkillsDir: skills, Dialect: dialect.Codex})
	payload := `{"session_id":"` + uniqueSessionID(
		t,
	) + `","tool_name":"exec_command","tool_input":{"command":"grep -r foo ."}}`

	_ = disp.Dispatch("pre-search", []byte(payload))

	if !strings.Contains(extractContext(t, out.String()), "pre-search gopls reminder") {
		t.Errorf("pre-search did not fire: %q", out.String())
	}
}

func TestDispatch_CodexStopShape(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		governance string
		wantBlock  bool
	}{
		{"findings block with a reason", "\nGovernance checks:\n  - stub\n", true},
		{"no findings prints nothing", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			gitInit(t, dir)
			seedFiles(t, dir, 1)
			disp, out := dispatchCapture(Deps{
				ProjectDir: dir, Dialect: dialect.Codex,
				Governance: stubGovernance(testCase.governance, 1),
			})

			_ = disp.Dispatch("stop", newSessionPayload(t))

			if !testCase.wantBlock {
				if out.Len() != 0 {
					t.Fatalf("want nothing, got %q", out.String())
				}
				return
			}
			decoded := decodeOne(t, out.String())
			if decoded["decision"] != "block" || decoded["reason"] == "" {
				t.Errorf("want a block decision with a reason, got %v", decoded)
			}
			if _, has := decoded["hookSpecificOutput"]; has {
				t.Errorf("Stop must not carry hookSpecificOutput: %v", decoded)
			}
		})
	}
}

func TestDispatch_ClaudeStopKeepsAdditionalContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	seedFiles(t, dir, 1)
	disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance("\nblock\n", 1)})

	_ = disp.Dispatch("stop", newSessionPayload(t))

	var emitted hookOutput
	if err := json.Unmarshal(
		out.Bytes(),
		&emitted,
	); err != nil ||
		emitted.HookSpecificOutput.HookEventName != "Stop" {
		t.Errorf("Claude Stop shape changed: %v %q", err, out.String())
	}
}

func TestDispatch_SessionStartDriftAndContractAreOneObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "knowledge", "standards"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "knowledge", "standards", "skills.yaml"),
		[]byte("skills: []\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", dialect.Codex} {
		disp, out := dispatchCapture(Deps{ProjectDir: dir, Dialect: name})

		_ = disp.Dispatch("session-start", newSessionPayload(t))

		decodeOne(t, out.String())
		ctx := extractContext(t, out.String())
		if !strings.Contains(ctx, "contract text") || !strings.Contains(ctx, "behind the source") {
			t.Errorf("dialect %q: contract and drift not both in one object: %q", name, ctx)
		}
	}
}
