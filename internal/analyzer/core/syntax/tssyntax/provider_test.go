package tssyntax

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsguard"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsproc"
)

func TestProviderHandles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want bool
	}{
		{ext: ".ts", want: true},
		{ext: ".tsx", want: true},
		{ext: ".mts", want: true},
		{ext: ".cts", want: true},
		{ext: ".js", want: true},
		{ext: ".jsx", want: true},
		{ext: ".mjs", want: true},
		{ext: ".cjs", want: true},
		{ext: ".svelte", want: true},
		{ext: ".TS", want: true},
		{ext: ".go", want: false},
		{ext: ".dart", want: false},
		{ext: ".json", want: false},
		{ext: ".d", want: false},
		{ext: "", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()

			if got := New().Handles(testCase.ext); got != testCase.want {
				t.Fatalf("handles %q = %v, want %v", testCase.ext, got, testCase.want)
			}
		})
	}
}

func TestProviderLanguage(t *testing.T) {
	t.Parallel()

	if got := New().Language(); got != syntax.LangTypeScript {
		t.Fatalf("language %q", got)
	}
}

func summarizeFixture(t *testing.T) map[string]*syntax.File {
	t.Helper()

	files := map[string]string{
		"src/a.ts":       "export function a() {}\n",
		"src/b.mjs":      "export const b = () => {};\n",
		"src/c.svelte":   "<script>function c() {}</script>\n",
		"src/broken.tsx": "const = ;\n",
	}
	read := func(path string) ([]byte, bool) {
		text, ok := files[path]
		return []byte(text), ok
	}
	sources := []syntax.Source{
		{Path: "src/a.ts"},
		{Path: "src/b.mjs"},
		{Path: "src/c.svelte"},
		{Path: "src/broken.tsx"},
		{Path: "src/gone.ts"},
	}
	got, err := New().Summarize(context.Background(), syntax.Corpus{Sources: sources, Read: read})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	return got
}

func TestProviderSummarize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path      string
		language  syntax.Language
		wantFns   int
		wantError string
	}{
		{path: "src/a.ts", language: syntax.LangTypeScript, wantFns: 1},
		{path: "src/b.mjs", language: syntax.LangJavaScript, wantFns: 1},
		{path: "src/c.svelte", language: syntax.LangSvelte, wantFns: 1},
		{path: "src/gone.ts", language: syntax.LangTypeScript, wantError: "file is not readable"},
	}
	got := summarizeFixture(t)
	if len(got) != len(testCases)+1 {
		t.Fatalf("%d summaries, want %d", len(got), len(testCases)+1)
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()

			file := got[testCase.path]
			if file == nil || file.Path != testCase.path || file.Language != testCase.language {
				t.Fatalf("summary = %+v, want language %s", file, testCase.language)
			}
			if file.ParseError != testCase.wantError || len(file.Functions) != testCase.wantFns {
				t.Fatalf("error %q functions %d", file.ParseError, len(file.Functions))
			}
		})
	}
}

func TestProviderSummarizeBroken(t *testing.T) {
	t.Parallel()

	file := summarizeFixture(t)["src/broken.tsx"]
	if file == nil || file.Language != syntax.LangTypeScript || file.ParseError == "" {
		t.Fatalf("broken file summary %+v", file)
	}
}

func TestProviderSummarizeCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	read := func(string) ([]byte, bool) { return []byte("let a;"), true }
	got, err := New().Summarize(ctx, syntax.Corpus{
		Sources: []syntax.Source{{Path: "a.ts"}},
		Read:    read,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v, want context.Canceled", err)
	}
	if len(got) != 0 {
		t.Fatalf("summaries after cancel: %v", got)
	}
}

func fileReader(root string) func(string) ([]byte, bool) {
	return func(path string) ([]byte, bool) {
		content, err := os.ReadFile(filepath.Join(root, path))
		return content, err == nil
	}
}

