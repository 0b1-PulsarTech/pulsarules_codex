package claude

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func renderedHooks(t *testing.T) map[string][]hookGroup {
	t.Helper()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("templates: %v", err)
	}
	raw, err := RenderHooksBlock(templates)
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

func TestRenderHooksBlock_StopTimeout(t *testing.T) {
	t.Parallel()

	const minimumMargin = 15 * time.Second
	stop := renderedHooks(t)["Stop"][0].Hooks[0]
	seconds, err := strconv.ParseInt(string(stop.Timeout), 10, 64)
	if err != nil {
		t.Fatalf("Stop timeout %q: %v", stop.Timeout, err)
	}
	if seconds != StopTimeoutSeconds {
		t.Errorf("rendered Stop timeout = %d, want StopTimeoutSeconds = %d",
			seconds, StopTimeoutSeconds)
	}
	hostTimeout := time.Duration(seconds) * time.Second
	if hostTimeout < hook.StopGovernanceDeadline+minimumMargin {
		t.Errorf("Stop timeout %s must exceed the governance deadline %s by %s",
			hostTimeout, hook.StopGovernanceDeadline, minimumMargin)
	}
}
