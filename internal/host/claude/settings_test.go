package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

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
			if err := WireSettings(claudeDir, testCase.settingsFile); err != nil {
				t.Fatalf("WireSettings: %v", err)
			}
			hooks := readHooks(t, claudeDir, testCase.settingsFile)
			if len(hooks["SessionStart"]) != 1 || len(hooks["PreToolUse"]) != 2 {
				t.Fatalf("want one SessionStart and two PreToolUse groups, got %+v", hooks)
			}
		})
	}
}

func wireTwice(tb testing.TB, claudeDir, settingsFile string) {
	tb.Helper()

	for range 2 {
		if err := WireSettings(claudeDir, settingsFile); err != nil {
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
	if len(hooks["PreToolUse"]) != 2 {
		t.Errorf("PreToolUse groups = %d, want 2", len(hooks["PreToolUse"]))
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

	if err := WireSettings(claudeDir, settingsFile); err != nil {
		t.Fatalf("WireSettings: %v", err)
	}

	hooks := readHooks(t, claudeDir, settingsFile)
	var thirdParty, ours int
	for _, group := range hooks["SessionStart"] {
		for _, cmd := range group.Hooks {
			switch {
			case strings.Contains(cmd.Command, "third-party-tool.sh"):
				thirdParty++
			case strings.Contains(cmd.Command, reminderScriptAsset):
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

func userStopSeed(timeout string) string {
	return `{"hooks":{"Stop":[{"hooks":[` +
		`{"type":"command","command":"bash third-party.sh","timeout":` + timeout + `},` +
		`{"type":"command","command":"bash ` + reminderScriptAsset + ` stop"}]}]}}`
}

type timeoutCase struct {
	name    string
	timeout string
}

func userTimeoutCases() []timeoutCase {
	return []timeoutCase{
		{name: "fractional", timeout: "30.5"},
		{name: "integer", timeout: "30"},
		{name: "exponent", timeout: "1e2"},
		{name: "text", timeout: `"abc"`},
		{name: "numeric text", timeout: `"30"`},
		{name: "boolean", timeout: "true"},
		{name: "null", timeout: "null"},
	}
}

func userStopTimeout(tb testing.TB, claudeDir string) string {
	tb.Helper()
	for _, group := range readHooks(tb, claudeDir, "settings.json")["Stop"] {
		for _, command := range group.Hooks {
			if strings.Contains(command.Command, "third-party.sh") {
				return string(command.Timeout)
			}
		}
	}
	tb.Fatal("user Stop hook lost")
	return ""
}

func TestWireSettings_KeepsAnyUserHookTimeoutValue(t *testing.T) {
	t.Parallel()

	testCases := userTimeoutCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			claudeDir := t.TempDir()
			seed := userStopSeed(testCase.timeout)
			path := filepath.Join(claudeDir, "settings.json")
			if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if err := WireSettings(claudeDir, "settings.json"); err != nil {
				t.Fatalf("WireSettings must accept any user timeout: %v", err)
			}
			if got := userStopTimeout(t, claudeDir); got != testCase.timeout {
				t.Errorf("user timeout = %q, want %q", got, testCase.timeout)
			}
		})
	}
}

func TestWireSettings_Unparseable(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	path := filepath.Join(claudeDir, "settings.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := WireSettings(claudeDir, "settings.json")
	if !errors.Is(err, jsondoc.ErrUnparseable) {
		t.Fatalf("err = %v, want jsondoc.ErrUnparseable", err)
	}
	got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != original {
		t.Errorf("file was rewritten: got %q, want unchanged %q", got, original)
	}
}
