package variadicindex

import (
	"slices"
	"testing"
)

func TestScope(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(
		shadowCases(), blockCases(), initCases(), rangeCases(),
		typeSwitchCases(), clauseCases(), closureCases(),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeSource(t, testCase.src)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func shadowCases() []checkCase {
	return []checkCase{
		{
			name: "other slice indexed",
			src: "package foo\nfunc f(opts ...int) int {\n\tother := []int{1}\n" +
				"\treturn other[0]\n}\n",
		},
		{
			name: "top-level define of a param is a plain assignment",
			src: "package foo\nfunc g() ([]int, error) { return nil, nil }\n" +
				"func f(opts ...int) int {\n\topts, err := g()\n\t_ = err\n" +
				"\treturn opts[0]\n}\n",
			expect: 1,
		},
		{
			name: "redeclared in a nested block stops the scan",
			src: "package foo\nfunc f(opts ...int) int {\n\tif true {\n" +
				"\t\topts := []int{1}\n\t\treturn opts[0]\n\t}\n\treturn 0\n}\n",
		},
		{
			name: "index before nested redeclaration still flagged",
			src: "package foo\nfunc f(opts ...int) int {\n\tif true {\n" +
				"\t\tx := opts[0]\n\t\topts := []int{x}\n\t\treturn opts[0]\n\t}\n" +
				"\treturn 0\n}\n",
			expect: 1,
		},
	}
}

func initCases() []checkCase {
	return []checkCase{
		{
			name: "unnamed variadic ignored",
			src:  "package foo\nfunc f(...int) {}\n",
		},
		{
			name: "var declaration in a nested block shadows",
			src: "package foo\nfunc f(opts ...int) int {\n\tif true {\n" +
				"\t\tvar opts []int\n\t\treturn opts[0]\n\t}\n\treturn 0\n}\n",
		},
		{
			name: "var declaration initialiser reads the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n\tif true {\n" +
				"\t\tvar opts = opts[0]\n\t\treturn opts\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "switch init shadows",
			src: "package foo\nfunc f(opts ...int) int {\n" +
				"\tswitch opts := []int{1}; len(opts) {\n\tcase 1:\n\t\treturn opts[0]\n" +
				"\t}\n\treturn 0\n}\n",
		},
		{
			name: "switch init rhs reads the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n" +
				"\tswitch opts := opts[0]; opts {\n\tcase 1:\n\t\treturn opts\n\t}\n" +
				"\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "if init shadows",
			src: "package foo\nfunc f(opts ...int) int {\n" +
				"\tif opts := []int{1}; len(opts) > 0 {\n\t\treturn opts[0]\n\t}\n" +
				"\treturn 0\n}\n",
		},
		{
			name: "if init rhs reads the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n" +
				"\tif opts := opts[0]; opts > 0 {\n\t\treturn opts\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "for init shadows",
			src: "package foo\nfunc f(opts ...int) int {\n" +
				"\tfor opts := []int{1}; len(opts) > 0; {\n\t\treturn opts[0]\n\t}\n" +
				"\treturn 0\n}\n",
		},
		{
			name: "var group reads the variadic before redeclaring it",
			src: "package foo\nfunc f(opts ...int) int {\n\tif true {\n\t\tvar (\n" +
				"\t\t\ta = opts[0]\n\t\t\topts = []int{1}\n\t\t\tb = opts[0]\n\t\t)\n" +
				"\t\treturn a + b\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
	}
}

func rangeCases() []checkCase {
	return []checkCase{
		{
			name: "range value shadows the variadic",
			src: "package foo\nfunc f(opts ...[]int) int {\n" +
				"\tfor _, opts := range opts {\n\t\treturn opts[0]\n\t}\n\treturn 0\n}\n",
		},
		{
			name: "range key shadows the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n\tfor opts := range 3 {\n" +
				"\t\treturn opts\n\t}\n\treturn 0\n}\n",
		},
		{
			name: "range expression reads the variadic",
			src: "package foo\nfunc f(opts ...[]int) int {\n" +
				"\tfor _, opts := range opts[0] {\n\t\treturn opts\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
	}
}

func typeSwitchCases() []checkCase {
	return []checkCase{
		{
			name: "type switch binding shadows",
			src: "package foo\nfunc f(v any, opts ...int) int {\n" +
				"\tswitch opts := v.(type) {\n\tcase []int:\n\t\treturn opts[0]\n\t}\n" +
				"\treturn 0\n}\n",
		},
		{
			name: "type switch init shadows",
			src: "package foo\nfunc f(v any, opts ...int) int {\n" +
				"\tswitch opts := []int{1}; v.(type) {\n\tcase int:\n\t\treturn opts[0]\n" +
				"\t}\n\treturn 0\n}\n",
		},
		{
			name: "type switch assign reads the variadic",
			src: "package foo\nfunc f(opts ...any) int {\n\tswitch opts[0].(type) {\n" +
				"\tcase int:\n\t\treturn 1\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "type switch binding of another name",
			src: "package foo\nfunc f(opts ...any) int {\n" +
				"\tswitch v := opts[0].(type) {\n\tcase int:\n\t\treturn v\n\t}\n" +
				"\treturn 0\n}\n",
			expect: 1,
		},
	}
}

func blockCases() []checkCase {
	return []checkCase{
		{
			name: "bare block shadows",
			src: "package foo\nfunc f(opts ...int) int {\n\t{\n\t\topts := []int{1}\n" +
				"\t\treturn opts[0]\n\t}\n}\n",
		},
		{
			name:   "bare block indexes the variadic",
			src:    "package foo\nfunc f(opts ...int) int {\n\t{\n\t\treturn opts[0]\n\t}\n}\n",
			expect: 1,
		},
		{
			name: "labeled statement indexes the variadic",
			src: "package foo\nfunc f(opts ...int) int {\nL:\n\tfor range 3 {\n" +
				"\t\tif len(opts) > 0 {\n\t\t\tbreak L\n\t\t}\n\t}\n\treturn opts[0]\n}\n",
			expect: 1,
		},
		{
			name: "labeled loop shadows",
			src: "package foo\nfunc f(opts ...int) int {\nL:\n\tfor {\n" +
				"\t\topts := []int{1}\n\t\t_ = opts[0]\n\t\tbreak L\n\t}\n\treturn 0\n}\n",
		},
		{
			name: "else branch indexes the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n\tif len(opts) == 0 {\n" +
				"\t\treturn 0\n\t} else {\n\t\treturn opts[0]\n\t}\n}\n",
			expect: 1,
		},
		{
			name: "else branch shadows",
			src: "package foo\nfunc f(opts ...int) int {\n\tif len(opts) == 0 {\n" +
				"\t\treturn 0\n\t} else {\n\t\topts := []int{1}\n\t\treturn opts[0]\n\t}\n" +
				"}\n",
		},
		{
			name: "else if chain indexes the variadic",
			src: "package foo\nfunc f(n int, opts ...int) int {\n\tif n == 0 {\n" +
				"\t\treturn 0\n\t} else if n == 1 {\n\t\treturn opts[0]\n\t}\n\treturn 2\n" +
				"}\n",
			expect: 1,
		},
	}
}

func clauseCases() []checkCase {
	return []checkCase{
		{
			name: "case expression reads the variadic",
			src: "package foo\nfunc f(n int, opts ...int) int {\n\tswitch n {\n" +
				"\tcase opts[0]:\n\t\treturn 1\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "switch tag reads the variadic",
			src: "package foo\nfunc f(opts ...int) int {\n\tswitch opts[0] {\n" +
				"\tcase 1:\n\t\treturn 1\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name: "for post reads the variadic",
			src: "package foo\nfunc f(opts ...int) {\n\tfor i := 0; i < 3; i += opts[0] " +
				"{\n\t}\n}\n",
			expect: 1,
		},
		{
			name: "comm send reads the variadic",
			src: "package foo\nfunc f(ch chan int, opts ...int) {\n\tselect {\n" +
				"\tcase ch <- opts[0]:\n\t}\n}\n",
			expect: 1,
		},
		{
			name: "comm receive binding shadows its body",
			src: "package foo\nfunc f(ch chan []int, opts ...int) int {\n\tselect {\n" +
				"\tcase opts := <-ch:\n\t\treturn opts[0]\n\t}\n}\n",
		},
		{
			name: "comm body reads the variadic",
			src: "package foo\nfunc f(ch chan int, opts ...int) int {\n\tselect {\n" +
				"\tcase <-ch:\n\t\treturn opts[0]\n\t}\n}\n",
			expect: 1,
		},
	}
}

func closureCases() []checkCase {
	return []checkCase{
		{
			name:   "parenthesised variadic flagged",
			src:    "package foo\nfunc f(opts ...int) int { return (opts)[0] }\n",
			expect: 1,
		},
		{
			name: "closure result shadows the variadic",
			src: "package foo\nfunc f(opts ...int) func() []int {\n" +
				"\treturn func() (opts []int) { return opts[:1] }\n}\n",
		},
		{
			name: "closure result shadows the variadic inside a block",
			src: "package foo\nfunc f(opts ...int) int {\n\tg := func() (opts []int) {\n" +
				"\t\t_ = opts[0]\n\t\treturn\n\t}\n\t_ = g\n\treturn 0\n}\n",
		},
		{
			name: "closure parameter shadows the variadic",
			src: "package foo\nfunc f(opts ...int) func([]int) int {\n" +
				"\treturn func(opts []int) int { return opts[0] }\n}\n",
		},
		{
			name: "closure capturing the variadic flagged",
			src: "package foo\nfunc f(opts ...int) func() int {\n" +
				"\treturn func() int { return opts[0] }\n}\n",
			expect: 1,
		},
		{
			name: "declaration without a body",
			src:  "package foo\nfunc f(opts ...int)\n",
		},
	}
}
