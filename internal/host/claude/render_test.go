package claude

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
)

func renderedHooks(t *testing.T) map[string][]hookGroup {
	t.Helper()

	raw, err := RenderHooksBlock(Location{})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var block hooksBlock
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return block.Hooks
}

func TestRenderHooksBlock_OnlyStopCarriesATimeout(t *testing.T) {
	t.Parallel()

	hooks := renderedHooks(t)
	testCases := []struct {
		name  string
		event string
		want  bool
	}{
		{name: "Stop carries a timeout", event: "Stop", want: true},
		{name: "SessionStart keeps the host default", event: "SessionStart"},
		{name: "PostToolUse keeps the host default", event: "PostToolUse"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			for _, group := range hooks[testCase.event] {
				for _, command := range group.Hooks {
					if (len(command.Timeout) > 0) != testCase.want {
						t.Errorf("%s timeout = %q, want set=%v",
							testCase.event, command.Timeout, testCase.want)
					}
				}
			}
		})
	}
}

func TestRenderHooksBlock_StopTimeoutFromTable(t *testing.T) {
	t.Parallel()

	stop := renderedHooks(t)["Stop"][0].Hooks[0]
	seconds, err := strconv.ParseInt(string(stop.Timeout), 10, 64)
	if err != nil {
		t.Fatalf("Stop timeout %q: %v", stop.Timeout, err)
	}
	for _, event := range hook.Events() {
		if event.Mode == "stop" && int64(event.Timeout) != seconds {
			t.Errorf("rendered Stop timeout %d, table says %d", seconds, event.Timeout)
		}
	}
}

func TestRenderHooksBlock_RendersEveryEvent(t *testing.T) {
	t.Parallel()

	hooks := renderedHooks(t)
	for _, event := range hook.Events() {
		t.Run(event.Mode, func(t *testing.T) {
			t.Parallel()

			found := false
			for _, group := range hooks[event.Host] {
				for _, command := range group.Hooks {
					if group.Matcher == event.Matcher &&
						strings.HasSuffix(command.Command, `" `+event.Mode) {
						found = true
					}
				}
			}
			if !found {
				t.Errorf(
					"no %s group with matcher %q runs mode %q",
					event.Host,
					event.Matcher,
					event.Mode,
				)
			}
		})
	}
}

func TestRenderHooksBlock_MatchesGolden(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile("testdata/hooks_block.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got, err := RenderHooksBlock(Location{})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("hooks block differs from the golden:\n%s", got)
	}
}

func TestRenderHooksBlock_GlobalNamesTheInstalledScript(t *testing.T) {
	t.Parallel()

	raw, err := RenderHooksBlock(Location{GlobalDir: "/home/user/.claude"})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var block hooksBlock
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for event, groups := range block.Hooks {
		for _, group := range groups {
			command := group.Hooks[0].Command
			const prefix = `bash '/home/user/.claude/hooks/skill-router-reminder.sh' `
			if !strings.HasPrefix(command, prefix) {
				t.Errorf("%s command = %q, want prefix %q", event, command, prefix)
			}
			if strings.Contains(command, "CLAUDE_PROJECT_DIR") {
				t.Errorf("%s command = %q, must not depend on the project dir", event, command)
			}
		}
	}
}
