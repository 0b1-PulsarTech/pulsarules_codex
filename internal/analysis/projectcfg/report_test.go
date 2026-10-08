package projectcfg

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"testing"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

func escapedKeys(spelling, indent string) string {
	lines := make([]string, 90)
	for idx := range lines {
		lines[idx] = fmt.Sprintf(`%s"%s%03d": 1`, indent, strings.Repeat(spelling, 80), idx)
	}
	return strings.Join(lines, "\n")
}

func TestLoadBudgetsTheRenderedLine(t *testing.T) {
	t.Parallel()

	shapes := map[string]string{
		"control": doc("analyzers:", escapedKeys(`\x01`, "  ")),
		"c1":      doc("analyzers:", escapedKeys(`\u0085`, "  ")),
		"bidi":    doc("analyzers:", escapedKeys(`\u202e`, "  ")),
		"params": doc(
			"analyzers:", "  big-comment:", "    params:", escapedKeys(`\x01`, "      ")),
	}
	testCases := []struct {
		name string
		spec layout
	}{
		{"control single", layout{base: shapes["control"]}},
		{"c1 single", layout{base: shapes["c1"]}},
		{"bidi single", layout{base: shapes["bidi"]}},
		{"control params single", layout{base: shapes["params"]}},
		{"control private only", layout{private: shapes["control"], git: true}},
		{"control joined", layout{base: shapes["control"], private: shapes["control"], git: true}},
		{"c1 joined", layout{base: shapes["c1"], private: shapes["c1"], git: true}},
		{"params joined", layout{base: shapes["params"], private: shapes["params"], git: true}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, findings := ForRoot(testCase.spec.build(t))
			if len(findings) != 1 {
				t.Fatalf("findings = %d, want exactly one", len(findings))
			}
			requireFits(t, findings[0])
		})
	}
}

func TestRenderOverhead(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, file string
	}{
		{"a plain name", config.ProjectFileName},
		{"a quoted name", " odd\x01 -> name "},
		{"a long name", strings.Repeat("d/", 200) + "pulsarules.yaml"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			bare := core.Finding{
				AnalyzerID: ConfigAnalyzerID, Severity: core.SeverityError, File: testCase.file,
			}
			want := len(analysis.FormatFindings([]core.Finding{bare}, analysis.StyleHook, ""))
			if got := renderOverhead(testCase.file); got != want {
				t.Fatalf("renderOverhead() = %d, want %d", got, want)
			}
		})
	}
}

func unknownKeys() string {
	lines := make([]string, 40)
	for idx := range lines {
		lines[idx] = fmt.Sprintf("bogus%02d: 1", idx)
	}
	return doc(lines...)
}

func delegatedKeys(count int) string {
	lines := make([]string, 0, count+1)
	lines = append(lines, "delegated:")
	for idx := range count {
		lines = append(lines, fmt.Sprintf(
			`  - {id: "%s/%d", run: [x], format: dart-analyze-json}`,
			strings.Repeat(`\x01`, 500), idx))
	}
	return doc(lines...)
}

func requireFits(t *testing.T, finding core.Finding, parts ...string) {
	t.Helper()
	for _, style := range []analysis.FindingStyle{analysis.StyleHook, analysis.StyleCLI} {
		rendered := analysis.FormatFindings([]core.Finding{finding}, style, "")
		if len(rendered) > maxJoinedReport {
			t.Fatalf("%s line = %d bytes, want at most %d", style, len(rendered), maxJoinedReport)
		}
		for _, part := range parts {
			if !strings.Contains(rendered, part) {
				t.Fatalf("%s line lacks %q: %.200q", style, part, rendered)
			}
		}
	}
	requireNoFragment(t, finding.Message)
}

func requireNoFragment(t *testing.T, message string) {
	t.Helper()
	dangling := regexp.MustCompile(
		`(\\|\\x[0-9a-f]?|\\u[0-9a-f]{0,3}|\\U[0-9a-f]{0,7})(\.\.\.|; )`)
	if fragment := dangling.FindString(message); fragment != "" {
		t.Fatalf("message holds the fragment %q", fragment)
	}
}

