package install

import (
	"errors"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

func TestPrintReport_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		report     target.Report
		wantStdout string
		wantStderr string
	}{
		{
			name:       "note",
			report:     target.Report{Notes: []string{"wrote: x\n[ERROR] forged"}},
			wantStdout: `wrote: x\n[ERROR] forged` + "\n",
		},
		{
			name:       "warning",
			report:     target.Report{Warnings: []string{"odd: y\n[ERROR] forged"}},
			wantStderr: `odd: y\n[ERROR] forged` + "\n",
		},
		{
			name:       "plain text is untouched",
			report:     target.Report{Notes: []string{"wrote: a/b"}},
			wantStdout: "wrote: a/b\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr strings.Builder
			printReport(&stdout, &stderr, testCase.report)
			if stdout.String() != testCase.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), testCase.wantStdout)
			}
			if stderr.String() != testCase.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), testCase.wantStderr)
			}
		})
	}
}

func TestPrintWarning_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printWarning(&out, "binary", errors.New("copy a\n[ERROR] forged"))
	want := `warning: binary: copy a\n[ERROR] forged` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}
