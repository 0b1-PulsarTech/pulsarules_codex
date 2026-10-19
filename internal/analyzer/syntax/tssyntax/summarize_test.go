package tssyntax

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/facts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/budget"
)

func summarizeText(t *testing.T, source syntax.Source, text string) *syntax.File {
	t.Helper()

	read := func(string) ([]byte, bool) { return []byte(text), true }
	out, err := New().Summarize(context.Background(), syntax.Corpus{
		Sources: withContent([]syntax.Source{source}, read),
		Read:    read,
	})
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	return out[source.Path]
}

func render(file *syntax.File) string {
	var out strings.Builder
	fmt.Fprintf(&out, "file %s language=%s error=%q\n", file.Path, file.Language, file.ParseError)
	for _, fn := range file.Functions {
		fmt.Fprintf(
			&out,
			"fn %s kind=%s line=%d body=%d-%d params=%d named=%d cx=%d test=%t nested=%t\n",
			fn.Name,
			fn.Kind,
			fn.Line,
			fn.BodyStart,
			fn.BodyEnd,
			fn.Params,
			fn.NamedParams,
			fn.Complexity,
			fn.TestContainer,
			fn.Nested,
		)
	}
	for _, block := range file.Blocks {
		fmt.Fprintf(&out, "block line=%d depth=%d\n", block.Line, block.Depth)
	}
	for _, imp := range file.Imports {
		fmt.Fprintf(&out, "import %s line=%d group=%d\n", imp.Literal, imp.Line, imp.Group)
	}
	fmt.Fprintf(&out, "groups %q\n", file.ImportGroups)
	for _, name := range file.Names {
		fmt.Fprintf(&out, "name %s kind=%s line=%d exported=%t exempt=%t\n",
			name.Name, name.Kind, name.Line, name.Exported, name.Exempt)
	}
	for _, site := range file.DocSites {
		fmt.Fprintf(&out, "doc %s kind=%s line=%d exempt=%t mixed=%t\n",
			site.Symbol, site.Kind, site.Line, site.Exempt, site.MixedDirective)
	}
	return out.String()
}

func TestSummarizeGolden(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"service.ts", "panel.tsx", "legacy.js", "panel.svelte", "plain.svelte",
	}
	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := "testdata/" + name
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read sample: %v", err)
			}
			got := render(summarizeText(t, syntax.Source{Path: path}, string(content)))
			want, err := os.ReadFile(path + ".golden")
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if got != string(want) {
				t.Fatalf("summary differs\n--- got\n%s--- want\n%s", got, want)
			}
		})
	}
}

type hostileCase struct {
	name      string
	path      string
	text      string
	wantError string
	wantFns   int
}

func hostileCases() []hostileCase {
	return slices.Concat(sizeCases(), shapeCases())
}

func sizeCases() []hostileCase {
	return []hostileCase{
		{
			name:      "file over the byte limit",
			path:      "big.ts",
			text:      strings.Repeat(" ", budget.MaxFileBytes+1),
			wantError: "skipped: 1048577 bytes is over the 1048576 byte limit",
		},
		{
			name:    "file at the byte limit is parsed",
			path:    "edge.ts",
			text:    "function f() {}" + strings.Repeat(" ", budget.MaxFileBytes-15),
			wantFns: 1,
		},
		{
			name: "parens over the nesting limit",
			path: "deep.ts",
			text: strings.Repeat(
				"(",
				budget.MaxNesting+1,
			) + "1" + strings.Repeat(
				")",
				budget.MaxNesting+1,
			),
			wantError: "skipped: nesting depth 501 is over the 500 parse limit",
		},
		{
			name: "parens at the nesting limit are parsed",
			path: "deep.js",
			text: "x = " + strings.Repeat(
				"(",
				budget.MaxNesting-1,
			) + "1" + strings.Repeat(
				")",
				budget.MaxNesting-1,
			),
			wantFns: 0,
		},
	}
}

func shapeCases() []hostileCase {
	return []hostileCase{
		{
			name:      "braces over the nesting limit in a svelte script",
			path:      "deep.svelte",
			text:      "<script>x = " + strings.Repeat("[", budget.MaxNesting+1) + "</script>",
			wantError: "skipped: nesting depth 501 is over the 500 parse limit",
		},
		{
			name:      "async defaults nested past the speculation limit",
			path:      "async.ts",
			text:      "x = " + strings.Repeat("async(a=", 11) + "0" + strings.Repeat(")", 11),
			wantError: "skipped: 11 nested arrow-like parentheses is over the 10 parse limit",
		},
		{
			name:      "openers hidden behind a CR line comment",
			path:      "cr.ts",
			text:      "// x\r" + strings.Repeat("(", budget.MaxOpeners+1),
			wantError: "skipped: 32769 brackets is over the 32768 parse limit",
		},
		{
			name:    "less-than loops are parsed",
			path:    "loops.ts",
			text:    strings.Repeat("for (;i < n;) {}\n", 10_000) + "function f() {}",
			wantFns: 1,
		},
		{
			name:      "tree deeper than the walk limit",
			path:      "chain.ts",
			text:      "x = 1" + strings.Repeat("+1", facts.MaxTreeDepth+10),
			wantError: "skipped: syntax tree is nested too deeply",
		},
		{
			name: "svelte markup is not bounded by the script guard",
			path: "wide.svelte",
			text: strings.Repeat(
				"<a></a>",
				budget.MaxOpeners,
			) + "<script>function f() {}</script>",
			wantError: "",
			wantFns:   1,
		},
	}
}

