package golangcilint

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func failedExit(code string) error {
	return exec.CommandContext(context.Background(), "sh", "-c", "exit "+code).Run()
}

func TestParseOutput_Results(t *testing.T) {
	t.Parallel()

	const stderrBoom = "boom"
	testCases := []struct {
		name         string
		stdout       string
		cmdErr       error
		wantMessages []string
		wantSeverity []pass.Severity
	}{
		{
			name:         "a command that never ran reports its cause",
			cmdErr:       errors.New("not found"),
			wantMessages: []string{"golangci-lint failed: not found"},
			wantSeverity: []pass.Severity{pass.SeverityError},
		},
		{
			name:         "a real exit status reports the stderr",
			cmdErr:       failedExit("3"),
			wantMessages: []string{"golangci-lint failed: boom"},
			wantSeverity: []pass.Severity{pass.SeverityError},
		},
		{
			name:         "malformed json is a failure",
			stdout:       "{not json",
			wantMessages: []string{"failed to parse golangci-lint output: "},
			wantSeverity: []pass.Severity{pass.SeverityError},
		},
		{
			name: "an error severity issue blocks and a plain one warns",
			stdout: `{"Issues":[` +
				`{"FromLinter":"a","Text":"x","Severity":"error",` +
				`"Pos":{"Filename":"f.go","Line":3}},` +
				`{"FromLinter":"b","Text":"y","Severity":"",` +
				`"Pos":{"Filename":"g.go","Line":4}}]}`,
			wantMessages: []string{"x", "y"},
			wantSeverity: []pass.Severity{pass.SeverityError, pass.SeverityWarning},
		},
		{
			name: "paths escaping the project are dropped with a warning",
			stdout: `{"Issues":[{"FromLinter":"a","Text":"x",` +
				`"Pos":{"Filename":"../gone/f.go","Line":1}}]}`,
			wantMessages: []string{"dropped 1 finding(s)"},
			wantSeverity: []pass.Severity{pass.SeverityWarning},
		},
		{
			name:         "no issues yields no findings",
			stdout:       `{"Issues":[]}`,
			wantMessages: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := process.Result{Stdout: testCase.stdout, Stderr: stderrBoom}
			got := NewRunner("").parseOutput(result, testCase.cmdErr)
			if len(got) != len(testCase.wantMessages) {
				t.Fatalf("got %d findings, want %d: %+v",
					len(got), len(testCase.wantMessages), got)
			}
			for idx, finding := range got {
				if !strings.HasPrefix(finding.Message, testCase.wantMessages[idx]) {
					t.Errorf("message[%d] = %q, want prefix %q",
						idx, finding.Message, testCase.wantMessages[idx])
				}
				if finding.Severity != testCase.wantSeverity[idx] {
					t.Errorf("severity[%d] = %v, want %v",
						idx, finding.Severity, testCase.wantSeverity[idx])
				}
			}
		})
	}
}
