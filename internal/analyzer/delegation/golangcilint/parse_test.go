package golangcilint

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func TestParseOutput_LockContention(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		stderr      string
		wantBlocked bool
	}{
		{
			name:   "another copy holds the lock",
			stderr: "Error: parallel golangci-lint is running\n",
		},
		{
			name:        "a real linter failure still blocks",
			stderr:      "Error: can't load config: unknown linter\n",
			wantBlocked: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := process.Result{Stderr: testCase.stderr}
			got := NewRunner("").parseOutput(result, &exec.ExitError{})
			if len(got) != 1 {
				t.Fatalf("got %d findings, want 1: %+v", len(got), got)
			}
			blocked := got[0].Severity == finding.SeverityError
			if blocked != testCase.wantBlocked {
				t.Fatalf("blocked = %v, want %v: %+v", blocked, testCase.wantBlocked, got[0])
			}
		})
	}
}

func failedExit(code string) error {
	return exec.CommandContext(context.Background(), "sh", "-c", "exit "+code).Run()
}

type parseCase struct {
	name         string
	stdout       string
	stderr       string
	cmdErr       error
	wantMessages []string
	wantSeverity []finding.Severity
}

func timedOutError() error {
	return fmt.Errorf("run: %w: %w", context.DeadlineExceeded, failedExit("1"))
}

func exitStatusCases() []parseCase {
	return []parseCase{
		{
			name:         "a command that never ran reports its cause",
			cmdErr:       errors.New("not found"),
			wantMessages: []string{"golangci-lint failed: not found"},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name:         "a real exit status reports the stderr",
			cmdErr:       failedExit("3"),
			wantMessages: []string{"golangci-lint failed: boom"},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name:         "exit 1 with no issues is not a failure",
			stdout:       `{"Issues":[]}`,
			cmdErr:       failedExit("1"),
			wantMessages: nil,
		},
		{
			name:         "exit 7 with no issues is not a failure",
			stdout:       `{"Issues":[]}`,
			cmdErr:       failedExit("7"),
			wantMessages: nil,
		},
		{
			name:         "exit 4 is a failure",
			cmdErr:       failedExit("4"),
			wantMessages: []string{"golangci-lint failed: boom"},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name:         "exit 5 is a failure",
			cmdErr:       failedExit("5"),
			wantMessages: []string{"golangci-lint failed: boom"},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name:         "a timeout reports the stderr like an exit status",
			cmdErr:       timedOutError(),
			wantMessages: []string{"golangci-lint failed: boom"},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name:         "a timeout with lock contention warns",
			stderr:       "Error: parallel golangci-lint is running\n",
			cmdErr:       timedOutError(),
			wantMessages: []string{"golangci-lint was busy elsewhere"},
			wantSeverity: []finding.Severity{finding.SeverityWarning},
		},
	}
}

func TestParseOutput_Results(t *testing.T) {
	t.Parallel()

	const stderrBoom = "boom"
	testCases := append(exitStatusCases(), []parseCase{
		{
			name:         "malformed json is a failure",
			stdout:       "{not json",
			wantMessages: []string{"failed to parse golangci-lint output: "},
			wantSeverity: []finding.Severity{finding.SeverityError},
		},
		{
			name: "an error severity issue blocks and a plain one warns",
			stdout: `{"Issues":[` +
				`{"FromLinter":"a","Text":"x","Severity":"error",` +
				`"Pos":{"Filename":"f.go","Line":3}},` +
				`{"FromLinter":"b","Text":"y","Severity":"",` +
				`"Pos":{"Filename":"g.go","Line":4}}]}`,
			wantMessages: []string{"x", "y"},
			wantSeverity: []finding.Severity{finding.SeverityError, finding.SeverityWarning},
		},
		{
			name: "paths escaping the project are dropped with a warning",
			stdout: `{"Issues":[{"FromLinter":"a","Text":"x",` +
				`"Pos":{"Filename":"../gone/f.go","Line":1}}]}`,
			wantMessages: []string{"dropped 1 finding(s)"},
			wantSeverity: []finding.Severity{finding.SeverityWarning},
		},
		{
			name:         "no issues yields no findings",
			stdout:       `{"Issues":[]}`,
			wantMessages: nil,
		},
	}...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stderr := stderrBoom
			if testCase.stderr != "" {
				stderr = testCase.stderr
			}
			result := process.Result{Stdout: testCase.stdout, Stderr: stderr}
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

func TestParseOutput_BoundsMessage(t *testing.T) {
	t.Parallel()

	const first, last = "FIRSTBYTES", "LASTBYTES"
	huge := first + strings.Repeat("e", 1<<20) + last
	exceeded := fmt.Sprintf(
		"golangci-lint failed: output exceeded %d bytes: ",
		command.MaxOutputBytes,
	)
	testCases := []struct {
		name      string
		cmdErr    error
		frame     string
		keepsTail bool
	}{
		{"exit status stderr", failedExit("3"), "golangci-lint failed: ", true},
		{"never ran", errors.New(huge), "golangci-lint failed: ", false},
		{"output exceeded", fmt.Errorf("x: %w", process.ErrOutputExceeded), exceeded, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewRunner("").parseOutput(process.Result{Stderr: huge}, testCase.cmdErr)
			if len(got) != 1 {
				t.Fatalf("got %d findings, want 1", len(got))
			}
			message := got[0].Message
			if limit := maxMessageBytes + len(testCase.frame); len(message) > limit {
				t.Errorf("message is %d bytes, want at most %d", len(message), limit)
			}
			if testCase.keepsTail &&
				(!strings.HasSuffix(message, last) || strings.Contains(message, first)) {
				t.Errorf("message must keep the tail of stderr, got %.40q...", message)
			}
		})
	}
}

func TestParseOutput_EmptyStderrHasNoDanglingColon(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		cmdErr error
		want   string
	}{
		{"exit status", failedExit("3"), "golangci-lint failed"},
		{
			"output exceeded",
			fmt.Errorf("x: %w", process.ErrOutputExceeded),
			fmt.Sprintf("golangci-lint failed: output exceeded %d bytes", command.MaxOutputBytes),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewRunner("").parseOutput(process.Result{}, testCase.cmdErr)
			if len(got) != 1 || got[0].Message != testCase.want {
				t.Fatalf("got %+v, want one message %q", got, testCase.want)
			}
		})
	}
}

func TestEscapesProject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path string
		want bool
	}{
		{"f.go", false},
		{"a/../f.go", false},
		{"../f.go", true},
		{"a/../../f.go", true},
		{"..", false},
		{"/abs/f.go", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()

			if got := escapesProject(testCase.path); got != testCase.want {
				t.Errorf("escapesProject(%q) = %v, want %v", testCase.path, got, testCase.want)
			}
		})
	}
}
