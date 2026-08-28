package loopexit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func tickerLoopCases() []checkFileTestCase {
	cases := []checkFileTestCase{
		{
			name: "ticker.C loop with a Done arm has no finding",
			source: "package foo\n" +
				"import (\n\t\"context\"\n\t\"time\"\n)\n" +
				"func run(ctx context.Context, ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-ctx.Done():\n" +
				"\t\t\treturn\n" +
				"\t\tcase <-ticker.C:\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "ticker.C loop with no Done arm is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
		},
		{
			name: "for range time.Tick is flagged with no Done arm",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(d time.Duration) {\n" +
				"\tfor range time.Tick(d) {\n" +
				"\t}\n" +
				"}\n",
			expect: 1,
		},
	}
	cases = append(cases, tickerRangeFormCases()...)
	cases = append(cases, tickerHoistedScopeCases()...)
	cases = append(cases, loopTerminationCases()...)
	return append(cases, structFieldTickerCases()...)
}

func structFieldTickerCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a *time.Ticker struct field with no exit is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type poller struct{ ticker *time.Ticker }\n" +
				"func (p *poller) run() {\n" +
				"\tfor {\n" +
				"\t\t<-p.ticker.C\n" +
				"\t\tp.work()\n" +
				"\t}\n" +
				"}\n" +
				"func (p *poller) work() {}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
		},
		{
			name: "a deeply nested struct field ticker with no exit is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type scheduler struct{ ticker *time.Ticker }\n" +
				"func run(current scheduler) {\n" +
				"\tfor {\n" +
				"\t\t<-current.ticker.C\n" +
				"\t\tcurrent.flush()\n" +
				"\t}\n" +
				"}\n" +
				"func (scheduler) flush() {}\n",
			expect:     1,
			wantSubstr: "ticker-paced loop has no",
		},
	}
}

func loopTerminationCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a select loop with a plain stop channel that returns is not flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func worker(stopCh <-chan struct{}, ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-stopCh:\n" +
				"\t\t\treturn\n" +
				"\t\tcase <-ticker.C:\n" +
				"\t\t\twork()\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n" +
				"func work() {}\n",
			expect: 0,
		},
		{
			name: "a bounded for-range-ticker.C loop with a count limit is not flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func bounded(ticker *time.Ticker) {\n" +
				"\tcount := 0\n" +
				"\tfor range ticker.C {\n" +
				"\t\tcount++\n" +
				"\t\tif count >= 10 {\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			expect: 0,
		},
	}
}

func tickerRangeFormCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "bare for range ticker.C is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor range ticker.C {\n" +
				"\t}\n" +
				"}\n",
			expect:     1,
			wantLine:   4,
			wantSubstr: "ticker-paced loop has no",
		},
		{
			name: "for t := range ticker.C is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor t := range ticker.C {\n" +
				"\t\t_ = t\n" +
				"\t}\n" +
				"}\n",
			expect:   1,
			wantLine: 4,
		},
		{
			name: "for range time.NewTicker(d).C is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(d time.Duration) {\n" +
				"\tfor range time.NewTicker(d).C {\n" +
				"\t}\n" +
				"}\n",
			expect:   1,
			wantLine: 4,
		},
	}
}

func tickerHoistedScopeCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a variable-assigned time.Tick receive is flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(d time.Duration) {\n" +
				"\ttick := time.Tick(d)\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-tick:\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			expect:   1,
			wantLine: 5,
		},
		{
			name: "a hoisted done := ctx.Done() shutdown arm suppresses the finding",
			source: "package foo\n" +
				"import (\n\t\"context\"\n\t\"time\"\n)\n" +
				"func run(ctx context.Context, ticker *time.Ticker) {\n" +
				"\tdone := ctx.Done()\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-done:\n" +
				"\t\t\treturn\n" +
				"\t\tcase <-ticker.C:\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "an outer loop around a ticker-paced inner loop is not itself flagged",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\tfor {\n" +
				"\t\t\t<-ticker.C\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			expect:   1,
			wantLine: 5,
		},
		{
			name: "a ctx.Done receive inside a nested goroutine does not suppress the loop",
			source: "package foo\n" +
				"import (\n\t\"context\"\n\t\"time\"\n)\n" +
				"func run(ctx context.Context, ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t\tgo func() {\n" +
				"\t\t\tmark()\n" +
				"\t\t\t<-ctx.Done()\n" +
				"\t\t\tcleanup()\n" +
				"\t\t}()\n" +
				"\t}\n" +
				"}\n" +
				"func mark()    {}\n" +
				"func cleanup() {}\n",
			expect:     1,
			wantLine:   7,
			wantSubstr: "ticker-paced loop has no",
		},
	}
}

func TestContainsMatch_StopsAtNestedLoopAndFuncLit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "a direct receive in this loop's body is found",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\t<-ticker.C\n" +
				"\t}\n" +
				"}\n",
			want: true,
		},
		{
			name: "a receive inside a nested loop is not this loop's",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\tfor {\n" +
				"\t\t\t<-ticker.C\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a receive inside a nested closure is not this loop's",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func run(ticker *time.Ticker) {\n" +
				"\tfor {\n" +
				"\t\tgo func() {\n" +
				"\t\t\t<-ticker.C\n" +
				"\t\t}()\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body := parseFirstForBody(t, testCase.source)
			scope := channelScope{typedIdents: map[string]bool{"ticker": true}}
			got := containsDotCReceive(body, scope)
			if got != testCase.want {
				t.Fatalf("containsDotCReceive() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func parseFirstForBody(t *testing.T, source string) *ast.BlockStmt {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var body *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if body != nil {
			return false
		}
		if forStmt, ok := node.(*ast.ForStmt); ok {
			body = forStmt.Body
			return false
		}
		return true
	})
	if body == nil {
		t.Fatal("no for statement found in source")
	}
	return body
}
