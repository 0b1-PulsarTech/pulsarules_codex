package registry

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRules_TargetsExist(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	for analyzerID, ruleID := range Rules() {
		if !idx.RuleExists(ruleID) {
			t.Errorf("rule %q of %q is not in the knowledge index", ruleID, analyzerID)
		}
	}
	pinned := []string{"commits", "named-returns", "source-files", "flag-arguments"}
	for _, ruleID := range pinned {
		if !idx.RuleExists(ruleID) {
			t.Errorf("pinned rule %q is not in the knowledge index", ruleID)
		}
	}
}
