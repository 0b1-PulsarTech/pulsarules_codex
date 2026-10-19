package facts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func single(name string, kind syntax.FunctionKind, line, params int) syntax.Function {
	return syntax.Function{
		Name: name, Kind: kind, Line: line, BodyStart: line, BodyEnd: line,
		Params: params, Complexity: 1,
	}
}

type functionCase struct {
	name string
	text string
	want []syntax.Function
}

func functionCases() []functionCase {
	return slices.Concat(declarationCases(), boundCases(), closureCases(), bodyCases())
}

func declarationCases() []functionCase {
	return slices.Concat(signatureCases(), memberCases())
}

func signatureCases() []functionCase {
	return []functionCase{
		{
			name: "declaration",
			text: "\nfunction run(a, b) {\n  return a;\n}\n",
			want: []syntax.Function{{
				Name: "run", Kind: syntax.FuncFunction, Line: 2, BodyStart: 2, BodyEnd: 4,
				Params: 2, Complexity: 1,
			}},
		},
		{
			name: "default export without a name",
			text: "export default function () {}\n",
			want: []syntax.Function{{
				Name: "default", Kind: syntax.FuncFunction, Line: 1, BodyStart: 1, BodyEnd: 1,
				Complexity: 1,
			}},
		},
		{
			name: "overload and ambient signatures have no body",
			text: "function f(a: number): void;\ndeclare function g(): void;\nabstract class A " +
				"{ abstract m(): void }\n",
		},
	}
}

func memberCases() []functionCase {
	return slices.Concat(classMemberCases(), thisParamCases())
}

func classMemberCases() []functionCase {
	return []functionCase{
		{
			name: "class members",
			text: "class A {\n  constructor(x) {}\n  get v() { return 1 }\n  set v(x) {}\n  " +
				"m(a) {}\n  #p() {}\n  'q-r'() {}\n  [key]() {}\n}\n",
			want: []syntax.Function{
				single("A", syntax.FuncConstructor, 2, 1),
				single("v", syntax.FuncAccessor, 3, 0),
				single("v", syntax.FuncAccessor, 4, 1),
				single("m", syntax.FuncMethod, 5, 1),
				single("#p", syntax.FuncMethod, 6, 0),
				single("q-r", syntax.FuncMethod, 7, 0),
				single("[key]", syntax.FuncMethod, 8, 0),
			},
		},
	}
}

func thisParamCases() []functionCase {
	return []functionCase{
		{
			name: "this parameter is not counted and destructuring is one",
			text: "function f(this: Foo, { a, b }, [c, d], ...rest) {}\n",
			want: []syntax.Function{{
				Name: "f", Kind: syntax.FuncFunction, Line: 1, BodyStart: 1, BodyEnd: 1,
				Params: 3, Complexity: 1,
			}},
		},
	}
}

func boundCases() []functionCase {
	return slices.Concat(bindingCases(), assignedCases())
}

func bindingCases() []functionCase {
	return []functionCase{
		{
			name: "bound closures keep the binding name and are not nested",
			text: "const a = () => {};\nconst b = function () {};\nconst c = (() => {}) as X;\n" +
				"class K { d = () => {}; }\nconst o = { e: () => {} };\nexport default () => " +
				"{};\nconst f = function named() {};\n",
			want: []syntax.Function{
				single("a", syntax.FuncClosure, 1, 0),
				single("b", syntax.FuncClosure, 2, 0),
				single("c", syntax.FuncClosure, 3, 0),
				single("d", syntax.FuncClosure, 4, 0),
				single("e", syntax.FuncClosure, 5, 0),
				single("default", syntax.FuncClosure, 6, 0),
				single("named", syntax.FuncClosure, 7, 0),
			},
		},
	}
}

func assignedCases() []functionCase {
	return []functionCase{
		{
			name: "assignment targets name a closure",
			text: "target = () => {};\nexports.api = function () {};\nthis.state.handler = () " +
				"=> {};\ncount += () => {};\n",
			want: []syntax.Function{
				single("target", syntax.FuncClosure, 1, 0),
				single("api", syntax.FuncClosure, 2, 0),
				single("handler", syntax.FuncClosure, 3, 0),
				single("closure", syntax.FuncClosure, 4, 0),
			},
		},
	}
}

func closureCases() []functionCase {
	return slices.Concat(topLevelCallbackCases(), nestedCallbackCases())
}

