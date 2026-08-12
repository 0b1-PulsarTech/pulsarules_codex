package hookwire

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

// TestUnwireSettings_ThirdPartyCommandSurvives is the regression test for
// the data-loss bug this file fixes: withoutHookScript (the install-time
// filter) drops an entire hookGroup when any command references the hook
// script, safe there only because the group is re-appended. UnwireSettings
// never re-appends, so it must filter at the COMMAND level.
func TestUnwireSettings_ThirdPartyCommandSurvives(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	seed := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" session-start"},
          {"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/third-party-tool.sh\""}
        ]
      }
    ]
  }
}`
	path := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if !changed {
		t.Error("expected changed = true, our command was removed")
	}

	hooks := readHooks(t, claudeDir, "settings.json")
	groups := hooks["SessionStart"]
	if len(groups) != 1 {
		t.Fatalf("expected the group to survive (with our command dropped), got %+v", groups)
	}
	if len(groups[0].Hooks) != 1 {
		t.Fatalf("expected exactly the third-party command to remain, got %+v", groups[0].Hooks)
	}
	if !strings.Contains(groups[0].Hooks[0].Command, "third-party-tool.sh") {
		t.Errorf("third-party command lost: got %+v", groups[0].Hooks)
	}
	if strings.Contains(groups[0].Hooks[0].Command, hookScript) {
		t.Errorf("our command should have been removed, got %+v", groups[0].Hooks)
	}
}

// TestUnwireSettings_DropsEmptyContainers asserts a group left with no
// commands is dropped, an event left with no groups is dropped, and the file
// is deleted once "hooks" was the only thing in it.
func TestUnwireSettings_DropsEmptyContainers(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	if err := WireSettings(fakeTemplates(), claudeDir, "settings.json"); err != nil {
		t.Fatalf("seed via WireSettings: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if !changed {
		t.Error("expected changed = true, the wired settings were removed")
	}

	if _, statErr := os.Stat(
		filepath.Join(claudeDir, "settings.json"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("expected settings.json to be removed once empty, stat err = %v", statErr)
	}
}

// TestUnwireSettings_PreservesUnrelated asserts unrelated top-level keys and
// an unrelated hook event survive, and the file is kept (not deleted) because
// they remain in it.
func TestUnwireSettings_PreservesUnrelated(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	seed := `{
  "permissions": {"allow": ["Bash(go test *)"]},
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" session-start"}]}],
    "Notification": [{"hooks": [{"type": "command", "command": "echo done"}]}]
  }
}`
	path := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if !changed {
		t.Error("expected changed = true, SessionStart's wiring was removed")
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var settings map[string]json.RawMessage
	if err = json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := settings["permissions"]; !ok {
		t.Error("lost unrelated top-level key permissions")
	}
	hooks := readHooks(t, claudeDir, "settings.json")
	if _, ok := hooks["SessionStart"]; ok {
		t.Errorf("SessionStart should have been dropped entirely, got %+v", hooks["SessionStart"])
	}
	if len(hooks["Notification"]) != 1 {
		t.Errorf("lost unrelated Notification hook, got %+v", hooks["Notification"])
	}
}

// TestUnwireSettings_UntouchedEmptyGroupSurvives is the regression test for
// the data-loss bug this file fixes: unwireGroups conflated "we removed
// something" with "this group ended up empty", so an untouched zero-command
// group was dropped, and once it was "hooks"' only content the whole file
// got deleted by writeOrRemoveSettings, though pulsarules wired nothing.
func TestUnwireSettings_UntouchedEmptyGroupSurvives(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	seed := `{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": []}
    ]
  }
}`
	path := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if changed {
		t.Error("expected changed = false, nothing of ours was in the file")
	}

	got, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("expected settings.json to survive, stat/read err = %v", err)
	}
	if string(got) != seed {
		t.Errorf("settings.json changed: got %q, want unchanged %q", got, seed)
	}
}

// TestUnwireSettings_UntouchedGroupSurvivesAlongsideOurs asserts an unrelated
// group with no commands survives exactly when it sits next to a group
// carrying our wired command, while only ours is removed.
func TestUnwireSettings_UntouchedGroupSurvivesAlongsideOurs(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	seed := `{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": []},
      {"matcher": "", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" session-start"}]}
    ]
  }
}`
	path := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if !changed {
		t.Error("expected changed = true, our command was removed")
	}

	hooks := readHooks(t, claudeDir, "settings.json")
	groups := hooks["SessionStart"]
	if len(groups) != 1 {
		t.Fatalf("expected the untouched empty group to survive alone, got %+v", groups)
	}
	if len(groups[0].Hooks) != 0 {
		t.Errorf("expected the surviving group to keep zero commands, got %+v", groups[0].Hooks)
	}
}

// TestUnwireSettings_EmptyGroupSurvivesWithPermissions asserts an untouched
// empty group and an unrelated top-level key both survive, and the file is
// left byte-identical since nothing of ours was in it.
func TestUnwireSettings_EmptyGroupSurvivesWithPermissions(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	seed := `{
  "permissions": {"allow": ["Bash(go test *)"]},
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": []}
    ]
  }
}`
	path := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings: %v", err)
	}
	if changed {
		t.Error("expected changed = false, nothing of ours was in the file")
	}

	got, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("expected settings.json to survive, stat/read err = %v", err)
	}
	if string(got) != seed {
		t.Errorf("settings.json changed: got %q, want unchanged %q", got, seed)
	}
}

// TestUnwireSettings_Unparseable asserts a settings file that is not valid
// JSON is left untouched and the error wraps fsx.ErrUnparseableJSON.
func TestUnwireSettings_Unparseable(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	path := filepath.Join(claudeDir, "settings.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := UnwireSettings(claudeDir, "settings.json")
	if !errors.Is(err, fsx.ErrUnparseableJSON) {
		t.Fatalf("err = %v, want fsx.ErrUnparseableJSON", err)
	}
	if changed {
		t.Error("expected changed = false on an unparseable file")
	}
	got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != original {
		t.Errorf("file was rewritten: got %q, want unchanged %q", got, original)
	}
}

// TestUnwireSettings_NoOpWhenAbsent asserts a missing settings file, or one
// with no "hooks" key, is not an error.
func TestUnwireSettings_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		seed string
	}{
		{name: "file does not exist"},
		{name: "no hooks key", seed: `{"permissions": {}}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			claudeDir := t.TempDir()
			if testCase.seed != "" {
				path := filepath.Join(claudeDir, "settings.json")
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			changed, err := UnwireSettings(claudeDir, "settings.json")
			if err != nil {
				t.Fatalf("UnwireSettings: %v", err)
			}
			if changed {
				t.Error("expected changed = false, there was nothing to unwire")
			}
		})
	}
}

// TestUnwireSettings_Idempotent asserts running UnwireSettings twice is not an
// error, and the second run is a genuine no-op once the file is gone.
func TestUnwireSettings_Idempotent(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	if err := WireSettings(fakeTemplates(), claudeDir, "settings.json"); err != nil {
		t.Fatalf("seed via WireSettings: %v", err)
	}
	firstChanged, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings #1: %v", err)
	}
	if !firstChanged {
		t.Error("expected the first run to report changed = true")
	}
	secondChanged, err := UnwireSettings(claudeDir, "settings.json")
	if err != nil {
		t.Fatalf("UnwireSettings #2: %v", err)
	}
	if secondChanged {
		t.Error("expected the second run to report changed = false, nothing left to unwire")
	}
}
