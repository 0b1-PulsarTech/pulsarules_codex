package cli

import (
	"strings"
	"testing"
)

func TestUsageText_ListsEveryBoundCommand(t *testing.T) {
	t.Parallel()

	for _, candidate := range table() {
		t.Run(candidate.name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(usageText, "pulsarules_cli "+candidate.name) {
				t.Errorf("usage.txt has no line for the bound command %q", candidate.name)
			}
		})
	}
}
