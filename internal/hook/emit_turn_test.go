package hook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestUserPromptNudge_ShortAndKeepsPhrases(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	body, err := fs.ReadFile(templates, "hooks/user-prompt.txt")
	if err != nil {
		t.Fatalf("read user-prompt.txt: %v", err)
	}
	text := strings.TrimSpace(string(body))

	const nudgeByteCeiling = 300
	if got := len(text); got > nudgeByteCeiling {
		t.Errorf(
			"user-prompt.txt is %d bytes, want <= %d",
			got,
			nudgeByteCeiling,
		)
	}
	if strings.ContainsRune(text, '\u2014') {
		t.Error("user-prompt.txt contains an em dash (U+2014)")
	}

	testCases := []struct {
		name string
		want string
	}{
		{"routing question kept", "Have you invoked the project-router skill"},
		{"go/sql/config trigger kept", "Go/SQL/config"},
		{"skip consequence kept", "Skipping the router skips the rules"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(text, testCase.want) {
				t.Errorf("user-prompt.txt missing %q:\n%s", testCase.want, text)
			}
		})
	}

	notWantedCases := []struct {
		name   string
		phrase string
	}{
		{"contract.txt's baseline skill list", "code-minimalism"},
		{"contract.txt/pre-edit.txt already state the test shape", "testCase"},
		{"pre-edit.txt's minimalism ladder", "ladder"},
		{"pre-edit.txt's stdlib rung", "slices/maps/cmp"},
		{"pre-edit.txt's named-domain-type rule", "minor-unit int64"},
	}
	for _, notWantedCase := range notWantedCases {
		t.Run(notWantedCase.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(text, notWantedCase.phrase) {
				t.Errorf(
					"user-prompt.txt duplicates contract.txt/pre-edit.txt phrase %q",
					notWantedCase.phrase,
				)
			}
		})
	}
}

func TestDispatchUserPrompt_EveryDispatch(t *testing.T) {
	t.Parallel()

	payload := newSessionPayload(t)
	disp, out := dispatchCapture(Deps{})

	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Errorf("second call did not fire: %q", out.String())
	}
}

func TestDispatchPreSearch(t *testing.T) {
	t.Parallel()

	const goGlob = `"pattern":"foo","glob":"*.go"`
	ready := preSearchFixture{hasGoMod: true, hasGoplsSkill: true}
	testCases := []struct {
		name      string
		toolName  string
		toolInput string
		fixture   preSearchFixture
		wantFire  bool
	}{
		{
			name: "grep on go glob fires", toolName: "Grep", toolInput: goGlob,
			fixture: ready, wantFire: true,
		},
		{
			name: "bash grep command fires", toolName: "Bash",
			toolInput: `"command":"grep -rn foo ."`, fixture: ready, wantFire: true,
		},
		{
			name: "grep on markdown stays silent", toolName: "Grep",
			toolInput: `"pattern":"foo","glob":"*.md"`, fixture: ready,
		},
		{
			name: "no go.mod stays silent", toolName: "Grep", toolInput: goGlob,
			fixture: preSearchFixture{hasGoplsSkill: true},
		},
		{
			name: "gopls-navigation not installed stays silent", toolName: "Grep",
			toolInput: goGlob, fixture: preSearchFixture{hasGoMod: true},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir, skillsDir := preSearchProjectDir(t, testCase.fixture)
			disp, out := dispatchCapture(Deps{ProjectDir: dir, SkillsDir: skillsDir})
			payload := fmt.Sprintf(
				`{"tool_name":%q,"tool_input":{%s},"session_id":%q}`,
				testCase.toolName, testCase.toolInput, uniqueSessionID(t),
			)
			_ = disp.Dispatch("pre-search", []byte(payload))
			if !testCase.wantFire {
				if strings.TrimSpace(out.String()) != "" {
					t.Errorf("expected no output, got %q", out.String())
				}
				return
			}
			ctx := extractContext(t, out.String())
			if !strings.Contains(ctx, "pre-search gopls reminder") {
				t.Errorf("missing reminder text:\n%s", ctx)
			}
		})
	}
}

type preSearchFixture struct {
	hasGoMod      bool
	hasGoplsSkill bool
}

func preSearchProjectDir(t *testing.T, fixture preSearchFixture) (dir, skillsDir string) {
	t.Helper()
	dir = t.TempDir()
	if fixture.hasGoMod {
		if err := os.WriteFile(
			filepath.Join(dir, "go.mod"),
			[]byte("module x\n"),
			0o600,
		); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
	}
	skillsDir = filepath.Join(dir, ".claude", "skills")
	if fixture.hasGoplsSkill {
		installSkillFixture(t, skillsDir, "gopls-navigation")
	}
	return dir, skillsDir
}

func TestDispatchPreSearchOncePerSession(t *testing.T) {
	t.Parallel()

	dir, skillsDir := preSearchProjectDir(t, preSearchFixture{hasGoMod: true, hasGoplsSkill: true})
	id := uniqueSessionID(t)
	payload := fmt.Appendf(
		nil,
		`{"tool_name":"Grep","tool_input":{"pattern":"foo","glob":"*.go"},"session_id":%q}`,
		id,
	)
	disp, out := dispatchCapture(Deps{ProjectDir: dir, SkillsDir: skillsDir})

	_ = disp.Dispatch("pre-search", payload)
	if !strings.Contains(extractContext(t, out.String()), "pre-search gopls reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("pre-search", payload)
	if strings.Contains(out.String(), "pre-search gopls reminder") {
		t.Errorf("second call fired again: %q", out.String())
	}
}
