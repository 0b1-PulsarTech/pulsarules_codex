//go:build unix

package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const (
	entryID       = "dart-analyze"
	outsideReport = `{"version":1,"diagnostics":[{"code":"c","severity":"ERROR",` +
		`"location":{"file":"/elsewhere/x.dart"}}]}`
)

func writeScript(t *testing.T, body string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake.sh")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return []string{"sh", path}
}

func recordedPayload(t *testing.T, root string) string {
	t.Helper()
	recorded, err := os.ReadFile("testdata/dart_analyze.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "payload.json")
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	payload := strings.ReplaceAll(string(recorded), "{{ROOT}}", physical)
	if writeErr := os.WriteFile(path, []byte(payload), 0o600); writeErr != nil {
		t.Fatalf("write payload: %v", writeErr)
	}
	return path
}

func newIndex(t *testing.T) *knowledge.Index {
	t.Helper()
	index, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}
	return index
}

type runCase struct {
	name     string
	entry    func(t *testing.T, root string) pass.DelegatedCommand
	wantIDs  []string
	wantFile []string
	wantSev  []finding.Severity
	wantText string
}

func entryRunning(
	body func(t *testing.T, root string) string,
) func(*testing.T, string) pass.DelegatedCommand {
	return func(t *testing.T, root string) pass.DelegatedCommand {
		t.Helper()
		return pass.DelegatedCommand{
			ID: entryID, Format: "dart-analyze-json", Run: writeScript(t, body(t, root)),
		}
	}
}

func recordedWithExit(code int) func(*testing.T, string) pass.DelegatedCommand {
	return entryRunning(func(t *testing.T, root string) string {
		t.Helper()
		return fmt.Sprintf("cat %q\nexit %d\n", recordedPayload(t, root), code)
	})
}

func inlineOutput(output string, code int) func(*testing.T, string) pass.DelegatedCommand {
	return entryRunning(func(*testing.T, string) string {
		return fmt.Sprintf("printf '%%s' '%s'\nexit %d\n", output, code)
	})
}

func TestRunEntryOutcomes(t *testing.T) {
	t.Parallel()

	runRunCases(t, slices.Concat(reportingCases(), droppedCases(), failureCases()))
}

func reportingCases() []runCase {
	recordedIDs := []string{
		entryID + "/inference_failure_on_untyped_parameter",
		entryID + "/prefer_interpolation_to_compose_strings",
		entryID + "/uri_has_not_been_generated",
	}
	recordedFiles := []string{
		"packages/nucleon/test/l10n/messages_for_test.dart",
		"packages/nucleon/test/settings/ai_config_scope_test.dart",
		"packages/nucleon/lib/nucleon.dart",
	}
	recordedSev := []finding.Severity{
		finding.SeverityWarning,
		finding.SeverityInfo,
		finding.SeverityError,
	}

	return []runCase{
		{
			"errors present, exit 3",
			recordedWithExit(3),
			recordedIDs,
			recordedFiles,
			recordedSev,
			"",
		},
		{
			"warnings only, exit 2",
			recordedWithExit(2),
			recordedIDs,
			recordedFiles,
			recordedSev,
			"",
		},
		{"infos only, exit 1", recordedWithExit(1), recordedIDs, recordedFiles, recordedSev, ""},
		{"clean, exit 0", recordedWithExit(0), recordedIDs, recordedFiles, recordedSev, ""},
		{
			"diagnostic without a file is kept unscoped",
			inlineOutput(`{"version":1,"diagnostics":[{"code":"c","severity":"INFO"}]}`, 3),
			[]string{entryID + "/c"},
			[]string{""},
			[]finding.Severity{finding.SeverityInfo},
			"",
		},
	}
}

