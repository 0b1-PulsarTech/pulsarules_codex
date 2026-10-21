package loader

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
)

type joinCase struct {
	name         string
	base, second error
	wantFull     string
	wantTail     string
	wantHas      string
}

func joinFaultCases() []joinCase {
	return slices.Concat(joinSizeCases(), joinNameCases(), joinEscapeCases())
}

func joinSizeCases() []joinCase {
	long := errors.New(strings.Repeat("\u00e9", 3000))
	problems := func(name string) error {
		item := name + ": line 3: k: unknown key"
		joined := strings.TrimSuffix(strings.Repeat(item+"; ", 200), "; ")
		return &fileError{File: name, Err: errors.New(joined)}
	}
	return []joinCase{
		{
			name:     "a base that overflows at a problem boundary",
			base:     problems("a"),
			second:   &fileError{File: "b", Err: long},
			wantTail: "unknown key; b: also invalid",
		},
		{
			name:     "a base name holding the separator",
			base:     problems("a; b"),
			second:   &fileError{File: "b", Err: long},
			wantTail: "unknown key; b: also invalid",
		},
		{
			name:     "both fit",
			base:     &fileError{File: "a", Err: errors.New("a: x")},
			second:   &fileError{File: "b", Err: errors.New("b: y")},
			wantFull: "a: x; b: y",
		},
		{
			name:     "under the message cap but over the rendered line",
			base:     &fileError{File: "a", Err: errors.New(strings.Repeat("x", 4070))},
			second:   &fileError{File: "b", Err: errors.New("b: y")},
			wantTail: "; b: also invalid",
		},
		{
			name:     "over the ceiling keeps the base and names the other",
			base:     &fileError{File: "a", Err: long},
			second:   &fileError{File: "b", Err: long},
			wantTail: "; b: also invalid",
		},
	}
}

func joinNameCases() []joinCase {
	long := errors.New(strings.Repeat("\u00e9", 3000))
	bigName := strings.Repeat("d", 3000) + "/pulsarules.yaml"
	bigErr := errors.New(bigName + ": line 1: k: unknown key")
	return []joinCase{
		{
			name:     "a long base name keeps the first reason",
			base:     &fileError{File: bigName, Err: bigErr},
			second:   &fileError{File: "b", Err: long},
			wantTail: "; b: also invalid",
			wantHas:  "unknown key",
		},
		{
			name:     "a long private name keeps its tail",
			base:     &fileError{File: "a", Err: long},
			second:   &fileError{File: strings.Repeat("p/", 2500) + "pulsarules.yaml", Err: long},
			wantTail: "p/pulsarules.yaml: also invalid",
		},
		{
			name:     "a multibyte rune at the private name cut",
			base:     &fileError{File: "a", Err: long},
			second:   &fileError{File: strings.Repeat("\u00e9", 700) + "/p.yaml", Err: long},
			wantTail: "\u00e9/p.yaml: also invalid",
		},
	}
}

func joinEscapeCases() []joinCase {
	long := errors.New(strings.Repeat("\u00e9", 3000))
	return []joinCase{
		{
			name:     "each part is escaped once",
			base:     &fileError{File: "a", Err: errors.New("a\x01")},
			second:   &fileError{File: "b", Err: errors.New("b\x01")},
			wantFull: `a\x01; b\x01`,
		},
		{
			name:     "c1 bytes grow threefold when rendered",
			base:     &fileError{File: "a", Err: errors.New(strings.Repeat("\u0085", 1000))},
			second:   &fileError{File: "bc", Err: errors.New(strings.Repeat("\u0085", 1000))},
			wantTail: "; bc: also invalid",
		},
		{
			name:     "a short base with a private fault that grows when rendered",
			base:     &fileError{File: "a", Err: errors.New("a: x")},
			second:   &fileError{File: "b", Err: errors.New(strings.Repeat("\x01", 1500))},
			wantTail: "; b: also invalid",
		},
		{
			name:     "a private file name with a control byte is escaped",
			base:     &fileError{File: "a", Err: long},
			second:   &fileError{File: "b\x01", Err: long},
			wantTail: `; b\x01: also invalid`,
		},
		{
			name:     "a private fault that is not typed falls back to the base name",
			base:     long,
			second:   long,
			wantTail: "; " + projectfile.Name + ": also invalid",
		},
	}
}

func TestJoinFaults(t *testing.T) {
	t.Parallel()

	testCases := joinFaultCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			joined := joinFaults(testCase.base, testCase.second)
			got := joined.Error()
			hit := finding.Finding{
				AnalyzerID: configAnalyzerID, Severity: finding.SeverityError,
				File: faultFile(joined, nil), Message: got,
			}
			rendered := renderHook([]finding.Finding{hit})
			requireNoFragment(t, got)
			if len(rendered) > maxJoinedReport || !utf8.ValidString(got) {
				t.Fatalf("rendered = %d bytes valid=%v", len(rendered), utf8.ValidString(got))
			}
			if !strings.Contains(got, testCase.wantHas) {
				t.Fatalf("joined = %.100q, want containing %q", got, testCase.wantHas)
			}
			if testCase.wantFull != "" && got != testCase.wantFull {
				t.Fatalf("joined = %q, want %q", got, testCase.wantFull)
			}
			if !strings.HasSuffix(got, testCase.wantTail) {
				t.Fatalf("joined ends %q, want suffix %q", got[len(got)-30:], testCase.wantTail)
			}
		})
	}
}

func TestLoadCapsTheJoinedReport(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		spec     layout
		wantPart string
	}{
		{
			name:     "both files over the budget",
			spec:     layout{base: badGlobs(20), private: badGlobs(20), worktree: true},
			wantPart: "/.git/pulsarules.yaml: also invalid",
		},
		{
			name:     "long unknown keys in the private file",
			spec:     layout{base: badGlobs(20), private: longUnknownKeys(9), git: true},
			wantPart: "; ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, findings := ForRoot(testCase.spec.build(t))
			if len(findings) != 1 {
				t.Fatalf("findings = %+v, want exactly one", findings)
			}
			got := findings[0]
			rendered := renderHook(findings)
			if len(rendered) > maxJoinedReport {
				t.Fatalf("rendered line = %d bytes, want at most %d",
					len(rendered), maxJoinedReport)
			}
			if !strings.Contains(got.Message, testCase.wantPart) {
				t.Fatalf("message = %.200q..., want containing %q", got.Message, testCase.wantPart)
			}
			if got.File != projectfile.Name {
				t.Fatalf("File = %q, want the base", got.File)
			}
		})
	}
}

func badGlobs(count int) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = "'[" + strings.Repeat("a", 100) + "'"
	}
	return doc("tests: [" + strings.Join(items, ", ") + "]")
}

func longUnknownKeys(count int) string {
	lines := make([]string, count)
	for idx := range lines {
		lines[idx] = strings.Repeat(string(rune('a'+idx)), 400) + ": 1"
	}
	return doc(lines...)
}

func renderHook(found []finding.Finding) string {
	return findings.Report{Findings: found}.Text(findings.Hook(), "")
}
