//go:build unix

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const (
	configPath = ".dart_tool/package_config.json"
)

func buildTree(t *testing.T, files []string) (string, listing) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	listed := listing{}
	for _, name := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if mkErr := os.MkdirAll(filepath.Dir(full), 0o700); mkErr != nil {
			t.Fatalf("mkdir: %v", mkErr)
		}
		if writeErr := os.WriteFile(full, nil, 0o600); writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
		listed[name] = struct{}{}
	}
	return root, listed
}

type precheckCase struct {
	name    string
	files   []string
	dir     string
	useList bool
	want    string
}

func precheckCases() []precheckCase {
	return []precheckCase{
		{
			"root config covers the member pubspecs",
			[]string{"pubspec.yaml", configPath, "packages/a/pubspec.yaml"},
			"", true, "",
		},
		{
			"deep members covered by the root config",
			[]string{"pubspec.yaml", configPath, "pkg/b/c/pubspec.yaml"},
			"", true, "",
		},
		{
			"entry dir below the workspace root config",
			[]string{configPath, "app/pubspec.yaml"},
			"app", true, "",
		},
		{
			"no pubspec at all",
			[]string{"lib/a.dart"},
			"", true, "",
		},
		{
			"root pubspec without config",
			[]string{"pubspec.yaml"},
			"", true,
			"pubspec.yaml in . has no " + configPath +
				"; run `dart pub get` (then `dart run build_runner build`)",
		},
		{
			"config only below the pubspec does not count",
			[]string{"pubspec.yaml", "sub/" + configPath},
			"", true,
			"pubspec.yaml in . has no " + configPath,
		},
		{
			"pubspec outside the entry dir is not checked",
			[]string{"pubspec.yaml", "app/lib/a.dart"},
			"app", true, "",
		},
		{
			"pubspec inside the entry dir is checked",
			[]string{"app/pubspec.yaml"},
			"app", true,
			"pubspec.yaml in app has no " + configPath,
		},
		{
			"sibling dir sharing the prefix is not inside",
			[]string{"app2/pubspec.yaml"},
			"app", true, "",
		},
		{
			"without a listing the entry dir pubspec is checked",
			[]string{"pubspec.yaml"},
			"", false,
			"pubspec.yaml in . has no " + configPath,
		},
		{
			"without a listing and without a pubspec",
			[]string{"lib/a.dart"},
			"", false, "",
		},
	}
}

func TestDartPrecheck(t *testing.T) {
	t.Parallel()

	for _, testCase := range precheckCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, listed := buildTree(t, testCase.files)
			if !testCase.useList {
				listed = nil
			}
			dir := filepath.Join(root, testCase.dir)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			got := dartPrecheck(root, dir, listed)
			if !strings.HasPrefix(got, testCase.want) || (testCase.want == "") != (got == "") {
				t.Errorf("dartPrecheck = %q, want prefix %q", got, testCase.want)
			}
		})
	}
}

func TestDartPrecheckConfigIsADirectory(t *testing.T) {
	t.Parallel()

	root, listed := buildTree(t, []string{"pubspec.yaml"})
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(configPath)), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := dartPrecheck(root, root, listed); !strings.Contains(got, "has no") {
		t.Errorf("dartPrecheck = %q, want a missing-config reason", got)
	}
}

func noticeOf(missing, dropped string) string {
	return "generated code is missing (" + missing +
		"); run `dart run build_runner build`; " + dropped + " dropped"
}

func codegenDiagnostics() map[string]diagnostic {
	missing := func(code, uri string) diagnostic {
		return diagnostic{
			code: code, severity: core.SeverityError, file: "lib/x.dart",
			message: "Target of URI hasn't been generated: '" + uri + "'.",
		}
	}
	return map[string]diagnostic{
		"notGenerated": missing(codegenMissing, "a.g.dart"),
		"otherFile":    missing(codegenMissing, "b.g.dart"),
		"absentMock":   missing(codeUnresolved, "a.mocks.dart"),
		"absentPlain":  missing(codeUnresolved, "a.dart"),
		"warning":      {code: "w", severity: core.SeverityWarning},
		"failure":      {code: "undefined_identifier", severity: core.SeverityError},
	}
}