func droppedCases() []runCase {
	mixedReport := `{"version":1,"diagnostics":[{"code":"c","severity":"INFO",` +
		`"location":{"file":"lib/a.dart"}},{"code":"c","severity":"ERROR",` +
		`"location":{"file":"/elsewhere/x.dart"}}]}`

	return []runCase{
		{
			"file outside the root, exit 0, is reported as dropped",
			inlineOutput(outsideReport, 0),
			[]string{entryID + "/dropped"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			"1 diagnostic on a path outside the project or missing",
		},
		{
			"a dropped error beside a kept info is reported after it",
			inlineOutput(mixedReport, 3),
			[]string{entryID + "/c", entryID + "/dropped"},
			[]string{"lib/a.dart", ""},
			[]finding.Severity{finding.SeverityInfo, finding.SeverityError},
			"",
		},
	}
}

func failureCases() []runCase {
	return slices.Concat(exitCases(), decodeOutputCases(), emptyReportCases(), setupCases())
}

func exitCases() []runCase {
	return []runCase{
		{
			"unexpected exit code",
			entryRunning(func(*testing.T, string) string { return "echo kaboom >&2\nexit 64\n" }),
			[]string{entryID + "/failed"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			"sh exited with code 64: kaboom",
		},
	}
}

func decodeOutputCases() []runCase {
	return []runCase{
		{
			"undecodable output",
			inlineOutput("not json", 0),
			[]string{entryID + "/failed"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			"undecodable output",
		},
		{
			"output past the cap",
			entryRunning(func(*testing.T, string) string {
				return fmt.Sprintf("head -c %d /dev/zero\n", MaxOutputBytes+1)
			}),
			[]string{entryID + "/failed"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			fmt.Sprintf("output exceeded %d bytes", MaxOutputBytes),
		},
	}
}

func emptyReportCases() []runCase {
	return []runCase{
		{
			"exit 3 with no diagnostics",
			inlineOutput(`{"version":1,"diagnostics":[]}`, 3),
			[]string{entryID + "/failed"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			"exit 3 but no diagnostics inside the project",
		},
		{
			"exit 1 with every file outside the root",
			inlineOutput(outsideReport, 1),
			[]string{entryID + "/failed"},
			[]string{""},
			[]finding.Severity{finding.SeverityError},
			"exit 1 but no diagnostics inside the project",
		},
	}
}

func setupCases() []runCase {
	return []runCase{
		{
			"binary missing",
			func(*testing.T, string) pass.DelegatedCommand {
				return pass.DelegatedCommand{
					ID: entryID, Format: "dart-analyze-json", Run: []string{"no-such-tool-xyz"},
				}
			},
			[]string{entryID + "/unavailable"},
			[]string{""},
			[]finding.Severity{finding.SeverityInfo},
			"no-such-tool-xyz not found on PATH",
		},
	}
}

func TestRunEntryRuleAndDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	script := `printf '{"version":1,"diagnostics":[{"code":"c","severity":"INFO",` +
		`"location":{"file":"%s/f.dart"}}]}' "$(pwd)"` + "\n"
	entry := pass.DelegatedCommand{
		ID: entryID, Dir: "sub", Format: "dart-analyze-json", Rule: "dart-style",
		Run: writeScript(t, script),
	}

	findings := NewAnalyzer(newIndex(t)).runEntry(root, nil, entry)
	if len(findings) != 1 {
		t.Fatalf("findings = %+v", findings)
	}
	if findings[0].File != "sub/f.dart" {
		t.Errorf("File = %q, want sub/f.dart (command must run in root/sub)", findings[0].File)
	}
	if findings[0].RuleID != "dart-style" {
		t.Errorf("RuleID = %q, want dart-style", findings[0].RuleID)
	}
}

func exitError(code int) error {
	runErr := exec.CommandContext(
		context.Background(), "sh", "-c", fmt.Sprintf("exit %d", code),
	).Run()
	return &process.Error{Name: "sh", Err: runErr}
}

type classifyCase struct {
	name     string
	err      error
	wantID   string
	wantExit int
	wantText string
}

func classifyCases() []classifyCase {
	return []classifyCase{
		{"no error", nil, "", 0, ""},
		{"reporting exit", exitError(3), "", 3, ""},
		{"other exit", exitError(7), entryID + "/failed", 0, ""},
		{
			"deadline",
			&process.Error{Err: fmt.Errorf("%w: killed", context.DeadlineExceeded)},
			entryID + "/timeout", 0, "",
		},
		{
			"output cap",
			&process.Error{Err: fmt.Errorf("1 bytes: %w", process.ErrOutputExceeded)},
			entryID + "/failed", 0, "",
		},
		{
			"start failure leaves no dangling colon",
			&process.Error{Err: errors.New("fork failed")},
			entryID + "/failed", 0, "sh did not run: fork failed",
		},
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	kind := formats()["dart-analyze-json"]
	for _, testCase := range classifyCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, exit := classify(
				invocation{rep: reports{id: entryID}, kind: kind, name: "sh"},
				testCase.err, process.Result{},
			)
			if exit != testCase.wantExit {
				t.Errorf("exit = %d, want %d", exit, testCase.wantExit)
			}
			if (got == nil) != (testCase.wantID == "") {
				t.Fatalf("classify = %+v, want id %q", got, testCase.wantID)
			}
			if got != nil && got.AnalyzerID != testCase.wantID {
				t.Errorf("AnalyzerID = %q, want %q", got.AnalyzerID, testCase.wantID)
			}
			if got != nil && testCase.wantText != "" && got.Message != testCase.wantText {
				t.Errorf("Message = %q, want %q", got.Message, testCase.wantText)
			}
		})
	}
}

func runRunCases(t *testing.T, testCases []runCase) {
	t.Helper()
	index := newIndex(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			findings := NewAnalyzer(index).runEntry(root, nil, testCase.entry(t, root))

			ids := make([]string, len(findings))
			files := make([]string, len(findings))
			severities := make([]finding.Severity, len(findings))
			for idx, finding := range findings {
				ids[idx], files[idx] = finding.AnalyzerID, finding.File
				severities[idx] = finding.Severity
				if !strings.HasPrefix(finding.AnalyzerID, entryID+"/") {
					t.Errorf("finding %q lacks the %s/ prefix", finding.AnalyzerID, entryID)
				}
			}
			if !slices.Equal(ids, testCase.wantIDs) {
				t.Fatalf("ids = %v, want %v", ids, testCase.wantIDs)
			}
			if !slices.Equal(files, testCase.wantFile) ||
				!slices.Equal(severities, testCase.wantSev) {
				t.Errorf("files = %v severities = %v", files, severities)
			}
			if testCase.wantText != "" &&
				!strings.Contains(findings[0].Message, testCase.wantText) {
				t.Errorf(
					"Message = %.200q, want containing %.100q",
					findings[0].Message,
					testCase.wantText,
				)
			}
		})
	}
}
