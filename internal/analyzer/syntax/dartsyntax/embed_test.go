package dartsyntax

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func semanticsContents(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("testdata", "semantics"))
	if err != nil {
		t.Fatal(err)
	}
	contents := make(map[string]string, len(entries))
	for _, entry := range entries {
		source, readErr := os.ReadFile(filepath.Join("testdata", "semantics", entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		folder := "lib"
		if strings.HasSuffix(entry.Name(), "_test.dart") {
			folder = "test"
		}
		contents[folder+"/"+entry.Name()] = string(source)
	}
	return contents
}

func TestHelperSemantics(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("dart is not installed")
	}
	files := summarizeInline(t, semanticsContents(t))
	testCases := []struct {
		name  string
		path  string
		check func(*testing.T, *syntax.File)
	}{
		{"loops and collection elements", "lib/loops.dart", checkLoops},
		{"switch defaults and guards", "lib/switches.dart", checkSwitches},
		{"else-if and case bodies", "lib/deep.dart", checkDeep},
		{"scopes feed only unbound closures", "lib/scopes.dart", checkScopes},
		{"test container bodies", "test/scopes_test.dart", checkContainerBlocks},
		{"owners, parameters, aliases, labels", "lib/owners.dart", checkNames},
		{"doc shapes", "lib/docs.dart", checkDocShapes},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file := files[testCase.path]
			if file == nil || file.ParseError != "" {
				t.Fatalf("file = %+v", file)
			}
			testCase.check(t, file)
		})
	}
}

func assertComplexity(t *testing.T, file *syntax.File, want map[string]int) {
	t.Helper()
	for name, complexity := range want {
		if got := functionNamed(t, file, name).Complexity; got != complexity {
			t.Errorf("%s complexity = %d, want %d", name, got, complexity)
		}
	}
}

func assertBlocks(t *testing.T, file *syntax.File, want []syntax.Block) {
	t.Helper()
	if !reflect.DeepEqual(file.Blocks, want) {
		t.Errorf("blocks = %+v, want %+v", file.Blocks, want)
	}
}

func checkLoops(t *testing.T, file *syntax.File) {
	t.Helper()
	assertComplexity(t, file, map[string]int{"loops": 2, "spread": 3})
}

func checkSwitches(t *testing.T, file *syntax.File) {
	t.Helper()
	assertComplexity(t, file, map[string]int{"kind": 2, "arms": 4, "typedCase": 2, "typedArm": 2})
}

func checkDeep(t *testing.T, file *syntax.File) {
	t.Helper()
	assertBlocks(t, file, []syntax.Block{
		{Line: 2, Depth: 1},
		{Line: 4, Depth: 1},
		{Line: 6, Depth: 1},
		{Line: 9, Depth: 1},
		{Line: 11, Depth: 2},
	})
}

func checkScopes(t *testing.T, file *syntax.File) {
	t.Helper()
	assertComplexity(t, file, map[string]int{
		"outer": 3, "local": 2, "bound": 2, "closure": 3, "main": 2,
	})
	if functionNamed(t, file, "main").TestContainer {
		t.Error("main of a non-test file is a test container")
	}
	assertBlocks(t, file, []syntax.Block{{Line: 3, Depth: 1}, {Line: 19, Depth: 1}})
}

func checkContainerBlocks(t *testing.T, file *syntax.File) {
	t.Helper()
	want := map[string]int{"main": 1}
	assertComplexity(t, file, want)
	var closures []int
	containers := 0
	for _, function := range file.Functions {
		if function.TestContainer {
			containers++
		}
		if function.Name == "closure" && function.TestContainer && function.Nested {
			closures = append(closures, function.Complexity)
		}
	}
	if containers != 3 {
		t.Errorf("test containers = %d, want 3: a call with a target is not one", containers)
	}
	if !reflect.DeepEqual(closures, []int{2, 2}) {
		t.Errorf("container closure complexities = %v, want [2 2]", closures)
	}
	assertBlocks(t, file, []syntax.Block{{Line: 8, Depth: 1}, {Line: 4, Depth: 1}})
}

func checkNames(t *testing.T, file *syntax.File) {
	t.Helper()
	exported := map[string]bool{
		"Public": true, "shown": true, "_hidden": false, "_Private": false,
		"buried": false, "top": true, "_topPrivate": false,
	}
	kinds := map[string]syntax.NameKind{
		"math": syntax.NameImportAlias, "outer": syntax.NameLabel,
		"plain": syntax.NameParameter, "value": syntax.NameField,
		"key": syntax.NameField, "index": syntax.NameLoopVariable,
	}
	seen := map[string]bool{}
	for _, name := range file.Names {
		seen[name.Name] = true
		assertName(t, name, exported, kinds)
	}
	for name := range kinds {
		if !seen[name] {
			t.Errorf("name %q was not collected", name)
		}
	}
}

func assertName(
	t *testing.T, name syntax.Name, exported map[string]bool, kinds map[string]syntax.NameKind,
) {
	t.Helper()
	if want, ok := exported[name.Name]; ok && name.Exported != want {
		t.Errorf("%s exported = %v, want %v", name.Name, name.Exported, want)
	}
	if want, ok := kinds[name.Name]; ok && name.Kind != want {
		t.Errorf("%s kind = %q, want %q", name.Name, name.Kind, want)
	}
	formal := name.Kind == syntax.NameParameter && name.Name != "plain"
	if name.Name == "label" || formal {
		t.Errorf("unexpected name %+v", name)
	}
}