func TestProviderHostileShapes(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir("testdata/hostile")
	if err != nil {
		t.Fatalf("list hostile inputs: %v", err)
	}
	sources := make([]syntax.Source, 0, len(entries))
	for _, entry := range entries {
		sources = append(sources, syntax.Source{Path: entry.Name()})
	}
	got, err := newProvider(32<<20, 5*time.Second).Summarize(context.Background(), syntax.Corpus{
		Sources: sources,
		Read:    fileReader("testdata/hostile"),
	})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	for _, source := range sources {
		file := got[source.Path]
		deep := strings.ContainsAny(source.Path, "2") && !strings.Contains(source.Path, "_12")
		skipped := strings.HasPrefix(file.ParseError, "skipped: ")
		if skipped && !strings.Contains(file.ParseError, "limit") {
			t.Fatalf("%s: unexpected skip %q", source.Path, file.ParseError)
		}
		if deep && !skipped {
			t.Fatalf("%s: deep shape was not bounded: %q", source.Path, file.ParseError)
		}
	}
}

func TestProviderKeepsBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		text string
	}{
		{name: "non-utf8 path", path: "bad\xffname.ts", text: "export function h() {}\n"},
		{name: "latin-1 body", path: "latin1.ts", text: "export const caf\xe9 = \"\xff\xfe\";\n" +
			"export function f\xe9() { return 1 }\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			read := func(string) ([]byte, bool) { return []byte(testCase.text), true }
			got, err := New().Summarize(context.Background(), syntax.Corpus{
				Sources: []syntax.Source{{Path: testCase.path}},
				Read:    read,
			})
			if err != nil {
				t.Fatalf("summarize: %v", err)
			}
			want := handleRequest(tsproc.Request{Path: testCase.path, Text: testCase.text})
			if fmt.Sprintf("%+v", got[testCase.path]) != fmt.Sprintf("%+v", want) {
				t.Fatalf("child summary %+v, direct %+v", got[testCase.path], want)
			}
			if len(want.Names) == 0 {
				t.Fatalf("direct summary is not a real parse: %+v", want)
			}
		})
	}
}

func TestProviderClosesSession(t *testing.T) {
	t.Parallel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	pidFile := filepath.Join(t.TempDir(), "pid")
	provider := newProvider(memoryCeiling, fileWall)
	provider.runner.Exe = exe
	provider.runner.Args = []string{"probechild", pidFile}
	read := func(string) ([]byte, bool) { return []byte("let a;"), true }
	sources := []syntax.Source{{Path: "a.ts"}}
	got, err := provider.Summarize(
		context.Background(),
		syntax.Corpus{Sources: sources, Read: read},
	)
	if err != nil || got["a.ts"] == nil || got["a.ts"].ParseError != "" {
		t.Fatalf("summarize %+v, %v", got, err)
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("pid file: %v", err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		t.Fatalf("pid %q: %v", raw, err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("find %d: %v", pid, err)
	}
	if err = proc.Signal(syscall.Signal(0)); err == nil {
		_ = proc.Kill()
		t.Fatalf("parser process %d outlived Summarize", pid)
	}
}

func TestNewLimits(t *testing.T) {
	t.Parallel()

	runner := New().runner
	if memoryCeiling != 256<<20 || fileWall != 10*time.Second ||
		runner.Memory != memoryCeiling || runner.Wall != fileWall {
		t.Fatalf("limits %d bytes, %s", runner.Memory, runner.Wall)
	}
	if len(runner.Args) != 1 || runner.Args[0] != tsproc.ChildArg || runner.Exe != "" {
		t.Fatalf("command %q %v", runner.Exe, runner.Args)
	}
}

func TestProviderWallBound(t *testing.T) {
	t.Parallel()

	const name = "jsx_async_20.tsx"
	provider := newProvider(memoryCeiling, 500*time.Millisecond)
	got, err := provider.Summarize(context.Background(), syntax.Corpus{
		Sources: []syntax.Source{{Path: name}},
		Read:    fileReader("testdata/hostile"),
	})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if reason := got[name].ParseError; !strings.Contains(reason, "time limit") {
		t.Fatalf("reason %q", reason)
	}
}

func TestProviderSkips(t *testing.T) {
	t.Parallel()

	oversize := strings.Repeat("x", tsguard.MaxFileBytes+1)
	testCases := []struct {
		name     string
		provider *Provider
		path     string
		want     string
	}{
		{name: "oversize", provider: New(), path: "big.svelte", want: "byte limit"},
		{name: "unreadable", provider: New(), path: "gone.ts", want: "not readable"},
	}
	read := func(path string) ([]byte, bool) {
		if path == "big.svelte" {
			return []byte(oversize), true
		}
		return []byte("let a;"), path != "gone.ts"
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := testCase.provider.Summarize(
				context.Background(),
				syntax.Corpus{Sources: []syntax.Source{{Path: testCase.path}}, Read: read},
			)
			file := got[testCase.path]
			if err != nil || file == nil || !strings.Contains(file.ParseError, testCase.want) {
				t.Fatalf("summary %+v, %v", file, err)
			}
			if file.Language == "" {
				t.Fatal("skipped file lost its language")
			}
		})
	}
}

