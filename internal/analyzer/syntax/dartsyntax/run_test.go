package dartsyntax

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestCollectRequests(t *testing.T) {
	t.Parallel()
	sources := []syntax.Source{
		{Path: "a.dart", Content: []byte("body of a.dart")},
		{Path: "a_test.dart", IsTest: true, Content: []byte("body of a_test.dart")},
	}
	want := []request{
		{Path: "a.dart", Content: "body of a.dart"},
		{Path: "a_test.dart", Content: "body of a_test.dart", Test: true},
	}
	if got := collectRequests(sources); !reflect.DeepEqual(got, want) {
		t.Errorf("collectRequests = %+v, want %+v", got, want)
	}
}

func TestDecodeFiles(t *testing.T) {
	t.Parallel()
	requests := []request{{Path: "lib/a.dart"}, {Path: "lib/b.dart"}, {Path: "lib/c.dart"}}
	const raw = `{
	  "lib/a.dart": {"functions": [{"name": "run", "kind": "method", "line": 3,
	    "bodyStart": 3, "bodyEnd": 9, "params": 2, "namedParams": 1, "complexity": 4,
	    "testContainer": false, "nested": true}],
	   "blocks": [{"line": 4, "depth": 2}],
	   "imports": [{"literal": "'dart:io'", "line": 1, "group": 0}],
	   "importGroups": ["dart:", "package:", "relative"],
	   "names": [{"name": "run", "kind": "method", "line": 3, "exported": true}],
	   "docSites": [{"symbol": "run", "kind": "method", "line": 2, "mixedDirective": true}],
	   "parseError": ""},
	  "lib/b.dart": {"parseError": "line 1: Expected a token"},
	  "lib/other.dart": {}
	}`
	testCases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "decodes by requested path", raw: raw},
		{name: "invalid json", raw: "not json", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			files, err := decodeFiles([]byte(testCase.raw), requests)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("decodeFiles error = %v, wantErr %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			assertDecoded(t, files)
		})
	}
}

func assertDecoded(t *testing.T, files map[string]*syntax.File) {
	t.Helper()
	if len(files) != 2 || files["lib/c.dart"] != nil || files["lib/other.dart"] != nil {
		t.Fatalf("unexpected keys: %v", files)
	}
	assertFirst(t, files["lib/a.dart"])
	if got := files["lib/b.dart"].ParseError; got != "line 1: Expected a token" {
		t.Errorf("parse error = %q", got)
	}
}

func assertFirst(t *testing.T, first *syntax.File) {
	t.Helper()
	wantFunc := syntax.Function{
		Name: "run", Kind: syntax.FuncMethod, Line: 3, BodyStart: 3, BodyEnd: 9,
		Params: 2, NamedParams: 1, Complexity: 4, Nested: true,
	}
	if !reflect.DeepEqual(first.Functions, []syntax.Function{wantFunc}) {
		t.Errorf("functions = %+v", first.Functions)
	}
	if !reflect.DeepEqual(first.Blocks, []syntax.Block{{Line: 4, Depth: 2}}) {
		t.Errorf("blocks = %+v", first.Blocks)
	}
	assertFirstRest(t, first)
}

func assertFirstRest(t *testing.T, first *syntax.File) {
	t.Helper()
	wantImports := []syntax.Import{{Literal: "'dart:io'", Line: 1, Group: 0}}
	if !reflect.DeepEqual(first.Imports, wantImports) {
		t.Errorf("imports = %+v", first.Imports)
	}
	wantNames := []syntax.Name{
		{Name: "run", Kind: syntax.NameMethod, Line: 3, Exported: true},
	}
	if !reflect.DeepEqual(first.Names, wantNames) {
		t.Errorf("names = %+v", first.Names)
	}
	wantGroups := []string{"dart:", "package:", "relative"}
	if !reflect.DeepEqual(first.ImportGroups, wantGroups) {
		t.Errorf("import groups = %v", first.ImportGroups)
	}
	wantDocs := []syntax.DocSite{
		{Symbol: "run", Kind: syntax.DocMethod, Line: 2, MixedDirective: true},
	}
	if !reflect.DeepEqual(first.DocSites, wantDocs) {
		t.Errorf("docSites = %+v", first.DocSites)
	}
}

