package dartsyntax

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestMain(m *testing.M) {
	os.Exit(runWithSharedCache(m))
}

func runWithSharedCache(m *testing.M) int {
	cache, err := os.MkdirTemp("", "dartsyntax-cache-")
	if err != nil {
		return 1
	}
	defer func() { _ = os.RemoveAll(cache) }()
	if setErr := os.Setenv("XDG_CACHE_HOME", cache); setErr != nil {
		return 1
	}
	if dart, lookErr := exec.LookPath("dart"); lookErr == nil {
		_, _ = ensureHelper(context.Background(), cache, dart)
	}
	return m.Run()
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", "integration", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

func TestProviderIdentity(t *testing.T) {
	t.Parallel()
	provider := New()
	if provider.buildBound != BuildTimeout {
		t.Errorf("buildBound = %v, want %v", provider.buildBound, BuildTimeout)
	}
	if provider.Language() != syntax.LangDart {
		t.Errorf("Language = %q", provider.Language())
	}
	for ext, want := range map[string]bool{".dart": true, ".go": false, ".svelte": false} {
		if provider.Handles(ext) != want {
			t.Errorf("Handles(%q) = %v, want %v", ext, !want, want)
		}
	}
}

func TestSummarizeWithoutDart(t *testing.T) {
	read := func(string) ([]byte, bool) { return []byte("void main() {}"), true }
	testCases := []struct {
		name    string
		sources []syntax.Source
		wantErr string
	}{
		{name: "no sources needs no dart"},
		{
			name:    "missing dart names the step",
			sources: []syntax.Source{{Path: "lib/a.dart"}},
			wantErr: "locate dart",
		},
		{
			name:    "missing dart mentions fvm",
			sources: []syntax.Source{{Path: "lib/a.dart"}},
			wantErr: "fvm users need one",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			files, err := New().Summarize(context.Background(), syntax.Corpus{
				Sources: testCase.sources,
				Read:    read,
			})
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil || len(files) != 0 {
				t.Errorf("Summarize = %v, %v", files, err)
			}
		})
	}
}

func summarizeInline(t *testing.T, contents map[string]string) map[string]*syntax.File {
	t.Helper()
	skipWhereXDGIsIgnored(t)
	sources := make([]syntax.Source, 0, len(contents))
	for path := range contents {
		sources = append(sources, syntax.Source{
			Path: path, IsTest: strings.HasSuffix(path, "_test.dart"),
		})
	}
	read := func(path string) ([]byte, bool) {
		content, ok := contents[path]
		return []byte(content), ok
	}
	files, err := New().Summarize(
		context.Background(), syntax.Corpus{Sources: sources, Read: read},
	)
	if err != nil {
		for _, unusable := range []string{"dart pub get", "Dart SDK 3.11+ required"} {
			if strings.Contains(err.Error(), unusable) {
				t.Skipf("dart helper cannot be built here: %v", err)
			}
		}
		t.Fatalf("Summarize: %v", err)
	}
	return files
}

func functionNamed(t *testing.T, file *syntax.File, name string) syntax.Function {
	t.Helper()
	for _, function := range file.Functions {
		if function.Name == name {
			return function
		}
	}
	t.Fatalf("no function %q in %+v", name, file.Functions)
	return syntax.Function{}
}