type capCase struct {
	name               string
	err                error
	file               *config.ProjectFile
	wantFile           string
	wantSame, wantClip bool
}

func capReportCases() []capCase {
	short := &FileError{File: "a.yaml", Err: errors.New("a.yaml: line 1: bad")}
	return []capCase{
		{"no error", nil, nil, "", true, false},
		{"a short error is returned as is", short, nil, "a.yaml", true, false},
		{
			"a long message",
			&FileError{File: "a.yaml", Err: errors.New(strings.Repeat("x", 5000))},
			nil, "a.yaml", false, true,
		},
		{
			"control bytes grow when escaped",
			&FileError{File: "a.yaml", Err: errors.New(strings.Repeat("\x01", 2000))},
			nil, "a.yaml", false, true,
		},
		{
			"a long file name is shortened",
			&FileError{File: strings.Repeat("\xff", 1100), Err: errors.New("bad")},
			nil, "..." + strings.Repeat("\xff", 254), false, false,
		},
		{
			"an untyped error takes the loaded file",
			errors.New(strings.Repeat("\u0085", 2000)),
			&config.ProjectFile{Path: "p.yaml"}, "p.yaml", false, true,
		},
	}
}

func errorFinding(err error) core.Finding {
	return core.Finding{
		AnalyzerID: ConfigAnalyzerID, Severity: core.SeverityError,
		File: faultFile(err, nil), Message: err.Error(),
	}
}

func TestCapReport(t *testing.T) {
	t.Parallel()

	testCases := capReportCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := capReport(testCase.err, testCase.file)
			if (got == testCase.err) != testCase.wantSame {
				t.Fatalf("capReport() = %v, want unchanged = %v", got, testCase.wantSame)
			}
			if got == nil {
				return
			}
			requireFits(t, errorFinding(got))
			if clipped := strings.HasSuffix(got.Error(), "..."); clipped != testCase.wantClip {
				t.Fatalf("clipped = %v, want %v", clipped, testCase.wantClip)
			}
			if faultFile(got, nil) != testCase.wantFile {
				t.Fatalf("File = %.60q, want %.60q", faultFile(got, nil), testCase.wantFile)
			}
		})
	}
}

func deepOf(size int) string {
	parts := make([]string, 0, size/200+1)
	for ; size > 200; size -= 201 {
		parts = append(parts, strings.Repeat("m", 200))
	}
	return filepath.Join(parts...)
}

func exactDeep(size int) string {
	var parts []string
	for ; size > 200; size -= 201 {
		parts = append(parts, strings.Repeat("m", 200))
	}
	return filepath.Join(append(parts, strings.Repeat("m", max(size, 1)))...)
}

func TestCapReportKeepsTheMeaning(t *testing.T) {
	t.Parallel()

	const tail = "/.git/pulsarules.yaml"
	problems := func(name string, count int) string {
		items := make([]string, count)
		for idx := range items {
			items[idx] = fmt.Sprintf("%s: line %d: k: unknown key", name, idx+1)
		}
		return strings.Join(items, "; ")
	}
	testCases := []struct {
		name, file, merged, wantSuffix string
		count                          int
	}{
		{"many problems keep the first", strings.Repeat("d", 1500) + tail, "", moreProblems, 6},
		{"one problem shortens only the name", strings.Repeat("d", 4000) + tail, "", "key", 1},
		{"a rune at the name cut", strings.Repeat("\u00e9", 900) + tail, "", moreProblems, 4},
		{"a merged name", strings.Repeat("d", 1500) + tail, strings.Repeat("p", 1500), "key", 1},
		{
			"a merged name, many problems",
			strings.Repeat("d", 1200) + tail, strings.Repeat("p", 1200), moreProblems, 6,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			prefix, file := testCase.file, (*config.ProjectFile)(nil)
			text := problems(prefix, testCase.count)
			err := error(&FileError{File: prefix, Err: errors.New(text)})
			if testCase.merged != "" {
				prefix = testCase.file + " + " + testCase.merged + " (merged)"
				file = &config.ProjectFile{Path: prefix}
				err = errors.New(problems(prefix, testCase.count))
			}
			got := capReport(err, file)
			shown, probe := faultFile(got, nil), cmp.Or(testCase.merged, prefix)[:1100]
			requireFits(t, errorFinding(got), "unknown key", "pulsarules.yaml")
			if strings.Contains(got.Error(), probe) || !strings.HasSuffix(shown, tail) ||
				!strings.HasSuffix(got.Error(), testCase.wantSuffix) || !utf8.ValidString(shown) {
				t.Fatalf("capReport() = %.80q, File %.30q", got, shown)
			}
		})
	}
}