func writeFakeHelper(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fake helper needs a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func runCases(t *testing.T) []runCase {
	t.Helper()
	const swapForDir = "rm \"$1\"\nmkdir \"$1\"\ntouch \"$1/x\"\n"
	const consume = "cat \"$1\" > {capture}\nrm \"$1\"\n"
	testCases := []runCase{
		{
			name:   "reads the request file",
			script: "cat \"$1\" > {capture}\necho '{\"lib/a.dart\": {}}'\n",
		},
		{name: "helper exits non-zero", script: "echo oops >&2\nexit 3\n", wantErr: "oops"},
		{
			name:    "helper exits non-zero and swaps the request for a directory",
			script:  swapForDir + "echo oops >&2\nexit 3\n",
			wantErr: "remove helper request", keeps: true,
		},
		{
			name:    "helper swaps the request for a directory, output is fine",
			script:  swapForDir + "echo '{}'\n",
			wantErr: "remove helper request", keeps: true,
		},
		{
			name:   "helper exits non-zero after consuming the request",
			script: consume + "exit 3\n", wantErr: "exit status 3",
		},
		{
			name:   "helper consumes the request, output is fine",
			script: consume + "echo '{\"lib/a.dart\": {}}'\n",
		},
		{name: "helper prints garbage", script: "echo nope\n", wantErr: "decode"},
		{
			name: "work dir missing", script: "echo '{}'\n", missing: true,
			wantErr: "create helper request",
		},
		{
			name: "deadline kills a surviving child", script: "sleep 30 &\nsleep 30\n",
			timeout: time.Second, wantErr: context.DeadlineExceeded.Error(),
		},
		{
			name: "deadline outlasts a detached child", script: "setsid sleep 30 &\nsleep 30\n",
			timeout: time.Second, wantErr: context.DeadlineExceeded.Error(), needs: "setsid",
		},
	}
	for index := range testCases {
		testCases[index].provision(t)
	}
	return testCases
}

func (testCase *runCase) provision(t *testing.T) {
	t.Helper()
	testCase.capture = filepath.Join(t.TempDir(), "request.json")
	script := strings.ReplaceAll(testCase.script, "{capture}", testCase.capture)
	testCase.exe = writeFakeHelper(t, script)
	testCase.workDir = t.TempDir()
}

func TestRunHelper(t *testing.T) {
	requests := []request{{Path: "lib/a.dart", Content: "int x = 1;", Test: true}}
	testCases := runCases(t)
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := exec.LookPath(testCase.needs); testCase.needs != "" && err != nil {
				t.Skipf("%s is not installed", testCase.needs)
			}
			testCase.run(t, requests)
		})
	}
}

type runCase struct {
	name    string
	script  string
	wantErr string
	timeout time.Duration
	missing bool
	keeps   bool
	capture string
	needs   string
	exe     string
	workDir string
}

func (testCase runCase) run(t *testing.T, requests []request) {
	t.Helper()
	const budget = 10 * time.Second
	ctx := context.Background()
	if testCase.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, testCase.timeout)
		defer cancel()
	}
	workDir := testCase.workDir
	if testCase.missing {
		workDir = filepath.Join(workDir, "absent")
	}
	start := time.Now()
	files, err := runHelper(ctx, testCase.exe, workDir, requests)
	if elapsed := time.Since(start); elapsed > budget {
		t.Errorf("runHelper took %v, want under %v", elapsed, budget)
	}
	if testCase.wantErr != "" {
		testCase.assertFailed(t, err, workDir)
		return
	}
	if err != nil || files["lib/a.dart"] == nil {
		t.Fatalf("runHelper = %v, %v", files, err)
	}
	assertSentAndCleaned(t, testCase.capture, workDir)
}

func (testCase runCase) assertFailed(t *testing.T, err error, workDir string) {
	t.Helper()
	assertErrContains(t, err, testCase.wantErr)
	if !testCase.missing && !testCase.keeps {
		assertCleaned(t, workDir)
		if strings.Contains(err.Error(), "remove helper request") {
			t.Errorf("a consumed request must not fail the removal: %v", err)
		}
	}
	if count := strings.Count(err.Error(), context.DeadlineExceeded.Error()); count > 1 {
		t.Errorf("deadline reported %d times in %v", count, err)
	}
}

func assertSentAndCleaned(t *testing.T, capture, workDir string) {
	t.Helper()
	sent, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"content":"int x = 1;"`, `"test":true`} {
		if !strings.Contains(string(sent), want) {
			t.Errorf("request file = %q, want containing %s", sent, want)
		}
	}
	assertCleaned(t, workDir)
}

func assertCleaned(t *testing.T, workDir string) {
	t.Helper()
	if left, _ := os.ReadDir(workDir); len(left) != 0 {
		t.Errorf("request file was not cleaned up: %v", left)
	}
}

type failingWriter struct{}

func (failingWriter) WriteTo(io.Writer) (int64, error) { return 0, errors.New("disk full") }

func TestWriteRequest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		payload io.WriterTo
		wantErr string
	}{
		{name: "writes the payload", payload: strings.NewReader("[]")},
		{name: "failed write is cleaned up", payload: failingWriter{}, wantErr: "disk full"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path, err := writeRequest(dir, testCase.payload)
			assertErrContains(t, err, testCase.wantErr)
			left, _ := os.ReadDir(dir)
			wantKept := testCase.wantErr == ""
			if kept := len(left) == 1; kept != wantKept || (path != "") != wantKept {
				t.Errorf("path = %q, dir = %v; want kept %v", path, left, wantKept)
			}
		})
	}
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}
