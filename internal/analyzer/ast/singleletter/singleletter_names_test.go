package singleletter

import "testing"

func TestCheckFile_Names(t *testing.T) {
	t.Parallel()

	testCases := declarationCases()
	testCases = append(testCases, valueCases()...)
	testCases = append(testCases, statementCases()...)
	testCases = append(testCases, bindingCases()...)
	testCases = append(testCases, receiverCases()...)
	testCases = append(testCases, importAndPackageCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertMessages(t, testCase.source, defaultSettings(), testCase.want)
		})
	}
}

func declarationCases() []testCase {
	return []testCase{
		{
			"function name",
			"package pkg\n\nfunc f() {}\n",
			[]string{`function "f" is a single letter`},
		},
		{
			"method name and receiver",
			"package pkg\n\ntype Srv struct{}\n\nfunc (s Srv) M() {}\n",
			[]string{`method "M" is a single letter`, `receiver "s" is a single letter`},
		},
		{
			"parameter",
			"package pkg\n\nfunc run(f int) {}\n",
			[]string{`parameter "f" is a single letter`},
		},
		{
			"result",
			"package pkg\n\nfunc run() (n int) { return }\n",
			[]string{`result "n" is a single letter`},
		},
		{
			"function literal parameter",
			"package pkg\n\nvar run = func(f int) {}\n",
			[]string{`parameter "f" is a single letter`},
		},
		{
			"func type field parameter",
			"package pkg\n\ntype Run struct{ Do func(f int) }\n",
			[]string{`parameter "f" is a single letter`},
		},
		{
			"interface method and its parameter",
			"package pkg\n\ntype Doer interface{ D(f int) }\n",
			[]string{`method "D" is a single letter`, `parameter "f" is a single letter`},
		},
		{"type name", "package pkg\n\ntype T struct{}\n", []string{`type "T" is a single letter`}},
		{
			"type parameter",
			"package pkg\n\ntype Box[V any] struct{}\n",
			[]string{`type parameter "V" is a single letter`},
		},
		{
			"function type parameter",
			"package pkg\n\nfunc Keep[V any]() {}\n",
			[]string{`type parameter "V" is a single letter`},
		},
	}
}

func valueCases() []testCase {
	return []testCase{
		{
			"package constant",
			"package pkg\n\nconst c = 1\n",
			[]string{`constant "c" is a single letter`},
		},
		{
			"package variable",
			"package pkg\n\nvar x int\n",
			[]string{`variable "x" is a single letter`},
		},
		{
			"local var",
			"package pkg\n\nfunc run() { var x int; _ = x }\n",
			[]string{`variable "x" is a single letter`},
		},
		{
			"struct field",
			"package pkg\n\ntype Row struct{ X int }\n",
			[]string{`field "X" is a single letter`},
		},
	}
}

func statementCases() []testCase {
	return []testCase{
		{
			"short declaration",
			"package pkg\n\nfunc run() { x := 1; _ = x }\n",
			[]string{`variable "x" is a single letter`},
		},
		{
			"if init",
			"package pkg\n\nfunc run() { if x := 1; x > 0 {} }\n",
			[]string{`variable "x" is a single letter`},
		},
		{
			"select receive",
			"package pkg\n\nfunc run(ch chan int) { select { case x := <-ch: _ = x } }\n",
			[]string{`variable "x" is a single letter`},
		},
		{
			"for clause variable",
			"package pkg\n\nfunc run() { for i := 0; i < 3; i++ {} }\n",
			[]string{`loop variable "i" is a single letter`},
		},
		{
			"range key and value",
			"package pkg\n\nfunc run(rows []int) { for i, v := range rows { _, _ = i, v } }\n",
			[]string{
				`loop variable "i" is a single letter`,
				`loop variable "v" is a single letter`,
			},
		},
	}
}

func bindingCases() []testCase {
	return []testCase{
		{
			"type switch binding",
			"package pkg\n\nfunc run(val any) { switch v := val.(type) { case int: _ = v } }\n",
			[]string{`type switch binding "v" is a single letter`},
		},
		{
			"label",
			"package pkg\n\nfunc run() {\nL:\n\tfor {\n\t\tbreak L\n\t}\n}\n",
			[]string{`label "L" is a single letter`},
		},
		{
			"import alias",
			"package pkg\n\nimport f \"fmt\"\n\nvar _ = f.Sprint\n",
			[]string{`import alias "f" is a single letter`},
		},
		{
			"a use is not a declaration",
			"package pkg\n\nfunc run(rows []int) { for idx := range rows { _ = rows[idx] } }\n",
			nil,
		},
		{
			"multi-byte single character counts as one rune",
			"package pkg\n\nvar é int\n",
			[]string{"variable \"é\" is a single letter"},
		},
	}
}

func receiverCases() []testCase {
	return []testCase{
		{
			"receiver type parameter on a pointer receiver",
			"package pkg\n\ntype Box[Item any] struct{}\n\nfunc (box *Box[T]) Get() {}\n",
			[]string{`type parameter "T" is a single letter`},
		},
		{
			"receiver type parameters on a value receiver",
			"package pkg\n\ntype Pair[Key, Val any] struct{}\n\n" +
				"func (pair Pair[K, V]) Swap() {}\n",
			[]string{
				`type parameter "K" is a single letter`,
				`type parameter "V" is a single letter`,
			},
		},
		{
			"parenthesised pointer receiver type parameter",
			"package pkg\n\ntype Box[Item any] struct{}\n\nfunc (box (*Box[T])) Get() {}\n",
			[]string{`type parameter "T" is a single letter`},
		},
		{
			"blank receiver type parameter",
			"package pkg\n\ntype Box[Item any] struct{}\n\nfunc (box Box[_]) Get() {}\n",
			nil,
		},
		{
			"named receiver type parameters",
			"package pkg\n\ntype Pair[Key, Val any] struct{}\n\n" +
				"func (pair Pair[Key, Val]) Swap() {}\n",
			nil,
		},
		{
			"non-generic receiver",
			"package pkg\n\ntype Srv struct{}\n\nfunc (srv Srv) Do() {}\n",
			nil,
		},
		{
			"receiver-less method without a name",
			"package pkg\n\ntype Srv struct{}\n\nfunc (Srv) Do() {}\n",
			nil,
		},
	}
}

func importAndPackageCases() []testCase {
	return []testCase{
		{
			"dot import is not an alias",
			"package pkg\n\nimport . \"fmt\"\n\nvar _ = Sprint\n",
			nil,
		},
		{
			"blank import is not an alias",
			"package pkg\n\nimport _ \"fmt\"\n",
			nil,
		},
		{
			"non-ASCII letter alias",
			"package pkg\n\nimport π \"math\"\n\nvar _ = π.Pi\n",
			[]string{`import alias "π" is a single letter`},
		},
		{
			"single letter package name",
			"package z\n",
			[]string{`package "z" is a single letter`},
		},
		{
			"two letter package name",
			"package zz\n",
			nil,
		},
	}
}
