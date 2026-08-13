package hookwire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRenderHooksBlock(t *testing.T) {
	t.Parallel()

	block, err := RenderHooksBlock(fakeTemplates())
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var parsed hooksBlock
	if err = json.Unmarshal(block, &parsed); err != nil {
		t.Fatalf("rendered block is not valid JSON: %v", err)
	}
	got := parsed.Hooks["SessionStart"][0].Hooks[0].Command
	want := `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh" session-start`
	if got != want {
		t.Errorf("session-start command = %q, want %q", got, want)
	}
}

func TestRenderHooksBlock_SubagentStart(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}
	block, err := RenderHooksBlock(templates)
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var parsed hooksBlock
	if err = json.Unmarshal(block, &parsed); err != nil {
		t.Fatalf("rendered block is not valid JSON: %v", err)
	}
	groups := parsed.Hooks["SubagentStart"]
	if len(groups) != 1 || len(groups[0].Hooks) != 1 {
		t.Fatalf("expected exactly one SubagentStart hook group, got %+v", groups)
	}
	got := groups[0].Hooks[0].Command
	want := `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh" subagent-start`
	if got != want {
		t.Errorf("subagent-start command = %q, want %q", got, want)
	}
}

func TestWireSettings_FreshFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		settingsFile string
	}{
		{"project scope", "settings.json"},
		{"local scope", "settings.local.json"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			claudeDir := filepath.Join(t.TempDir(), ".claude")
			if err := WireSettings(fakeTemplates(), claudeDir, testCase.settingsFile); err != nil {
				t.Fatalf("WireSettings: %v", err)
			}
			hooks := readHooks(t, claudeDir, testCase.settingsFile)
			if len(hooks["SessionStart"]) != 1 || len(hooks["PreToolUse"]) != 1 {
				t.Fatalf("expected one group per event, got %+v", hooks)
			}
		})
	}
}

func wireTwice(tb testing.TB, claudeDir, settingsFile string) {
	tb.Helper()

	for range 2 {
		if err := WireSettings(fakeTemplates(), claudeDir, settingsFile); err != nil {
			tb.Fatalf("WireSettings: %v", err)
		}
	}
}

func readTopLevel(tb testing.TB, path string) map[string]json.RawMessage {
	tb.Helper()

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read: %v", err)
	}
	var settings map[string]json.RawMessage
	if err = json.Unmarshal(raw, &settings); err != nil {
		tb.Fatalf("parse: %v", err)
	}
	return settings
}

func TestWireSettings_PreservesNotificationHook(t *testing.T) {
	t.Parallel()

	const settingsFile = "settings.json"
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	seed := `{
  "permissions": {"allow": ["Bash(go test *)"]},
  "enabledMcpjsonServers": ["example"],
  "hooks": {
    "Notification": [{"hooks": [{"type": "command", "command": "echo done"}]}]
  }
}`
	settingsPath := filepath.Join(claudeDir, settingsFile)
	seedFile(t, settingsPath, seed)

	wireTwice(t, claudeDir, settingsFile)

	settings := readTopLevel(t, settingsPath)
	for _, key := range []string{"permissions", "enabledMcpjsonServers"} {
		if _, ok := settings[key]; !ok {
			t.Errorf("lost top-level key %q", key)
		}
	}

	hooks := readHooks(t, claudeDir, settingsFile)
	if len(hooks["SessionStart"]) != 1 {
		t.Errorf("SessionStart groups = %d, want 1", len(hooks["SessionStart"]))
	}
	if len(hooks["PreToolUse"]) != 1 {
		t.Errorf("PreToolUse groups = %d, want 1", len(hooks["PreToolUse"]))
	}
	cmd := hooks["SessionStart"][0].Hooks[0].Command
	if !strings.Contains(cmd, `$CLAUDE_PROJECT_DIR`) {
		t.Errorf("expected $CLAUDE_PROJECT_DIR command, got %q", cmd)
	}
	if len(hooks["Notification"]) != 1 {
		t.Errorf("lost unrelated Notification hook, got %+v", hooks["Notification"])
	}
}

func TestWireSettings_ThirdPartyCommand(t *testing.T) {
	t.Parallel()

	const settingsFile = "settings.json"
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	seed := `{
  "hooks": {
    "SessionStart": [{"hooks": [
      {"type": "command", "command": "bash .claude/hooks/skill-router-reminder.sh session-start"},
      {"type": "command", "command": "bash third-party-tool.sh"}
    ]}]
  }
}`
	if err := os.WriteFile(
		filepath.Join(claudeDir, settingsFile), []byte(seed), 0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WireSettings(fakeTemplates(), claudeDir, settingsFile); err != nil {
		t.Fatalf("WireSettings: %v", err)
	}

	hooks := readHooks(t, claudeDir, settingsFile)
	var thirdParty, ours int
	for _, group := range hooks["SessionStart"] {
		for _, cmd := range group.Hooks {
			switch {
			case strings.Contains(cmd.Command, "third-party-tool.sh"):
				thirdParty++
			case strings.Contains(cmd.Command, hookScript):
				ours++
			}
		}
	}
	if thirdParty != 1 {
		t.Errorf("third-party command count = %d, want 1 (it was deleted)", thirdParty)
	}
	if ours != 1 {
		t.Errorf("our command count = %d, want exactly 1 (no duplication)", ours)
	}
}

func readHooks(tb testing.TB, claudeDir, settingsFile string) map[string][]hookGroup {
	tb.Helper()
	raw, err := os.ReadFile(
		filepath.Join(claudeDir, settingsFile),
	) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read settings: %v", err)
	}
	var settings struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}
	if err = json.Unmarshal(raw, &settings); err != nil {
		tb.Fatalf("parse settings: %v", err)
	}
	return settings.Hooks
}
