package hook

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestDispatchContract_MatchesContract is the drift test for the first two
// consumers: dispatches session-start/subagent-start through the real
// Dispatcher and real embedded templates (not hookTemplates, which
// decouples other tests from asset wording), asserting byte-identical
// output against what contract renders directly - proving Dispatcher holds no copy.
func TestDispatchContract_MatchesContract(t *testing.T) {
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