type codegenCase struct {
	name       string
	found      []diagnostic
	want       []diagnostic
	wantNotice string
}

func codegenRows() []codegenCase {
	byName := codegenDiagnostics()
	warning, failure := byName["warning"], byName["failure"]
	hint := diagnostic{code: "i", severity: core.SeverityInfo}
	return []codegenCase{
		{"none", []diagnostic{failure, warning}, []diagnostic{failure, warning}, ""},
		{"empty", nil, nil, ""},
		{
			"errors dropped, lints kept",
			[]diagnostic{warning, byName["notGenerated"], failure, hint},
			[]diagnostic{warning, hint},
			noticeOf("1 file: lib/a.g.dart", "2 derived errors"),
		},
		{
			"distinct files are counted once each",
			[]diagnostic{byName["notGenerated"], byName["notGenerated"], byName["otherFile"]},
			[]diagnostic{},
			noticeOf("2 files: lib/a.g.dart", "3 derived errors"),
		},
		{
			"unresolved generated suffix counts",
			[]diagnostic{byName["absentMock"], failure},
			[]diagnostic{},
			noticeOf("1 file: lib/a.mocks.dart", "2 derived errors"),
		},
		{
			"unresolved plain file is a real error",
			[]diagnostic{byName["absentPlain"], failure},
			[]diagnostic{byName["absentPlain"], failure},
			"",
		},
	}
}

