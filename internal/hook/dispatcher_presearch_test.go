package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchPreSearch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		toolName      string
		toolInput     string
		hasGoMod      bool
		hasGoplsSkill bool
		wantFire      bool
	}{
		{
			name:          "grep on go glob fires",
			toolName:      "Grep",
			toolInput:     `"pattern":"foo","glob":"*.go"`,
			hasGoMod:      true,
			hasGoplsSkill: true,
			wantFire:      true,
		},
		{
			name:          "bash grep command fires",
			toolName:      "Bash",
			toolInput:     `"command":"grep -rn foo ."`,
			hasGoMod:      true,
			hasGoplsSkill: true,
			wantFire:      true,
		},
		{
			name:          "grep on markdown stays silent",
			toolName:      "Grep",
			toolInput:     `"pattern":"foo","glob":"*.md"`,
			hasGoMod:      true,
			hasGoplsSkill: true,
			wantFire:      false,
		},
		{
			name:          "no go.mod stays silent",
			toolName:      "Grep",
			toolInput:     `"pattern":"foo","glob":"*.go"`,
			hasGoMod:      false,
			hasGoplsSkill: true,
			wantFire:      false,
		},
		{
			name:          "gopls-navigation not installed stays silent",
			toolName:      "Grep",
			toolInput:     `"pattern":"foo","glob":"*.go"`,
			hasGoMod:      true,
			hasGoplsSkill: false,
			wantFire:      false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := preSearchProjectDir(t, testCase.hasGoMod, testCase.hasGoplsSkill)
			disp, out := dispatchCapture(Deps{ProjectDir: dir})
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

// preSearchProjectDir builds a fresh project dir carrying a go.mod (when
// hasGoMod) and an installed gopls-navigation skill (when hasGoplsSkill), so
// pre-search tests can exercise every gate combination.
func preSearchProjectDir(t *testing.T, hasGoMod, hasGoplsSkill bool) string {
	t.Helper()
	dir := t.TempDir()
	if hasGoMod {
		if err := os.WriteFile(
			filepath.Join(dir, "go.mod"),
			[]byte("module x\n"),
			0o600,
		); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
	}
	if hasGoplsSkill {
		skillDir := filepath.Join(dir, ".claude", "skills", "gopls-navigation")
		if err := os.MkdirAll(skillDir, 0o750); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		if err := os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte("stub\n"),
			0o600,
		); err != nil {
			t.Fatalf("write SKILL.md: %v", err)
		}
	}
	return dir
}

func TestDispatchPreSearchOncePerSession(t *testing.T) {
	t.Parallel()

	dir := preSearchProjectDir(t, true, true)
	id := uniqueSessionID(t)
	payload := fmt.Appendf(
		nil,
		`{"tool_name":"Grep","tool_input":{"pattern":"foo","glob":"*.go"},"session_id":%q}`,
		id,
	)
	disp, out := dispatchCapture(Deps{ProjectDir: dir})

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
