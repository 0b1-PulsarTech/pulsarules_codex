package goast

import (
	"go/ast"
	"testing"
)

type pkgCallTestCase struct {
	name   string
	source string
	names  []string
	want   bool
}

func TestPkgCall(t *testing.T) {
	t.Parallel()

	testCases := append(pkgCallImportCases(), pkgCallShadowCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := parse(t, testCase.source)
			var got bool
			ast.Inspect(file, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					got = got || PkgCall(file, call, "time", testCase.names...)
				}
				return true
			})
			if got != testCase.want {
				t.Fatalf("PkgCall = %v, want %v", got, testCase.want)
			}
		})
	}
}

func pkgCallImportCases() []pkgCallTestCase {
	return []pkgCallTestCase{
		{
			name:   "default import",
			source: "import \"time\"\nfunc f() { time.Sleep(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "aliased import",
			source: "import clock \"time\"\nfunc f() { clock.Sleep(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "default name after alias is not the package",
			source: "import clock \"time\"\nfunc f() { time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
		{
			name:   "blank import binds nothing",
			source: "import _ \"time\"\nfunc f() { time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
		{
			name:   "dot import calls the bare name",
			source: "import . \"time\"\nfunc f() { Sleep(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "dot import without names matches nothing bare",
			source: "import . \"time\"\nfunc f() { Sleep(1) }",
		},
		{
			name:   "parenthesised callee",
			source: "import \"time\"\nfunc f() { (time.Sleep)(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "no names matches any member",
			source: "import \"time\"\nfunc f() { time.Now() }",
			want:   true,
		},
		{
			name:   "other member",
			source: "import \"time\"\nfunc f() { time.Now() }",
			names:  []string{"Sleep"},
		},
	}
}

func pkgCallShadowCases() []pkgCallTestCase {
	return []pkgCallTestCase{
		{
			name:   "not imported",
			source: "func f() { time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
		{
			name:   "local variable shadows the import",
			source: "import \"time\"\nfunc f() { time := sleeper{}; time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
		{
			name:   "parameter shadows the import",
			source: "import \"time\"\nfunc f(time sleeper) { time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
		{
			name:   "closure captured local shadows the import",
			source: "import \"time\"\nfunc f() { time := sleeper{}; func() { time.Sleep(1) }() }",
			names:  []string{"Sleep"},
		},
		{
			name:   "inner block shadow then a real call",
			source: "import \"time\"\nfunc f() { { time := sleeper{}; _ = time }; time.Sleep(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "shadow in a func literal then an outer real call",
			source: "import \"time\"\nfunc f() { func() { time := sleeper{}; _ = time }(); time.Sleep(1) }",
			names:  []string{"Sleep"},
			want:   true,
		},
		{
			name:   "range variable shadows the import",
			source: "import \"time\"\nfunc f() { for _, time := range list { time.Sleep(1) } }",
			names:  []string{"Sleep"},
		},
		{
			name:   "named result shadows the import",
			source: "import \"time\"\nfunc f() (time sleeper) { time.Sleep(1); return }",
			names:  []string{"Sleep"},
		},
		{
			name:   "type switch binding shadows the import",
			source: "import \"time\"\nfunc f(v any) { switch time := v.(type) { case sleeper: time.Sleep(1) } }",
			names:  []string{"Sleep"},
		},
		{
			name:   "local variable and no import",
			source: "func f() { var time sleeper; time.Sleep(1) }",
			names:  []string{"Sleep"},
		},
	}
}