func topLevelCallbackCases() []functionCase {
	return []functionCase{
		{
			name: "top level anonymous callbacks are their own functions",
			text: "run(() => {});\nrun(function () {});\n",
			want: []syntax.Function{
				single("closure", syntax.FuncClosure, 1, 0),
				single("closure", syntax.FuncClosure, 2, 0),
			},
		},
	}
}

func nestedCallbackCases() []functionCase {
	return []functionCase{
		{
			name: "anonymous callbacks inside a function are nested closures",
			text: "function f() {\n  run(() => {});\n}\n",
			want: []syntax.Function{
				{
					Name:       "f",
					Kind:       syntax.FuncFunction,
					Line:       1,
					BodyStart:  1,
					BodyEnd:    3,
					Complexity: 1,
				},
				{
					Name:       "closure",
					Kind:       syntax.FuncClosure,
					Line:       2,
					BodyStart:  2,
					BodyEnd:    2,
					Complexity: 1,
					Nested:     true,
				},
			},
		},
		{
			name: "expression body spans its expression",
			text: "const f = (a) =>\n  a +\n  1;\n",
			want: []syntax.Function{{
				Name: "f", Kind: syntax.FuncClosure, Line: 1, BodyStart: 2, BodyEnd: 3,
				Params: 1, Complexity: 1,
			}},
		},
	}
}

func bodyCases() []functionCase {
	return slices.Concat(nestingCases(), declaredBodyCases())
}

func nestingCases() []functionCase {
	return []functionCase{
		{
			name: "nested function declaration is not nested",
			text: "function outer() {\n  function inner() {}\n}\n",
			want: []syntax.Function{
				{
					Name:       "outer",
					Kind:       syntax.FuncFunction,
					Line:       1,
					BodyStart:  1,
					BodyEnd:    3,
					Complexity: 1,
				},
				single("inner", syntax.FuncFunction, 2, 0),
			},
		},
	}
}

func declaredBodyCases() []functionCase {
	return slices.Concat(overloadCases(), constructorOwnerCases(), decoratedCases())
}

func overloadCases() []functionCase {
	return []functionCase{
		{
			name: "overload signatures emit one function",
			text: "function f(a: string): void;\nfunction f(a: number): void;\nfunction f(a: " +
				"any) {}\nclass B {\n  m(x: string): void;\n  m(x: any) {}\n  constructor(x: " +
				"string);\n  constructor(x: any) {}\n}\ndeclare function g(): void;\n",
			want: []syntax.Function{
				single("f", syntax.FuncFunction, 3, 1),
				single("m", syntax.FuncMethod, 6, 1),
				single("B", syntax.FuncConstructor, 8, 1),
			},
		},
	}
}

func constructorOwnerCases() []functionCase {
	return []functionCase{
		{
			name: "export default class constructor",
			text: "export default class {\n  constructor() {}\n}\n",
			want: []syntax.Function{single("default", syntax.FuncConstructor, 2, 0)},
		},
		{
			name: "class expression takes the binding name",
			text: "const Anon = class {\n  constructor() {}\n};\n",
			want: []syntax.Function{single("Anon", syntax.FuncConstructor, 2, 0)},
		},
		{
			name: "class expression takes the assigned property",
			text: "ns.Made = class {\n  constructor() {}\n};\n",
			want: []syntax.Function{single("Made", syntax.FuncConstructor, 2, 0)},
		},
		{
			name: "named class expression keeps its own name",
			text: "const Anon = class Own {\n  constructor() {}\n};\n",
			want: []syntax.Function{single("Own", syntax.FuncConstructor, 2, 0)},
		},
		{
			name: "unbound class expression is the default",
			text: "run(class {\n  constructor() {}\n});\n",
			want: []syntax.Function{single("default", syntax.FuncConstructor, 2, 0)},
		},
	}
}

func decoratedCases() []functionCase {
	return []functionCase{
		{
			name: "decorated method starts at its name",
			text: "class A {\n  @dec()\n  m() {}\n}\n",
			want: []syntax.Function{
				single("m", syntax.FuncMethod, 3, 0),
			},
		},
	}
}

func TestFunctions(t *testing.T) {
	t.Parallel()

	testCases := functionCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := functionsOf(t, testCase.text, Options{}).out
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("functions\n got %+v\nwant %+v", got, testCase.want)
			}
		})
	}
}
