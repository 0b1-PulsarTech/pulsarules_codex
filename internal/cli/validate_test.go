package cli

import (
	"strings"
	"testing"
)

func TestPrintProblems_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printProblems(&out, []string{"skill a\n[ERROR] forged", "plain"})
	want := "validation failed:\n  - skill a\\n[ERROR] forged\n  - plain\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