func TestCapReportFillsTheLine(t *testing.T) {
	t.Parallel()

	testCases := make([]struct{ name, padding string }, 17)
	for pad := range testCases {
		testCases[pad].name = fmt.Sprintf("key padding %d", pad)
		testCases[pad].padding = strings.Repeat("p", pad)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			items := make([]string, 300)
			for idx := range items {
				items[idx] = fmt.Sprintf(
					"a.yaml: line %d: k%s: unknown key", idx+100, testCase.padding)
			}
			fault := &FileError{File: "a.yaml", Err: errors.New(strings.Join(items, "; "))}
			got := capReport(fault, nil)
			rendered := analysis.FormatFindings(
				[]core.Finding{errorFinding(got)}, analysis.StyleHook, "")
			if len(rendered) > maxJoinedReport || len(rendered) <= maxJoinedReport-60 {
				t.Fatalf("rendered = %d bytes, want it to fill the line", len(rendered))
			}
		})
	}
}

func dirList(prefix string, count int) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = fmt.Sprintf("%s%d", prefix, idx)
	}
	return doc("skip_dirs: [" + strings.Join(items, ", ") + "]")
}

func TestLoadCapsEveryConfigError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, reason string
		spec         layout
	}{
		{"many unknown keys", "unknown key", layout{
			worktree: true, deep: deepOf(600),
			private: unknownKeys(),
		}},
		{"delegated ids", "larger than", layout{
			worktree: true, deep: deepOf(600),
			private: delegatedKeys(128),
		}},
		{"a private path of 2000 bytes", "unknown key", layout{
			worktree: true,
			deep:     deepOf(1900), private: unknownKeys(),
		}},
		{"a common dir of 2900 bytes, both files", "unknown key", layout{
			worktree: true,
			deep:     deepOf(2800), base: unknownKeys(), private: unknownKeys(),
		}},
		{"a read fault under a long root", "stat: " + syscall.ELOOP.Error(), layout{
			deep: deepOf(3950) + "/" + strings.Repeat("m", 120), baseLoop: true,
		}},
		{"an open fault under a long root", "open: " + syscall.EACCES.Error(), layout{
			deep: deepOf(3950) + "/" + strings.Repeat("m", 120), unreadable: true,
		}},
		{"a merged fault under a long main path", "more than 1024 entries", layout{
			worktree: true, privateLen: 4040,
			base: dirList("d", 600), private: dirList("p", 600),
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.spec.unreadable && os.Geteuid() == 0 {
				t.Skip("root opens a mode 000 file")
			}
			_, findings := ForRoot(testCase.spec.build(t))
			if len(findings) != 1 {
				t.Fatalf("findings = %d, want exactly one", len(findings))
			}
			requireFits(t, findings[0], testCase.reason, "pulsarules.yaml")
		})
	}
}