func TestSummarizeIntegration(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("dart"); err != nil {
		t.Skip("dart is not installed")
	}
	files := summarizeInline(t, map[string]string{
		"lib/service.dart":       fixture(t, "service.dart"),
		"lib/flow.dart":          fixture(t, "flow.dart"),
		"test/service_test.dart": fixture(t, "service_test.dart"),
	})
	testCases := []struct {
		name  string
		path  string
		check func(*testing.T, *syntax.File)
	}{
		{"signatures", "lib/service.dart", checkSignatures},
		{"members and closures", "lib/service.dart", checkMembers},
		{"imports and docs", "lib/service.dart", checkImportsAndDocs},
		{"complexity", "lib/flow.dart", checkComplexity},
		{"blocks and names", "lib/flow.dart", checkBlocksAndNames},
		{"test containers", "test/service_test.dart", checkTestContainers},
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

func checkSignatures(t *testing.T, file *syntax.File) {
	t.Helper()
	add := functionNamed(t, file, "add")
	if add.Kind != syntax.FuncFunction || add.Params != 2 || add.NamedParams != 0 {
		t.Errorf("add = %+v", add)
	}
	if add.Line != 6 || add.BodyStart != 6 || add.BodyEnd != 6 {
		t.Errorf("add lines = %+v", add)
	}
	configure := functionNamed(t, file, "configure")
	if configure.Params != 1 || configure.NamedParams != 2 || configure.BodyEnd != 8 {
		t.Errorf("configure = %+v", configure)
	}
}

func checkMembers(t *testing.T, file *syntax.File) {
	t.Helper()
	want := map[string]struct {
		kind   syntax.FunctionKind
		nested bool
	}{
		"run":          {syntax.FuncMethod, false},
		"Service":      {syntax.FuncConstructor, false},
		"Service.make": {syntax.FuncConstructor, false},
		"doubled":      {syntax.FuncAccessor, false},
		"callback":     {syntax.FuncClosure, false},
		"closure":      {syntax.FuncClosure, true},
	}
	for name, expected := range want {
		function := functionNamed(t, file, name)
		if function.Kind != expected.kind || function.Nested != expected.nested {
			t.Errorf("%s = %+v, want %+v", name, function, expected)
		}
	}
	for _, function := range file.Functions {
		if function.Name == "Service.named" {
			t.Errorf("bodiless constructor reported: %+v", function)
		}
	}
}

func checkImportsAndDocs(t *testing.T, file *syntax.File) {
	t.Helper()
	groups := []int{2, 1, 0}
	if len(file.Imports) != 3 {
		t.Fatalf("imports = %+v", file.Imports)
	}
	for index, want := range groups {
		if file.Imports[index].Group != want || file.Imports[index].Line != index+1 {
			t.Errorf("import %d = %+v, want group %d", index, file.Imports[index], want)
		}
	}
	if file.Imports[1].Literal != "'package:foo/foo.dart'" {
		t.Errorf("literal = %q", file.Imports[1].Literal)
	}
	if strings.Join(file.ImportGroups, ",") != "dart:,package:,relative" {
		t.Errorf("import groups = %v", file.ImportGroups)
	}
	want := syntax.DocSite{Symbol: "add", Kind: syntax.DocFunction, Line: 5}
	if len(file.DocSites) != 1 || file.DocSites[0] != want {
		t.Errorf("doc sites = %+v, want %+v", file.DocSites, want)
	}
}

func checkComplexity(t *testing.T, file *syntax.File) {
	t.Helper()
	flow := functionNamed(t, file, "flow")
	const want = 9
	if flow.Complexity != want || flow.Params != 2 || flow.Line != 1 || flow.BodyEnd != 25 {
		t.Errorf("flow = %+v, want complexity %d", flow, want)
	}
}

func checkBlocksAndNames(t *testing.T, file *syntax.File) {
	t.Helper()
	wantBlocks := []syntax.Block{
		{Line: 4, Depth: 1},
		{Line: 5, Depth: 2},
		{Line: 6, Depth: 3},
		{Line: 11, Depth: 1},
		{Line: 19, Depth: 1},
		{Line: 21, Depth: 1},
	}
	if len(file.Blocks) != len(wantBlocks) {
		t.Fatalf("blocks = %+v", file.Blocks)
	}
	for index, want := range wantBlocks {
		if file.Blocks[index] != want {
			t.Errorf("block %d = %+v, want %+v", index, file.Blocks[index], want)
		}
	}
	wantNames := map[string]syntax.NameKind{
		"z": syntax.NameVariable, "e": syntax.NameCatchParameter,
		"index": syntax.NameLoopVariable, "count": syntax.NameParameter,
		"flow": syntax.NameFunction,
	}
	for _, name := range file.Names {
		if kind, ok := wantNames[name.Name]; ok && name.Kind != kind {
			t.Errorf("name %s kind = %q, want %q", name.Name, name.Kind, kind)
		}
		delete(wantNames, name.Name)
	}
	if len(wantNames) != 0 {
		t.Errorf("names missing: %v", wantNames)
	}
}

func checkTestContainers(t *testing.T, file *syntax.File) {
	t.Helper()
	containers := 0
	for _, function := range file.Functions {
		if function.TestContainer {
			containers++
		}
	}
	const want = 4
	if containers != want {
		t.Errorf("test containers = %d, want %d: %+v", containers, want, file.Functions)
	}
	if plain := functionNamed(t, file, "helper"); plain.TestContainer {
		t.Errorf("plain function flagged as container: %+v", plain)
	}
}

func skipWhereXDGIsIgnored(tb testing.TB) {
	tb.Helper()
	if runtime.GOOS == "darwin" {
		tb.Skip("os.UserCacheDir ignores XDG_CACHE_HOME on darwin, so the real cache is used")
	}
}

func TestProviderPrepare(t *testing.T) {
	skipWhereXDGIsIgnored(t)
	dart := writeFakeDart(t)
	t.Setenv("PATH", filepath.Dir(dart)+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	testCases := []struct {
		name     string
		provider *Provider
	}{
		{name: "XDG override for the default base", provider: New()},
		{name: "zero-value provider", provider: &Provider{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("XDG_CACHE_HOME", dir)
			exe, err := testCase.provider.prepare(context.Background())
			if err != nil {
				t.Fatalf("prepare: %v", err)
			}
			if root := cacheRoot(dir); filepath.Dir(filepath.Dir(exe)) != root {
				t.Errorf("exe = %q, want under %q", exe, root)
			}
		})
	}
}

func TestProviderBuildBound(t *testing.T) {
	skipWhereXDGIsIgnored(t)
	dart := writeFakeDart(t)
	t.Setenv("PATH", filepath.Dir(dart)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_FAIL", "hang")
	base := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", base)
	_, err := (&Provider{buildBound: 300 * time.Millisecond}).prepare(context.Background())
	if err == nil {
		t.Fatal("prepare succeeded with a hanging dart")
	}
	if _, marked := loadMarker(currentDir(t, base)); !marked {
		t.Errorf("no failure marker after the build bound expired: %v", err)
	}
}

func TestSummarizeReadOnlyCache(t *testing.T) {
	skipWhereXDGIsIgnored(t)
	dart := writeFakeDart(t)
	t.Setenv("PATH", filepath.Dir(dart)+string(os.PathListSeparator)+os.Getenv("PATH"))
	tempDir, base := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CACHE_HOME", base)
	capture := filepath.Join(t.TempDir(), "request-path")
	t.Setenv("TMPDIR", tempDir)
	entry := currentDir(t, base)
	exe := filepath.Join(entry, exeName())
	script := "#!/bin/sh\necho \"$1\" > " + capture + "\necho '{\"lib/a.dart\": {}}'\n"
	if err := os.MkdirAll(entry, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	lockDir(t, entry)
	lockDir(t, filepath.Dir(entry))
	read := func(string) ([]byte, bool) { return []byte("void main() {}"), true }
	sources := []syntax.Source{{Path: "lib/a.dart"}}
	files, err := New().Summarize(
		context.Background(),
		syntax.Corpus{Sources: sources, Read: read},
	)
	if err != nil || files["lib/a.dart"] == nil {
		t.Fatalf("Summarize = %v, %v", files, err)
	}
	sent, readErr := os.ReadFile(capture)
	requestPath := strings.TrimSpace(string(sent))
	if readErr != nil || filepath.Dir(requestPath) != tempDir ||
		!strings.HasPrefix(filepath.Base(requestPath), "pulsarules-dart-request-") {
		t.Errorf("request file = %q, %v; want it under TMPDIR %q", sent, readErr, tempDir)
	}
}
