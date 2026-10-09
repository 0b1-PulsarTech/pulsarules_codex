//go:build unix

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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

type preconditionCase struct {
	name     string
	files    []string
	dir      string
	report   string
	exit     int
	wantIDs  []string
	wantRun  bool
	wantText string
}

const (
	unavailableID = entryID + "/unavailable"
)

func wrapReport(items ...string) string {
	return `{"version":1,"diagnostics":[` + strings.Join(items, ",") + `]}`
}

func skipCases() []preconditionCase {
	const lintInfo = `{"code":"lint","severity":"INFO","location":{"file":"lib/a.dart"}}`
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

func TestRunEntryPrecondition(t *testing.T) {
	t.Parallel()

	testCases := skipCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runPreconditionCase(t, testCase)
		})
	}
}

func runPreconditionCase(t *testing.T, testCase preconditionCase) {
	t.Helper()
	root, listed := buildTree(t, testCase.files)
	marker := filepath.Join(t.TempDir(), "ran")
	script := fmt.Sprintf("echo \"$@\" > %q\nprintf '%%s' '%s'\nexit %d\n",
		marker, testCase.report, testCase.exit)
	entry := pass.DelegatedCommand{
		ID: entryID, Format: "dart-analyze-json", Dir: testCase.dir,
		Run: append(writeScript(t, script), "analyze"),
	}
	findings := NewAnalyzer(newIndex(t)).runEntry(root, listed, entry)
	ids := make([]string, len(findings))
	text := ""
	for idx, findingItem := range findings {
		ids[idx] = findingItem.AnalyzerID
		if findingItem.AnalyzerID == unavailableID {
			text = findingItem.Message
			if findingItem.Severity != finding.SeverityInfo {
				t.Errorf("unavailable severity = %v, want info", findingItem.Severity)
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
