package command

import (
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestStderrTail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"blank", " \n\t ", ""},
		{"short is trimmed", "  boom\n", "boom"},
		{"control bytes become spaces", "a\x1b[31mb\x00c\nd\te", "a [31mb c\nd\te"},
		{"invalid utf8 is replaced", "a\xffb", "a?b"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := stderrTail(testCase.input)
			if got != testCase.want {
				t.Errorf("stderrTail = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestRelativeTo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	other := t.TempDir()
	testCases := []struct {
		name   string
		base   string
		file   string
		want   string
		inside bool
	}{
		{"absolute inside", root, filepath.Join(root, "a", "b.dart"), "a/b.dart", true},
		{"absolute outside", root, filepath.Join(other, "b.dart"), "", false},
		{"sibling prefix is outside", root, root + "x/b.dart", "", false},
		{"relative joins the run dir", filepath.Join(root, "web"), "b.dart", "web/b.dart", true},
		{"relative escaping the root", root, filepath.Join("..", "b.dart"), "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, inside := relativeTo(root, testCase.base, testCase.file)
			if got != testCase.want || inside != testCase.inside {
				t.Errorf("relativeTo = %q, %v; want %q, %v",
					got, inside, testCase.want, testCase.inside)
			}
		})
	}
}

func TestReportsSubIDs(t *testing.T) {
	t.Parallel()

	rep := reports{id: "tool", rule: "dart-style"}
	testCases := []struct {
		name     string
		finding  core.Finding
		wantID   string
		wantSev  core.Severity
		wantRule string
	}{
		{"config", rep.config("x"), "tool/config", core.SeverityError, ""},
		{"unavailable", rep.unavailable("dart"), "tool/unavailable", core.SeverityInfo, ""},
		{"timeout", rep.timeout("dart"), "tool/timeout", core.SeverityError, ""},
		{"failed", rep.failed("x", "e"), "tool/failed", core.SeverityError, ""},
		{
			"diagnostic",
			rep.diagnostic(diagnostic{code: "c", severity: core.SeverityInfo}, "f"),
			"tool/c", core.SeverityInfo, "dart-style",
		},
		{
			"diagnostic without a code",
			rep.diagnostic(diagnostic{severity: core.SeverityInfo}, "f"),
			"tool/diagnostic", core.SeverityInfo, "dart-style",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			finding := testCase.finding
			if finding.AnalyzerID != testCase.wantID || finding.Severity != testCase.wantSev ||
				finding.RuleID != testCase.wantRule {
				t.Errorf("finding = %+v", finding)
			}
		})
	}
}

func TestReportsFailedAppendsTail(t *testing.T) {
	t.Parallel()

	rep := reports{id: "tool"}
	if got := rep.failed(
		"dart exited with code 4",
		"boom",
	).Message; got != "dart exited with code 4: boom" {
		t.Errorf("Message = %q", got)
	}
	if got := rep.failed("dart exited with code 4", "").Message; got != "dart exited with code 4" {
		t.Errorf("Message without stderr = %q", got)
	}
}

func TestReportsDiagnosticCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		code        string
		wantID      string
		wantMessage string
	}{
		{"plain code", "unused_import", "tool/unused_import", "m"},
		{"digits and underscore", "rule_2", "tool/rule_2", "m"},
		{"no code", "", "tool/diagnostic", "m"},
		{"uppercase", "Bad", "tool/diagnostic", `m (code "Bad")`},
		{"hostile code", "../x y\nz", "tool/diagnostic", `m (code "../x y\nz")`},
		{"slash in code", "a/b", "tool/diagnostic", `m (code "a/b")`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			finding := reports{id: "tool"}.diagnostic(
				diagnostic{code: testCase.code, message: "m"}, "f",
			)
			if finding.AnalyzerID != testCase.wantID || finding.Message != testCase.wantMessage {
				t.Errorf("id %q message %q", finding.AnalyzerID, finding.Message)
			}
		})
	}
}
