package message

import (
	"testing"
)

func TestRuleSpecs_IDsNonEmptyAndDistinct(t *testing.T) {
	t.Parallel()

	seen := map[string]rule{}
	for index, spec := range ruleSpecs() {
		if spec.id == "" {
			t.Errorf("rule %d has an empty id", index)
		}
		if first, dup := seen[spec.id]; dup {
			t.Errorf("rule %d repeats id %q of rule %d", index, spec.id, first)
		}
		seen[spec.id] = rule(index)
	}
}