func TestSummarizeHostile(t *testing.T) {
	t.Parallel()

	testCases := hostileCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: testCase.path}, testCase.text)
			if got.ParseError != testCase.wantError {
				t.Fatalf("parse error %q, want %q", got.ParseError, testCase.wantError)
			}
			if len(got.Functions) != testCase.wantFns {
				t.Fatalf("%d functions, want %d", len(got.Functions), testCase.wantFns)
			}
		})
	}
}

func TestGuardedRecoversPanics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		run       func()
		wantError string
	}{
		{name: "no panic", run: func() {}},
		{
			name:      "string panic",
			run:       func() { panic("boom") },
			wantError: "skipped: parser failed: boom",
		},
		{
			name: "runtime panic",
			run:  func() { _ = []int(nil)[3] },
			wantError: "skipped: parser failed: runtime error: index out of range [3] with " +
				"length 0",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			summary := &syntax.File{Path: "x.ts"}
			got := guarded(summary, testCase.run)
			if got != summary || got.ParseError != testCase.wantError {
				t.Fatalf("got %+v, want error %q", got, testCase.wantError)
			}
		})
	}
}

type lineCase struct {
	name     string
	path     string
	text     string
	wantLine int
}

func lineCases() []lineCase {
	return []lineCase{
		{name: "lf", path: "a.ts", text: "\n\nfunction f() {}\n", wantLine: 3},
		{name: "crlf", path: "a.ts", text: "\r\n\r\nfunction f() {}\r\n", wantLine: 3},
		{name: "lone cr is not a break", path: "a.ts", text: "\r\rfunction f() {}\n", wantLine: 1},
		{
			name:     "line separator is not a break",
			path:     "a.ts",
			text:     "// \u2028\u2029\nfunction f() {}\n",
			wantLine: 2,
		},
		{
			name:     "multibyte before the function",
			path:     "a.js",
			text:     "const s = '\u03a9\u00e9\U0001F600';\nfunction f() {}\n",
			wantLine: 2,
		},
		{name: "byte order mark", path: "a.ts", text: "\ufeff\nfunction f() {}\n", wantLine: 2},
		{
			name:     "svelte line equals file line",
			path:     "a.svelte",
			text:     "<p>\u00e9</p>\n<!-- c -->\n<script>\n\nfunction f() {}\n</script>\n",
			wantLine: 5,
		},
		{
			name: "svelte second script",
			path: "a.svelte",
			text: "<script context=\"module\">\nlet a;\n</script>\n\n<script>\n\nfunction " +
				"f() {}\n</script>\n",
			wantLine: 7,
		},
	}
}

func TestSummarizeLineNumbers(t *testing.T) {
	t.Parallel()

	testCases := lineCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: testCase.path}, testCase.text)
			if len(got.Functions) != 1 || got.Functions[0].Line != testCase.wantLine {
				t.Fatalf("functions %+v, want one at line %d", got.Functions, testCase.wantLine)
			}
			if got.Functions[0].BodyStart != testCase.wantLine {
				t.Fatalf(
					"body starts at %d, want %d",
					got.Functions[0].BodyStart,
					testCase.wantLine,
				)
			}
		})
	}
}

func TestSummarizeTestFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		isTest        bool
		wantContainer bool
		wantBlocks    int
	}{
		{name: "test file", isTest: true, wantContainer: true, wantBlocks: 1},
		{name: "source file", isTest: false, wantContainer: false, wantBlocks: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			text := "describe('x', () => {\n  if (a) {\n  }\n});\n"
			source := syntax.Source{Path: "x.test.ts", IsTest: testCase.isTest}
			got := summarizeText(t, source, text)
			funcs := got.Functions
			if len(funcs) != 1 || funcs[0].TestContainer != testCase.wantContainer {
				t.Fatalf("functions %+v", funcs)
			}
			if len(got.Blocks) != testCase.wantBlocks {
				t.Fatalf("blocks %+v, want %d", got.Blocks, testCase.wantBlocks)
			}
		})
	}
}

func TestSummarizeScriptKinds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path      string
		text      string
		wantError bool
	}{
		{path: "a.ts", text: "const a: number = 1;"},
		{path: "a.mts", text: "const a: number = 1;"},
		{path: "a.cts", text: "const a: number = 1;"},
		{path: "a.tsx", text: "const a = <div>{1}</div>;"},
		{path: "a.ts", text: "const a = <div>{1}</div>;", wantError: true},
		{path: "a.js", text: "const a = 1;"},
		{path: "a.mjs", text: "export const a = 1;"},
		{path: "a.cjs", text: "module.exports = 1;"},
		{path: "a.jsx", text: "const a = <div>{1}</div>;"},
		{path: "a.ts", text: "const a = <number>b;"},
		{path: "a.js", text: "const a = <number>b;", wantError: true},
		{path: "UPPER.TS", text: "const a: number = 1;"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path+" "+testCase.text, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: testCase.path}, testCase.text)
			if (got.ParseError != "") != testCase.wantError {
				t.Fatalf("parse error %q, want error %v", got.ParseError, testCase.wantError)
			}
		})
	}
}

func TestScriptKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path string
		want core.ScriptKind
	}{
		{path: "a.ts", want: core.ScriptKindTS},
		{path: "a.MTS", want: core.ScriptKindTS},
		{path: "a.cts", want: core.ScriptKindTS},
		{path: "a.tsx", want: core.ScriptKindTSX},
		{path: "a.js", want: core.ScriptKindJS},
		{path: "a.mjs", want: core.ScriptKindJS},
		{path: "a.cjs", want: core.ScriptKindJS},
		{path: "a.jsx", want: core.ScriptKindJSX},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()

			ext := strings.ToLower(filepath.Ext(testCase.path))
			id := language.ForExtension(ext).ID
			if got := scriptKind(id, ext); got != testCase.want {
				t.Fatalf("scriptKind(%s, %s) = %v, want %v", id, ext, got, testCase.want)
			}
		})
	}
}
