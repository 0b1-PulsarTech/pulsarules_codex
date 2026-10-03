package hook

import (
	"errors"
	"strings"
	"testing"
)

func TestPrintHookError_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printHookError(&out, "get templates", errors.New("open a\n[ERROR] forged"))
	want := `hook: get templates: open a\n[ERROR] forged` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
