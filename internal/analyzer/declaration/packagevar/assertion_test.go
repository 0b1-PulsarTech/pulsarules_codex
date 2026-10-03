package packagevar

import (
	"slices"
	"testing"
)

func assertionCases() []levelVarCase {
	const typed = "package p\n\ntype T struct{}\ntype I interface{}\n\n" +
		"func register() bool { return true }\n\n"
	return []levelVarCase{
		{
			name:  "typed blank with a call flagged",
			files: map[string]string{"a.go": typed + "var _ bool = register()\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name: "error typed blank with a call flagged",
			files: map[string]string{
				"a.go": typed + "func initDB() error { return nil }\n\nvar _ error = initDB()\n",
			},
			want: []string{"a.go:10:package-level var _"},
		},
		{
			name:  "typed blank pointer nil conversion exempt",
			files: map[string]string{"a.go": typed + "var _ I = (*T)(nil)\n"},
		},
		{
			name:  "typed blank composite exempt",
			files: map[string]string{"a.go": typed + "var _ I = T{}\n"},
		},
		{
			name:  "typed blank address of composite exempt",
			files: map[string]string{"a.go": typed + "var _ I = &T{}\n"},
		},
		{
			name:  "typed blank without a value exempt",
			files: map[string]string{"a.go": typed + "var _ I\n"},
		},
	}
}

func conversionCases() []levelVarCase {
	const typed = "package p\n\ntype T struct{}\ntype I interface{}\n\n" +
		"func register() bool { return true }\n\n"
	return []levelVarCase{
		{
			name: "call taking a composite flagged",
			files: map[string]string{
				"a.go": typed + "var _ = register2(T{})\n\nfunc register2(T) int { return 0 }\n",
			},
			want: []string{"a.go:8:package-level var _"},
		},
		{
			name: "package function taking an address of composite flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/caddy\"\n\ntype M struct{}\n\n" +
					"var _ = caddy.Register(&M{})\n",
			},
			want: []string{"a.go:7:package-level var _"},
		},
		{
			name: "generic call taking a composite flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype E struct{}\n\nfunc mk[V any](V) int { return " +
					"0 }\n\nvar _ = mk[E](E{})\n",
			},
			want: []string{"a.go:7:package-level var _"},
		},
		{
			name:  "parenthesized function call flagged",
			files: map[string]string{"a.go": typed + "var _ = (register)()\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "conversion to a local type exempt",
			files: map[string]string{"a.go": typed + "var _ = I(T{})\n"},
		},
		{
			name: "conversion to a local type declared in a sibling file exempt",
			files: map[string]string{
				"a.go": "package p\n\nvar _ = Iface(struct{}{})\n",
				"b.go": "package p\n\ntype Iface interface{}\n",
			},
		},
		{
			name:  "conversion to a predeclared type exempt",
			files: map[string]string{"a.go": typed + "var _ = any(T{})\n"},
		},
		{
			name: "conversion to a package type flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/pkg\"\n\ntype T struct{}\n\n" +
					"var _ = pkg.Iface(T{})\n",
			},
			want: []string{"a.go:7:package-level var _"},
		},
		{
			name:  "conversion with two arguments flagged",
			files: map[string]string{"a.go": typed + "var _ = I(T{}, T{})\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "conversion of a call flagged",
			files: map[string]string{"a.go": typed + "var _ = I(register())\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "lastErr without a value flagged",
			files: map[string]string{"a.go": "package p\n\nvar lastErr error\n"},
			want:  []string{"a.go:3:package-level var lastErr"},
		},
	}
}

func callInCompositeCases() []levelVarCase {
	const pre = "package p\n\ntype T struct{ db int }\ntype I interface{}\n\n" +
		"func openDB() int { return 0 }\n\n"
	return []levelVarCase{
		{
			name:  "typed blank composite with a call field flagged",
			files: map[string]string{"a.go": pre + "var _ I = T{db: openDB()}\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "blank composite with a call field flagged",
			files: map[string]string{"a.go": pre + "var _ = T{db: openDB()}\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "blank slice composite with a call element flagged",
			files: map[string]string{"a.go": pre + "var _ = []int{openDB()}\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "conversion of a composite with a call field flagged",
			files: map[string]string{"a.go": pre + "var _ = I(T{db: openDB()})\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "address of composite with a call field flagged",
			files: map[string]string{"a.go": pre + "var _ I = &T{db: openDB()}\n"},
			want:  []string{"a.go:8:package-level var _"},
		},
		{
			name:  "composite with a literal field exempt",
			files: map[string]string{"a.go": pre + "var _ I = T{db: 1}\n"},
		},
		{
			name:  "composite with a conversion field exempt",
			files: map[string]string{"a.go": pre + "var _ I = T{db: int(2)}\n"},
		},
		{
			name:  "composite with a func literal field exempt",
			files: map[string]string{"a.go": pre + "var _ = []func(){func() { openDB() }}\n"},
		},
		{
			name:  "parenthesized slice type conversion exempt",
			files: map[string]string{"a.go": pre + "var _ = ([]int)(nil)\n"},
		},
		{
			name: "dereference of a func variable conversion flagged",
			files: map[string]string{
				"a.go": pre + "var fp *func(any) I\n\nvar _ I = (*fp)(nil)\n",
			},
			want: []string{"a.go:8:package-level var fp", "a.go:10:package-level var _"},
		},
		{
			name: "pointer conversion to a generic type exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype G[A, B any] struct{}\ntype H[A any] struct{}\n\n" +
					"var _ = (*G[int, string])(nil)\n\nvar _ = (*H[int])(nil)\n",
			},
		},
		{
			name: "pointer conversion to an index of a variable flagged",
			files: map[string]string{
				"a.go": "package p\n\nvar tbl []*int\n\nvar _ = (*tbl[0])(nil)\n",
			},
			want: []string{"a.go:3:package-level var tbl", "a.go:5:package-level var _"},
		},
		{
			name: "pointer conversion to an imported type exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"io\"\n\nvar _ = (*io.Reader)(nil)\n",
			},
		},
	}
}

func literalConversionCases() []levelVarCase {
	const pre = "package p\n\nimport (\n\t\"bytes\"\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n)\n\n" +
		"type Level int\ntype S string\ntype I interface{}\n\n" +
		"func F(int) I { return nil }\n\nvar x int\n\n"
	return []levelVarCase{
		{
			name:  "typed blank int literal conversion exempt",
			files: map[string]string{"a.go": pre + "var _ fmt.Stringer = Level(0)\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "string literal conversion exempt",
			files: map[string]string{"a.go": pre + "var _ I = S(\"\")\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "new of an imported type exempt",
			files: map[string]string{"a.go": pre + "var _ io.Writer = new(bytes.Buffer)\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "imported conversion of nil exempt",
			files: map[string]string{"a.go": pre + "var _ http.Handler = http.HandlerFunc(nil)\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "imported call with a non nil argument flagged",
			files: map[string]string{"a.go": pre + "var _ http.Handler = http.HandlerFunc(x)\n"},
			want:  []string{"a.go:16:package-level var x", "a.go:18:package-level var _"},
		},
		{
			name: "method call on a local variable with nil flagged",
			files: map[string]string{
				"a.go": pre + "var _ = cfg.Do(nil)\n\nvar cfg struct{ Do func(any) }\n",
			},
			want: []string{
				"a.go:16:package-level var x", "a.go:18:package-level var _",
				"a.go:20:package-level var cfg",
			},
		},
		{
			name:  "call with a variable argument flagged",
			files: map[string]string{"a.go": pre + "var _ I = F(x)\n"},
			want:  []string{"a.go:16:package-level var x", "a.go:18:package-level var _"},
		},
		{
			name:  "conversion of a variable flagged",
			files: map[string]string{"a.go": pre + "var _ I = S(x)\n"},
			want:  []string{"a.go:16:package-level var x", "a.go:18:package-level var _"},
		},
		{
			name:  "new of a non type flagged",
			files: map[string]string{"a.go": pre + "var _ = new(x)\n"},
			want:  []string{"a.go:16:package-level var x", "a.go:18:package-level var _"},
		},
	}
}

func signedLiteralCases() []levelVarCase {
	const pre = "package p\n\nimport (\n\t\"bytes\"\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n)\n\n" +
		"type Level int\ntype S string\ntype I interface{}\n\n" +
		"func F(int) I { return nil }\n\nvar x int\n\n"
	return []levelVarCase{
		{
			name:  "negative literal conversion exempt",
			files: map[string]string{"a.go": pre + "var _ I = Level(-1)\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "parenthesized positive literal conversion exempt",
			files: map[string]string{"a.go": pre + "var _ I = Level(+(1))\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "parenthesized signed literal argument exempt",
			files: map[string]string{"a.go": pre + "var _ I = Level((-1))\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "negated variable conversion flagged",
			files: map[string]string{"a.go": pre + "var _ I = Level(-x)\n"},
			want:  []string{"a.go:16:package-level var x", "a.go:18:package-level var _"},
		},
		{
			name:  "parenthesized interface conversion of nil exempt",
			files: map[string]string{"a.go": pre + "var _ = (io.Writer)(nil)\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name:  "interface conversion of a parenthesized nil exempt",
			files: map[string]string{"a.go": pre + "var _ = io.Writer((nil))\n"},
			want:  []string{"a.go:16:package-level var x"},
		},
		{
			name: "imported type conversion of a literal flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"time\"\n\nvar _ = time.Duration(0)\n",
			},
			want: []string{"a.go:5:package-level var _"},
		},
	}
}

func typeExprCases() []levelVarCase {
	const pre = "package p\n\ntype List[E any] []E\ntype Pair[A, B any] struct{}\n" +
		"type I interface{}\n\n"
	return []levelVarCase{
		{
			name:  "parenthesized slice under a pointer exempt",
			files: map[string]string{"a.go": pre + "var _ = (*([]int))(nil)\n"},
		},
		{
			name:  "pointer to a slice literal type exempt",
			files: map[string]string{"a.go": pre + "var _ = (*[]int)(nil)\n"},
		},
		{
			name:  "pointer to a func literal type exempt",
			files: map[string]string{"a.go": pre + "var _ = (*func())(nil)\n"},
		},
		{
			name:  "pointer to a pointer type exempt",
			files: map[string]string{"a.go": pre + "type T struct{}\n\nvar _ = (**T)(nil)\n"},
		},
		{
			name:  "doubly parenthesized pointer type exempt",
			files: map[string]string{"a.go": pre + "type T struct{}\n\nvar _ = ((*T))(nil)\n"},
		},
		{
			name:  "generic type conversion exempt",
			files: map[string]string{"a.go": pre + "var _ I = List[int](nil)\n"},
		},
		{
			name:  "generic type with two arguments conversion exempt",
			files: map[string]string{"a.go": pre + "var _ I = Pair[int, string](nil)\n"},
		},
		{
			name: "generic function instantiation flagged",
			files: map[string]string{
				"a.go": pre + "func mk[E any](v []E) I { return nil }\n\nvar _ = mk[int](nil)\n",
			},
			want: []string{"a.go:9:package-level var _"},
		},
		{
			name: "pointer to a field of a local variable flagged",
			files: map[string]string{
				"a.go": pre + "var cfg struct{ fn int }\n\nvar _ = (*cfg.fn)(nil)\n",
			},
			want: []string{"a.go:7:package-level var cfg", "a.go:9:package-level var _"},
		},
		{
			name: "type of the external test package found beside its siblings",
			files: map[string]string{
				"a_test.go": "package p_test\n\ntype T struct{}\n\nvar _ = (*T)(nil)\n",
			},
		},
	}
}

func TestAnalyzeAssertion(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(assertionCases(), conversionCases(), callInCompositeCases(),
		literalConversionCases(), signedLiteralCases(), typeExprCases()))
}