func TestLoadCapsAnExplicitPath(t *testing.T) {
	t.Parallel()

	odd := "a\x01\u202e\u0085\xff/" + strings.Repeat("z", 250) + "/\x01\x01"
	narrowing := doc("preset: minimal", overridesOf(slices.Repeat(
		[]string{enable("lib/**", "doc-comment")}, 30)...))
	testCases := []struct {
		name, dir, content string
		reason             string
	}{
		{"a 1500 byte path", deepOf(1450), unknownKeys(), "unknown key"},
		{"a 2040 byte path", deepOf(1990), unknownKeys(), "unknown key"},
		{"a 3700 byte path", deepOf(3650), unknownKeys(), "unknown key"},
		{"a 4000 byte path", deepOf(3950), unknownKeys(), "unknown key"},
		{"control and bidi bytes", odd, unknownKeys(), "unknown key"},
		{"a 2000 byte path, build fault", deepOf(1950), narrowing, "narrow it"},
		{"control and bidi bytes, build fault", odd + "/" + deepOf(1950), narrowing, "narrow it"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), testCase.dir)
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			path := filepath.Join(dir, privateFileName)
			write(t, path, testCase.content)
			file, err := Load("", path, "")
			if err == nil {
				_, err = Build(file, "")
			}
			if err == nil {
				t.Fatal("Load() and Build() error = nil, want the faults")
			}
			requireFits(t, errorFinding(err), testCase.reason, privateFileName)
		})
	}
}

func TestCapReportCutsAtAProblem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, dir, message string
		wantMarker         bool
	}{
		{"a name holding the separator", strings.Repeat("a; b/", 205), "", true},
		{"a name holding the separator, longer", strings.Repeat("a; b/", 150), "", true},
		{"one problem with the separator", "", "x.yaml: " + strings.Repeat("a; b ", 1500), false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := capReport(&FileError{File: "x.yaml", Err: errors.New(testCase.message)}, nil)
			if testCase.dir != "" {
				dir := filepath.Join(t.TempDir(), testCase.dir)
				if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
					t.Fatalf("mkdir: %v", mkErr)
				}
				write(t, filepath.Join(dir, privateFileName), unknownKeys())
				_, err = Load("", filepath.Join(dir, privateFileName), "")
			}
			got := err.Error()
			if testCase.wantMarker != strings.HasSuffix(got, moreProblems) {
				t.Fatalf("message ends %q, want marker %v", got[len(got)-40:], testCase.wantMarker)
			}
			kept := strings.TrimSuffix(got, moreProblems)
			if testCase.dir != "" && !strings.HasSuffix(kept, "unknown key") {
				t.Fatalf("last kept problem lacks its reason: %q", kept[len(kept)-60:])
			}
		})
	}
}

func TestClipHeadLeavesNoFragment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, text string
		limit      int
	}{
		{"a cut escape start", "->\u00e9\\x\xc3a...", 10},
		{"a lone backslash", "ab\\xyzw", 6},
		{"a cut unicode start", "ab\\u00zwvv", 9},
		{"backslash pairs", "a\\\\\\xyzwvv", 8},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := config.ClipHead(testCase.text, testCase.limit)
			requireNoFragment(t, got)
			if !strings.HasSuffix(got, "...") || strings.HasSuffix(got, `\...`) {
				t.Fatalf("ClipHead() = %q, want an ellipsis after no backslash", got)
			}
		})
	}
}

func TestEscapeLineIsIdempotent(t *testing.T) {
	t.Parallel()

	testCases := []struct{ name, text string }{
		{"control bytes", "a\x01\x1b[31m\n\t\x7f"},
		{"c1 bytes", "a\u0085\u009b\u0090b"},
		{"bidi marks", "a\u202e\u2066\u200fb"},
		{"invalid bytes", "a\xff\xc3\xe2\x80b"},
		{"backslashes", `a\x01\\u0085\`},
		{"escape-looking text", `\x01\u0085\U000e0001\xff`},
		{"mixed", "->\u00e9\\x\xc3a\x01\u202e\\"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			once := escape.Line(testCase.text)
			if twice := escape.Line(once); twice != once {
				t.Fatalf("Line(Line(x)) = %q, want %q", twice, once)
			}
		})
	}
}