func TestProviderChildCannotStart(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		paths []string
	}{
		{name: "one file", paths: []string{"a.ts"}},
		{name: "many files", paths: []string{"a.ts", "b.mjs", "c.svelte"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			provider := New()
			provider.runner.Exe = "/nonexistent/parser"
			sources := make([]syntax.Source, 0, len(testCase.paths))
			for _, path := range testCase.paths {
				sources = append(sources, syntax.Source{Path: path})
			}
			read := func(string) ([]byte, bool) { return []byte("let a;"), true }
			cache := syntax.NewCache(
				context.Background(), []syntax.Provider{provider},
				syntax.Corpus{Sources: sources, Read: read})
			for _, path := range testCase.paths {
				if file, ok := cache.Get(path); ok {
					t.Fatalf("%s got a summary %+v, want it unavailable", path, file)
				}
			}
			got := cache.Unavailable()
			if len(got) != len(testCase.paths) {
				t.Fatalf("unavailable = %+v, want one entry per language", got)
			}
			if reason := got[0].Reason; !strings.Contains(reason, tsproc.ErrUnavailable.Error()) {
				t.Fatalf("reason %q does not name the sticky failure", reason)
			}
		})
	}
}

func TestProviderSummarizeAliases(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"web/package.json":  `{"imports": {"#lib/*": "./src/lib/*"}}`,
		"web/tsconfig.json": `{"compilerOptions": {"paths": {"@app/*": ["./src/app/*"]}}}`,
		"web/src/lib/a.ts": "import 'pkg';\nimport '#lib/b';\nimport '../x';\n" +
			"import './c';\nimport '../../../out';\n",
		"web/src/routes/p.ts": "import '#lib/q';\nimport '../lib/b';\nimport '@app/q';\n",
		"free/f.ts":           "import '../g';\nimport './h';\n",
	}
	read := func(path string) ([]byte, bool) {
		text, ok := files[path]
		return []byte(text), ok
	}
	sources := []syntax.Source{
		{Path: "web/src/lib/a.ts"}, {Path: "web/src/routes/p.ts"}, {Path: "free/f.ts"},
	}
	got, err := New().Summarize(context.Background(), syntax.Corpus{Sources: sources, Read: read})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	type outcome struct {
		prefer  string
		group   int
		upward  bool
		foreign bool
	}
	testCases := []struct {
		path string
		want []outcome
	}{
		{path: "web/src/lib/a.ts", want: []outcome{
			{group: 1},
			{prefer: "./b", group: 2},
			{group: 2, upward: true},
			{group: 2},
			{group: 2, foreign: true},
		}},
		{path: "web/src/routes/p.ts", want: []outcome{
			{group: 2}, {prefer: "#lib/b", group: 2}, {group: 2},
		}},
		{path: "free/f.ts", want: []outcome{{group: 2, upward: true}, {group: 2}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()

			var have []outcome
			for _, imported := range got[testCase.path].Imports {
				have = append(
					have,
					outcome{imported.Prefer, imported.Group, imported.Upward, imported.Foreign},
				)
			}
			if !slices.Equal(have, testCase.want) {
				t.Fatalf("imports %+v, want %+v", have, testCase.want)
			}
		})
	}
}
