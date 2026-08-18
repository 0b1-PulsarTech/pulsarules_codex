package analysis

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestRuleAnalyzersCheck asserts a rule declaring a registered analyzer id
// passes, one declaring an unknown id is reported naming both the rule and
// the id, and a rule declaring no analyzers at all also passes.
func TestRuleAnalyzersCheck(t *testing.T) {
	t.Parallel()

	registered := RegisteredAnalyzerIDs()
	if len(registered) == 0 {
		t.Fatal("expected at least one registered analyzer id")
	}
	knownID := registered[0]

	testCases := []struct {
		name    string
		rule    knowledge.Rule
		wantErr string
	}{
		{
			name: "known analyzer id passes",
			rule: knowledge.Rule{ID: "r", Analyzers: []string{knownID}},
		},
		{
			name:    "unknown analyzer id fails",
			rule:    knowledge.Rule{ID: "r", Analyzers: []string{"ghost-analyzer"}},
			wantErr: `rule "r" declares unknown analyzer "ghost-analyzer"`,
		},
		{
			name: "no analyzers declared passes",
			rule: knowledge.Rule{ID: "r"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			idx := &knowledge.Index{Rules: []knowledge.Rule{testCase.rule}}
			problems := RuleAnalyzersCheck(idx)

			if testCase.wantErr == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if len(problems) != 1 || !strings.Contains(problems[0], testCase.wantErr) {
				t.Fatalf("expected one problem containing %q, got %v", testCase.wantErr, problems)
			}
		})
	}
}

// TestRuleAnalyzersCheck_Embedded is the regression guard: every analyzers:
// entry in the committed knowledge base must resolve against the real
// registered analyzer set, proving the check stays silent on today's
// truthful data.
func TestRuleAnalyzersCheck_Embedded(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := RuleAnalyzersCheck(idx); len(problems) != 0 {
		t.Errorf("embedded rules should declare only registered analyzers, got %v", problems)
	}
}