func checkDocShapes(t *testing.T, file *syntax.File) {
	t.Helper()
	want := []syntax.DocSite{
		{Symbol: "mixed", Kind: syntax.DocFunction, Line: 1, MixedDirective: true},
		{Symbol: "old", Kind: syntax.DocFunction, Line: 5, Exempt: true},
		{Symbol: "plain", Kind: syntax.DocFunction, Line: 8},
	}
	if !reflect.DeepEqual(file.DocSites, want) {
		t.Errorf("doc sites = %+v, want %+v", file.DocSites, want)
	}
}

func realHelper(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("dart is not installed")
	}
	exe, err := New().helper.Exe(t.Context())
	if err != nil {
		t.Skipf("dart helper cannot be built here: %v", err)
	}
	return exe
}

func writeInput(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("request file %s survives the helper run: %v", path, err)
	}
}

func helperExit(t *testing.T, exe string, args []string) (int, string) {
	t.Helper()
	return helperExitTo(t, exe, nil, args)
}

func helperExitTo(t *testing.T, exe string, stdout io.Writer, args []string) (int, string) {
	t.Helper()
	var stderr strings.Builder
	command := exec.CommandContext(t.Context(), exe, args...)
	command.Stdout = stdout
	command.Stderr = &stderr
	err := command.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode(), strings.TrimSpace(stderr.String())
	}
	if err != nil {
		t.Fatalf("run helper: %v", err)
	}
	return 0, strings.TrimSpace(stderr.String())
}

type inputCase struct {
	name     string
	args     []string
	wantCode int
	wantErr  string
}

func inputCases(t *testing.T) []inputCase {
	t.Helper()
	dir := t.TempDir()
	valid := writeInput(t, dir, "valid.json",
		`[{"path":"a.dart","content":"void f() {}","test":false}]`)
	return []inputCase{
		{name: "valid request", args: []string{valid}},
		{name: "no argument", wantCode: 64, wantErr: "usage"},
		{name: "two arguments", args: []string{valid, valid}, wantCode: 64, wantErr: "usage"},
		{
			name: "missing file", args: []string{filepath.Join(dir, "absent.json")},
			wantCode: 66, wantErr: "cannot read",
		},
		{
			name:     "invalid utf-8",
			args:     []string{writeInput(t, dir, "latin1.json", "[\"caf\xe9\"]")},
			wantCode: 65, wantErr: "not valid UTF-8",
		},
		{
			name: "bad json", args: []string{writeInput(t, dir, "bad.json", "{")},
			wantCode: 65, wantErr: "not valid JSON",
		},
		{
			name: "not a list", args: []string{writeInput(t, dir, "map.json", "{}")},
			wantCode: 65, wantErr: "JSON list",
		},
		{
			name:     "entry without a path",
			args:     []string{writeInput(t, dir, "nopath.json", "[{}]")},
			wantCode: 65, wantErr: "JSON list",
		},
	}
}

func TestHelperInputs(t *testing.T) {
	t.Parallel()
	exe := realHelper(t)
	testCases := inputCases(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			code, message := helperExit(t, exe, testCase.args)
			if code != testCase.wantCode || !strings.Contains(message, testCase.wantErr) ||
				strings.Contains(message, "\n") || strings.Contains(message, "#0") {
				t.Errorf("exit %d, stderr %q; want exit %d, one line with %q",
					code, message, testCase.wantCode, testCase.wantErr)
			}
			if len(testCase.args) == 1 && testCase.wantCode != 66 {
				assertGone(t, testCase.args[0])
			}
		})
	}
}

func checkPathologicalFile(t *testing.T, exe string) {
	t.Helper()
	const operands = 20000
	chain := "var x = " + strings.Repeat("1 + ", operands) + "1;"
	requests := []request{
		{Path: "lib/chain.dart", Content: chain},
		{Path: "lib/normal.dart", Content: "void f() {}"},
	}
	files, err := runHelper(t.Context(), exe, t.TempDir(), requests)
	if err != nil {
		t.Fatalf("runHelper: %v", err)
	}
	const wantError = "helper failed: StackOverflowError"
	if got := files["lib/chain.dart"]; got == nil || got.ParseError != wantError {
		t.Errorf("chain file = %+v, want parse error %q", got, wantError)
	}
	normal := files["lib/normal.dart"]
	if normal == nil || normal.ParseError != "" || len(normal.Functions) != 1 {
		t.Errorf("normal file = %+v, want one function and no parse error", normal)
	}
}

func checkLastResort(t *testing.T, exe string) {
	t.Helper()
	input := writeInput(t, t.TempDir(), "valid.json",
		`[{"path":"a.dart","content":"void f() {}","test":false}]`)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	if closeErr := reader.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	code, message := helperExitTo(t, exe, writer, []string{input})
	if code != 70 || !strings.HasPrefix(message, "summary: internal error") ||
		strings.Contains(message, "\n") || strings.Contains(message, "#0") {
		t.Errorf("exit %d, stderr %q; want exit 70 and one internal error line", code, message)
	}
}

func TestHelperFailures(t *testing.T) {
	t.Parallel()
	exe := realHelper(t)
	testCases := []struct {
		name  string
		check func(*testing.T, string)
	}{
		{name: "pathological file spares its neighbours", check: checkPathologicalFile},
		{name: "unwritable stdout exits 70", check: checkLastResort},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.check(t, exe)
		})
	}
}
