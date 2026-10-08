package version

import (
	"strings"
	"testing"
)

func TestUsageText_ListsEveryBoundCommand(t *testing.T) {
	t.Parallel()

	for _, command := range Commands() {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(usageText, "pulsarules_cli "+command) {
				t.Errorf("usage.txt has no line for the bound command %q", command)
			}
		})
	}
}
