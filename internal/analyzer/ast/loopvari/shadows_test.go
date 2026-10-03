package loopvari

import (
	"go/ast"
	"testing"
)

func TestDeclShadows(t *testing.T) {
	t.Parallel()

	testCases := append(topShadowCases(), localShadowCases()...)
	testCases = append(testCases, siblingShadowCases()...)
	testCases = append(testCases, enclosingShadowCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f := parseDecls(t, "package foo\n"+testCase.source)
			decl := f.Decls[testCase.decl]
			at := ast.Node(decl)
			ast.Inspect(decl, func(n ast.Node) bool {
				if _, ok := n.(*ast.RangeStmt); ok {
					at = n
				}
				return true
			})
			got := declShadows(packageShadows(f), decl, at)
			if got["len"] != testCase.wantLen || got["cap"] != testCase.wantCap {
				t.Fatalf(
					"shadowed = %v, want len %v cap %v",
					got,
					testCase.wantLen,
					testCase.wantCap,
				)
			}
		})
	}
}

type shadowCase struct {
	name    string
	source  string
	decl    int
	wantLen bool
	wantCap bool
}

const shadowLoop = "for i := range s {\n_ = i\n}"

func topShadowCases() []shadowCase {
	return []shadowCase{
		{name: "nothing declared", source: "func f() {}"},
		{
			name:    "package function len",
			source:  "func len() {}\nfunc f() {}",
			decl:    1,
			wantLen: true,
		},
		{name: "package var cap", source: "var cap = 1\nfunc f() {}", decl: 1, wantCap: true},
		{name: "package type len", source: "type len int\nfunc f() {}", decl: 1, wantLen: true},
		{name: "method named len", source: "type B struct{}\nfunc (B) len() {}", decl: 1},
		{name: "struct field cap", source: "type B struct{ cap int }", decl: 0},
		{name: "interface method len", source: "type B interface{ len() int }", decl: 0},
		{
			name:    "parameter around the loop",
			source:  "func f(s []int, len int) {\n" + shadowLoop + "\n}",
			wantLen: true,
		},
		{
			name:    "receiver around the loop",
			source:  "type B struct{}\nfunc (len B) f(s []int) {\n" + shadowLoop + "\n}",
			decl:    1,
			wantLen: true,
		},
		{name: "parameter in another decl", source: "func f(len int) {}\nfunc g() {}", decl: 1},
	}
}

func localShadowCases() []shadowCase {
	return []shadowCase{
		{
			name:    "local define before the loop",
			source:  "func f(s []int) { cap := 1; _ = cap\n" + shadowLoop + "\n}",
			wantCap: true,
		},
		{
			name:   "local define after the loop",
			source: "func f(s []int) {\n" + shadowLoop + "\ncap := 1; _ = cap }",
		},
		{
			name:    "local var before the loop",
			source:  "func f(s []int) { var len int; _ = len\n" + shadowLoop + "\n}",
			wantLen: true,
		},
		{
			name:    "range define around the loop",
			source:  "func f(s []int) { for len := range s {\n" + shadowLoop + "\n} }",
			wantLen: true,
		},
		{
			name:    "if init around the loop",
			source:  "func f(s []int) { if len := 1; len > 0 {\n" + shadowLoop + "\n} }",
			wantLen: true,
		},
		{
			name: "case clause local around the loop",
			source: "func f(s []int) { switch { default:\ncap := 1; _ = cap\n" + shadowLoop +
				"\n} }",
			wantCap: true,
		},
		{
			name:   "block local in a closed sibling block",
			source: "func f(s []int) { { len := 1; _ = len }\n" + shadowLoop + "\n}",
		},
		{
			name:   "sibling closure parameter",
			source: "func f(s []int) { g := func(len int) {}; _ = g\n" + shadowLoop + "\n}",
		},
		{
			name: "sibling closure local",
			source: "func f(s []int) { g := func() { cap := 1; _ = cap }; _ = g\n" + shadowLoop +
				"\n}",
		},
		{
			name:    "closure parameter around the loop",
			source:  "func f(s []int) { g := func(len int) {\n" + shadowLoop + "\n}; _ = g }",
			wantLen: true,
		},
		{
			name:   "function type variable parameter",
			source: "func f(s []int) { var g func(len int); _ = g\n" + shadowLoop + "\n}",
		},
		{
			name:   "function type in a type declaration",
			source: "func f(s []int) { type h func(cap int)\n" + shadowLoop + "\n}",
		},
		{
			name:   "plain assignment",
			source: "func f(s []int) { var n int; n = 1; _ = n\n" + shadowLoop + "\n}",
		},
	}
}

func siblingShadowCases() []shadowCase {
	return []shadowCase{
		{
			name:   "sibling if init",
			source: "func f(s []int) { if len := 1; len > 0 {\n}\n" + shadowLoop + "\n}",
		},
		{
			name: "sibling switch init",
			source: "func f(s []int) { switch len := 1; {\ncase len > 0:\n}\n" + shadowLoop +
				"\n}",
		},
		{
			name:   "sibling for init",
			source: "func f(s []int) { for len := 0; ; {\nbreak\n}\n" + shadowLoop + "\n}",
		},
		{
			name: "sibling select case define",
			source: "func f(s []int, ch chan int) { select {\ncase len := <-ch:\n_ = len\n}\n" +
				shadowLoop + "\n}",
		},
		{
			name: "loop in another select clause",
			source: "func f(s []int, ch chan int) { select {\ncase len := <-ch:\n_ = len\n" +
				"default:\n" + shadowLoop + "\n} }",
		},
		{
			name: "loop in another case clause",
			source: "func f(s []int) { switch {\ncase true:\nlen := 1; _ = len\ndefault:\n" +
				shadowLoop + "\n} }",
		},
		{
			name:   "sibling range define",
			source: "func f(s []int) { for _, len := range s {\n_ = len\n}\n" + shadowLoop + "\n}",
		},
	}
}

func enclosingShadowCases() []shadowCase {
	return []shadowCase{
		{
			name:    "range value around the loop",
			source:  "func f(s []int) { for _, len := range s {\n" + shadowLoop + "\n} }",
			wantLen: true,
		},
		{
			name:    "local type before the loop",
			source:  "func f(s []int) { type len int\n" + shadowLoop + "\n}",
			wantLen: true,
		},
		{
			name:    "named result around the loop",
			source:  "func f(s []int) (len int) {\n" + shadowLoop + "\nreturn }",
			wantLen: true,
		},
	}
}
