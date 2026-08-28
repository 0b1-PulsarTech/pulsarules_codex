package loopexit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

type channelScopeTestCase struct {
	name           string
	source         string
	wantTickerName string
	wantDoneName   string
	wantTypedName  string
	wantFieldName  string
}

func channelScopeCases() []channelScopeTestCase {
	return append(channelScopeAssignmentCases(), channelScopeTypeCases()...)
}

func channelScopeAssignmentCases() []channelScopeTestCase {
	return []channelScopeTestCase{
		{
			name: "a time.Tick assignment is tracked as a ticker identifier",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func poll(interval time.Duration) {\n" +
				"\ttick := time.Tick(interval)\n" +
				"\t_ = tick\n" +
				"}\n",
			wantTickerName: "tick",
		},
		{
			name: "a time.NewTicker assignment is tracked as a ticker identifier",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func poll(interval time.Duration) {\n" +
				"\tticker := time.NewTicker(interval)\n" +
				"\t_ = ticker\n" +
				"}\n",
			wantTickerName: "ticker",
		},
		{
			name: "a ctx.Done assignment is tracked as a done identifier",
			source: "package foo\n" +
				"import \"context\"\n" +
				"func poll(ctx context.Context) {\n" +
				"\tdone := ctx.Done()\n" +
				"\t_ = done\n" +
				"}\n",
			wantDoneName: "done",
		},
		{
			name: "an assignment inside a nested func literal is not this function's scope",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func poll(interval time.Duration) {\n" +
				"\tgo func() {\n" +
				"\t\ttick := time.Tick(interval)\n" +
				"\t\t_ = tick\n" +
				"\t}()\n" +
				"}\n",
			wantTickerName: "",
		},
	}
}

func channelScopeTypeCases() []channelScopeTestCase {
	return []channelScopeTestCase{
		{
			name: "a *time.Ticker parameter is tracked as a typed identifier",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func poll(ticker *time.Ticker) {\n" +
				"\t_ = ticker\n" +
				"}\n",
			wantTypedName: "ticker",
		},
		{
			name: "a time.Ticker var declaration is tracked as a typed identifier",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func poll() {\n" +
				"\tvar pollTicker time.Ticker\n" +
				"\t_ = pollTicker\n" +
				"}\n",
			wantTypedName: "pollTicker",
		},
		{
			name: "a plain struct parameter is not tracked as a typed identifier",
			source: "package foo\n" +
				"type Watcher struct{ C chan int }\n" +
				"func poll(watcher *Watcher) {\n" +
				"\t_ = watcher\n" +
				"}\n",
			wantTypedName: "",
		},
		{
			name: "a *time.Ticker struct field is tracked as a field name",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type poller struct{ ticker *time.Ticker }\n" +
				"func poll(target *poller) {\n" +
				"\t_ = target\n" +
				"}\n",
			wantFieldName: "ticker",
		},
		{
			name: "a Watcher's plain C channel field is not a ticker field name",
			source: "package foo\n" +
				"type Watcher struct{ C chan int }\n" +
				"func poll(watcher *Watcher) {\n" +
				"\t_ = watcher\n" +
				"}\n",
			wantFieldName: "",
		},
	}
}

func TestCollectChannelScope(t *testing.T) {
	t.Parallel()

	for _, testCase := range channelScopeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, params, body := parseFirstFuncBody(t, testCase.source)
			scope := collectChannelScope(params, body, collectTickerFieldNames(file))
			assertChannelScope(t, scope, testCase)
		})
	}
}

func assertChannelScope(t *testing.T, scope channelScope, testCase channelScopeTestCase) {
	t.Helper()

	assertIdentTracked(t, "tickerIdents", scope.tickerIdents, testCase.wantTickerName)
	assertIdentTracked(t, "doneIdents", scope.doneIdents, testCase.wantDoneName)
	assertIdentTracked(t, "typedIdents", scope.typedIdents, testCase.wantTypedName)
	assertIdentTracked(t, "tickerFieldNames", scope.tickerFieldNames, testCase.wantFieldName)
}

func assertIdentTracked(t *testing.T, mapName string, ident map[string]bool, want string) {
	t.Helper()

	if want != "" && !ident[want] {
		t.Fatalf("%s = %v, want %q tracked", mapName, ident, want)
	}
	if want == "" && len(ident) != 0 {
		t.Fatalf("%s = %v, want none tracked", mapName, ident)
	}
}

func TestIsDotCSelector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		expr string
		want bool
	}{
		{name: "a .C selector matches", expr: "ticker.C", want: true},
		{name: "a call ending in .C matches", expr: "time.NewTicker(interval).C", want: true},
		{name: "a bare identifier does not match", expr: "tick", want: false},
		{name: "a differently named selector does not match", expr: "ticker.Stop", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isDotCSelector(parseExpr(t, testCase.expr))
			if got != testCase.want {
				t.Fatalf("isDotCSelector(%s) = %v, want %v", testCase.expr, got, testCase.want)
			}
		})
	}
}

func TestIsTickerDotCSelector(t *testing.T) {
	t.Parallel()

	scope := channelScope{
		tickerIdents:     map[string]bool{},
		typedIdents:      map[string]bool{"ticker": true},
		tickerFieldNames: map[string]bool{"ticker": true},
	}

	testCases := []struct {
		name string
		expr string
		want bool
	}{
		{name: "a typed identifier's .C matches", expr: "ticker.C", want: true},
		{
			name: "a direct time.NewTicker(...).C call matches",
			expr: "time.NewTicker(interval).C",
			want: true,
		},
		{name: "a one-level struct field selector matches", expr: "poller.ticker.C", want: true},
		{
			name: "a deeply nested struct field selector still matches",
			expr: "server.pool.worker.ticker.C",
			want: true,
		},
		{name: "a plain struct's own C field does not match", expr: "watcher.C", want: false},
		{name: "an untracked identifier's .C does not match", expr: "other.C", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := isTickerDotCSelector(parseExpr(t, testCase.expr), scope)
			if got != testCase.want {
				t.Fatalf(
					"isTickerDotCSelector(%s) = %v, want %v",
					testCase.expr,
					got,
					testCase.want,
				)
			}
		})
	}
}

func parseFirstFuncBody(t *testing.T, source string) (*ast.File, *ast.FieldList, *ast.BlockStmt) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var params *ast.FieldList
	var body *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if body != nil {
			return false
		}
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			params = funcDecl.Type.Params
			body = funcDecl.Body
			return false
		}
		return true
	})
	if body == nil {
		t.Fatal("no function declaration found in source")
	}
	return file, params, body
}

func parseExpr(t *testing.T, expr string) ast.Expr {
	t.Helper()

	parsed, err := parser.ParseExpr(expr)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
