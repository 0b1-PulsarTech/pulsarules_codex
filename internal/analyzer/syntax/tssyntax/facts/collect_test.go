package facts

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/tsparse"
)

func collectOf(t *testing.T, text string, options Options) syntax.File {
	t.Helper()

	facts, err := Collect(tsparse.Parse(text, core.ScriptKindTS), text, options)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return facts
}

func parseOf(t *testing.T, text string) (*ast.Node, source) {
	t.Helper()

	return tsparse.Parse(text, core.ScriptKindTS).AsNode(), newSource(text)
}

func walkOf(
	t *testing.T,
	text string,
	options Options,
	pick func(*functionFacts) []nodeVisitor,
) *functionFacts {
	t.Helper()

	root, src := parseOf(t, text)
	functions := &functionFacts{source: src, isTest: options.IsTest}
	if err := newWalker(functions, pick(functions)...).run(root); err != nil {
		t.Fatalf("walk: %v", err)
	}
	return functions
}

func functionsOf(t *testing.T, text string, options Options) *functionFacts {
	t.Helper()

	return walkOf(t, text, options, func(*functionFacts) []nodeVisitor { return nil })
}

func complexityOf(t *testing.T, text string, options Options) *functionFacts {
	t.Helper()

	return walkOf(t, text, options, func(functions *functionFacts) []nodeVisitor {
		return []nodeVisitor{complexityFacts{functions}}
	})
}

func namesOf(t *testing.T, text string) []syntax.Name {
	t.Helper()

	root, src := parseOf(t, text)
	names := &nameFacts{source: src, locals: exportedLocals(root.Statements())}
	if err := newWalker(&functionFacts{source: src}, names).run(root); err != nil {
		t.Fatalf("walk: %v", err)
	}
	return names.out
}

func importsOf(t *testing.T, text string, local []string) []syntax.Import {
	t.Helper()

	root, src := parseOf(t, text)
	imports := &importFacts{source: src, local: local}
	imports.collect(root.Statements())
	return imports.out
}

func docSitesOf(t *testing.T, text string) []syntax.DocSite {
	t.Helper()

	root, src := parseOf(t, text)
	docs := &docFacts{source: src, factory: ast.NewNodeFactory(ast.NodeFactoryHooks{})}
	docs.docStatements(root.Statements(), "")
	return docs.out
}

type recorder struct{ kinds []ast.Kind }

func (rec *recorder) visit(node *ast.Node) { rec.kinds = append(rec.kinds, node.Kind) }

func TestWalkerFansOutToEveryVisitor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want []ast.Kind
	}{
		{name: "empty file", text: "", want: []ast.Kind{ast.KindEndOfFile}},
		{
			name: "function with a parameter",
			text: "function f(a) {}",
			want: []ast.Kind{
				ast.KindFunctionDeclaration, ast.KindIdentifier, ast.KindParameter,
				ast.KindIdentifier, ast.KindBlock, ast.KindEndOfFile,
			},
		},
		{
			name: "bodiless overload prunes its subtree",
			text: "function f(a: number): void;\nfunction f(a: any) {}",
			want: []ast.Kind{
				ast.KindFunctionDeclaration, ast.KindIdentifier, ast.KindParameter,
				ast.KindIdentifier, ast.KindAnyKeyword, ast.KindBlock, ast.KindEndOfFile,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			first, second := &recorder{}, &recorder{}
			walkOf(t, testCase.text, Options{}, func(*functionFacts) []nodeVisitor {
				return []nodeVisitor{first, second}
			})
			if !slices.Equal(first.kinds, second.kinds) {
				t.Fatalf("visitors saw different nodes: %v vs %v", first.kinds, second.kinds)
			}
			if !slices.Equal(first.kinds, testCase.want) {
				t.Fatalf("kinds %v, want %v", first.kinds, testCase.want)
			}
		})
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		wantErr error
		wantFns int
	}{
		{name: "empty file", text: ""},
		{name: "one function", text: "function f() {}\n", wantFns: 1},
		{
			name:    "chain just under the tree limit",
			text:    "const a = 1" + strings.Repeat("+1", MaxTreeDepth/2) + ";",
			wantFns: 0,
		},
		{
			name:    "chain over the tree limit",
			text:    "const a = 1" + strings.Repeat("+1", MaxTreeDepth+10) + ";",
			wantErr: errTooDeep,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := tsparse.Parse(testCase.text, core.ScriptKindTS)
			facts, err := Collect(root, testCase.text, Options{})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
			if len(facts.Functions) != testCase.wantFns {
				t.Fatalf("%d functions, want %d", len(facts.Functions), testCase.wantFns)
			}
		})
	}
}

func TestCollectCommonJSExportsAreCountedOnce(t *testing.T) {
	t.Parallel()

	text := "module.exports = { f: function (a) {} };\nexports.g = (b) => { return b; };\n"
	facts, err := Collect(tsparse.Parse(text, core.ScriptKindJS), text, Options{})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	names := make([]string, 0, len(facts.Functions))
	for _, fn := range facts.Functions {
		names = append(names, fn.Name)
	}
	if want := []string{"f", "g"}; !slices.Equal(names, want) {
		t.Fatalf("functions %v, want %v", names, want)
	}
	if len(facts.Names) != 2 {
		t.Fatalf("names %+v, want the two parameters once", facts.Names)
	}
}

func TestCollectWiresEveryFact(t *testing.T) {
	t.Parallel()

	const nested = "function f(param) {\n  if (param) {\n    return 1;\n  }\n  return 2;\n}\n"
	testCases := []struct {
		name string
		text string
		got  func(syntax.File) int
		want int
	}{
		{"functions", nested, func(facts syntax.File) int { return len(facts.Functions) }, 1},
		{
			"complexity",
			nested,
			func(facts syntax.File) int { return facts.Functions[0].Complexity },
			2,
		},
		{"blocks", nested, func(facts syntax.File) int { return len(facts.Blocks) }, 1},
		{"names", nested, func(facts syntax.File) int { return len(facts.Names) }, 2},
		{
			"imports",
			"import x from 'pkg';\n",
			func(facts syntax.File) int { return len(facts.Imports) },
			1,
		},
		{
			"import groups", "import x from 'pkg';\n",
			func(facts syntax.File) int { return len(facts.ImportGroups) }, 3,
		},
		{
			"doc sites", "/** a */\nfunction g() {}\n",
			func(facts syntax.File) int { return len(facts.DocSites) }, 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.got(collectOf(t, testCase.text, Options{})); got != testCase.want {
				t.Fatalf("%s = %d, want %d", testCase.name, got, testCase.want)
			}
		})
	}
}

func TestWalkerStopsAtTheDepthLimit(t *testing.T) {
	t.Parallel()

	text := "const a = 1" + strings.Repeat("+1", MaxTreeDepth+10) + ";\nconst b = 2;\n"
	root, src := parseOf(t, text)
	rec := &recorder{}
	err := newWalker(&functionFacts{source: src}, rec).run(root)
	if !errors.Is(err, errTooDeep) {
		t.Fatalf("error %v, want %v", err, errTooDeep)
	}
	statements := 0
	for _, kind := range rec.kinds {
		if kind == ast.KindVariableStatement {
			statements++
		}
	}
	if statements != 1 {
		t.Fatalf("visited %d statements, want the walk to stop after the first", statements)
	}
}
