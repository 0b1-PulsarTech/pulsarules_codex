package hookwire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestRenderHooksBlock asserts the portable $CLAUDE_PROJECT_DIR hook command is
// rendered and the result is valid JSON.
func TestRenderHooksBlock(t *testing.T) {
	t.Parallel()

	block, err := RenderHooksBlock(fakeTemplates())
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var parsed hooksBlock
	if err := json.Unmarshal(block, &parsed); err != nil {
		t.Fatalf("rendered block is not valid JSON: %v", err)
	}
	got := parsed.Hooks["SessionStart"][0].Hooks[0].Command
	want := `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh" session-start`
	if got != want {
		t.Errorf("session-start command = %q, want %q", got, want)
	}
}

// TestRenderHooksBlock_SubagentStart renders the REAL embedded settings
// template (not the fakeTemplates fixture, which is a deliberately minimal
// stand-in), so a future edit to settings.hooks.json.tmpl that drops
// SubagentStart fails this test instead of silently regressing every
// spawned subagent back to zero injected contract.
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
	if err := json.Unmarshal(block, &parsed); err != nil {
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

// TestWireSettings_FreshFile asserts each scope creates its own settings file
// with both events wired when none exists.
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

// TestWireSettings_Idempotent asserts re-running never duplicates the hook
// entries and preserves unrelated settings + hooks.
func TestWireSettings_Idempotent(t *testing.T) {
	t.Parallel()

	const settingsFile = "settings.json"
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Seed unrelated permissions + an unrelated Notification hook that must survive.
	// (Stop is now owned by the router block, so we use a different event here.)
	seed := `{
  "permissions": {"allow": ["Bash(go test *)"]},
  "enabledMcpjsonServers": ["example"],
  "hooks": {
    "Notification": [{"hooks": [{"type": "command", "command": "echo done"}]}]
  }
}`
	settingsPath := filepath.Join(claudeDir, settingsFile)
	if err := os.WriteFile(settingsPath, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WireSettings(fakeTemplates(), claudeDir, settingsFile); err != nil {
		t.Fatalf("WireSettings #1: %v", err)
	}
	if err := WireSettings(fakeTemplates(), claudeDir, settingsFile); err != nil {
		t.Fatalf("WireSettings #2: %v", err)
	}

	// Unrelated top-level keys preserved.
	var settings map[string]json.RawMessage
	raw, err := os.ReadFile(settingsPath) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, key := range []string{"permissions", "enabledMcpjsonServers"} {
		if _, ok := settings[key]; !ok {
			t.Errorf("lost top-level key %q", key)
		}
	}

	hooks := readHooks(t, claudeDir, settingsFile)
	// No duplication: exactly one router group per touched event.
	if len(hooks["SessionStart"]) != 1 {
		t.Errorf("SessionStart groups = %d, want 1", len(hooks["SessionStart"]))
	}
	if len(hooks["PreToolUse"]) != 1 {
		t.Errorf("PreToolUse groups = %d, want 1", len(hooks["PreToolUse"]))
	}
	// The wired command is the portable $CLAUDE_PROJECT_DIR form.
	cmd := hooks["SessionStart"][0].Hooks[0].Command
	if !strings.Contains(cmd, `$CLAUDE_PROJECT_DIR`) {
		t.Errorf("expected $CLAUDE_PROJECT_DIR command, got %q", cmd)
	}
	// Unrelated Notification hook preserved.
	if len(hooks["Notification"]) != 1 {
		t.Errorf("lost unrelated Notification hook, got %+v", hooks["Notification"])
	}
}

func readHooks(t testing.TB, claudeDir, settingsFile string) map[string][]hookGroup {
	t.Helper()
	raw, err := os.ReadFile(
		filepath.Join(claudeDir, settingsFile),
	) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var settings struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	return settings.Hooks
}