func TestDartCodegen(t *testing.T) {
	t.Parallel()

	for _, testCase := range codegenRows() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, notice := dartCodegen(testCase.found, scope{root: "/r", dir: "/r"})
			if notice != testCase.wantNotice {
				t.Errorf("notice = %q, want %q", notice, testCase.wantNotice)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("kept = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

type preconditionCase struct {
	name     string
	files    []string
	sources  map[string]string
	dir      string
	report   string
	exit     int
	wantIDs  []string
	wantRun  bool
	wantText string
}

const (
	unavailableID = entryID + "/unavailable"
	derivedErr    = `{"code":"undefined_identifier","severity":"ERROR",` +
		`"location":{"file":"lib/a.dart"}}`
	lintInfo = `{"code":"lint","severity":"INFO","location":{"file":"lib/a.dart"}}`
)

func wrapReport(items ...string) string {
	return `{"version":1,"diagnostics":[` + strings.Join(items, ",") + `]}`
}

func skipCases() []preconditionCase {
	withConfig := []string{"pubspec.yaml", configPath, "lib/a.dart", "m/pubspec.yaml"}
	noRootConfig := []string{"pubspec.yaml", "lib/a.dart", "m/pubspec.yaml", "m/" + configPath}
	return []preconditionCase{
		{
			name: "root without config is skipped", files: noRootConfig,
			report: wrapReport(lintInfo), exit: 1, wantIDs: []string{unavailableID},
			wantText: "; " + entryID + " was skipped",
		},
		{
			name:   "entry dir at the root with only a member pubspec is skipped",
			files:  []string{"packages/a/pubspec.yaml", "lib/a.dart"},
			report: wrapReport(lintInfo), exit: 1, wantIDs: []string{unavailableID},
			wantText: "; " + entryID + " was skipped",
		},
		{
			name: "root config covers the member", files: withConfig,
			report: wrapReport(lintInfo), exit: 1,
			wantIDs: []string{entryID + "/lint"}, wantRun: true,
		},
		{
			name: "no pubspec runs", files: []string{"lib/a.dart"},
			report: wrapReport(lintInfo), exit: 1,
			wantIDs: []string{entryID + "/lint"}, wantRun: true,
		},
	}
}

func codegenCases() []preconditionCase {
	const generatedErr = `{"code":"uri_has_not_been_generated","severity":"ERROR",` +
		`"problemMessage":"x","location":{"file":"lib/a.dart"}}`
	withConfig := []string{"pubspec.yaml", configPath, "lib/a.dart", "m/pubspec.yaml"}
	return []preconditionCase{
		{
			name: "missing codegen drops errors, keeps lints", files: withConfig,
			report: wrapReport(generatedErr, derivedErr, lintInfo), exit: 3,
			wantIDs: []string{entryID + "/lint", unavailableID}, wantRun: true,
		},
		{
			name: "all dropped under exit 3 is no failure", files: withConfig,
			report: wrapReport(generatedErr, derivedErr), exit: 3,
			wantIDs: []string{unavailableID}, wantRun: true,
		},
		{
			name: "errors without codegen are kept", files: withConfig,
			report: wrapReport(derivedErr), exit: 3,
			wantIDs: []string{entryID + "/undefined_identifier"}, wantRun: true,
		},
	}
}

func workspaceCases() []preconditionCase {
	local := []string{
		"packages/fiacao/pubspec.yaml", "packages/fiacao/" + configPath,
		"packages/fiacao/lib/a.dart", "packages/other/lib/o.dart",
	}
	files := []string{
		"pubspec.yaml", configPath, "packages/fiacao/lib/a.dart", "packages/fiacao/pubspec.yaml",
		"packages/nucleon/lib/nucleon.dart",
	}
	sources := map[string]string{
		"packages/nucleon/lib/nucleon.dart": "export 'src/catalog.g.dart';\n",
	}
	generated := map[string]string{
		"packages/nucleon/lib/nucleon.dart":       "export 'src/catalog.g.dart';\n",
		"packages/nucleon/lib/src/catalog.g.dart": "",
	}
	return []preconditionCase{
		{
			name: "codegen missing outside the entry workspace is ignored", files: local,
			sources: map[string]string{"packages/other/lib/o.dart": "export 'x.g.dart';\n"},
			dir:     "packages/fiacao", report: wrapReport(derivedErr, lintInfo), exit: 3,
			wantIDs: []string{entryID + "/undefined_identifier", entryID + "/lint"},
			wantRun: true,
		},
		{
			name: "sibling package codegen missing drops the entry errors", files: files,
			sources: sources, dir: "packages/fiacao",
			report: wrapReport(derivedErr, lintInfo), exit: 3,
			wantIDs: []string{entryID + "/lint", unavailableID}, wantRun: true,
			wantText: "(1 file: packages/nucleon/lib/src/catalog.g.dart)",
		},
		{
			name: "sibling package codegen present keeps the errors", files: files,
			sources: generated, dir: "packages/fiacao",
			report: wrapReport(derivedErr, lintInfo), exit: 3,
			wantIDs: []string{entryID + "/undefined_identifier", entryID + "/lint"},
			wantRun: true,
		},
	}
}

func TestRunEntryPrecondition(t *testing.T) {
	t.Parallel()

	testCases := skipCases()
	testCases = append(testCases, codegenCases()...)
	testCases = append(testCases, workspaceCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runPreconditionCase(t, testCase)
		})
	}
}

func writeSources(t *testing.T, root string, sources map[string]string) {
	t.Helper()
	for name, text := range sources {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(text), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func runPreconditionCase(t *testing.T, testCase preconditionCase) {
	t.Helper()
	root, listed := buildTree(t, testCase.files)
	writeSources(t, root, testCase.sources)
	marker := filepath.Join(t.TempDir(), "ran")
	script := fmt.Sprintf("echo \"$@\" > %q\nprintf '%%s' '%s'\nexit %d\n",
		marker, testCase.report, testCase.exit)
	entry := core.DelegatedCommand{
		ID: entryID, Format: "dart-analyze-json", Dir: testCase.dir,
		Run: append(writeScript(t, script), "analyze"),
	}
	findings := NewAnalyzer(newIndex(t)).runEntry(root, listed, entry)
	ids := make([]string, len(findings))
	text := ""
	for idx, finding := range findings {
		ids[idx] = finding.AnalyzerID
		if finding.AnalyzerID == unavailableID {
			text = finding.Message
			if finding.Severity != core.SeverityInfo {
				t.Errorf("unavailable severity = %v, want info", finding.Severity)
			}
		}
	}
	if !slices.Equal(ids, testCase.wantIDs) {
		t.Fatalf("ids = %v, want %v", ids, testCase.wantIDs)
	}
	if !strings.Contains(text, testCase.wantText) {
		t.Errorf("unavailable message = %q, want it to contain %q", text, testCase.wantText)
	}
	if _, err := os.Stat(marker); (err == nil) != testCase.wantRun {
		t.Errorf("command ran = %v, want %v", err == nil, testCase.wantRun)
	}
}
