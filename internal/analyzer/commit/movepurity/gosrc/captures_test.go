package gosrc

import (
	"go/ast"
	"slices"
	"testing"
)

func TestCaptureSets_Captures(t *testing.T) {
	t.Parallel()

	sets := CaptureSets{
		Left:  map[string]bool{"max": true, "Foo": true},
		Taken: map[string]bool{"Bar": true},
	}
	testCases := []struct {
		name string
		ref  string
		want bool
	}{
		{"taken in the new directory", "Bar", true},
		{"left behind and predeclared", "max", true},
		{"left behind only", "Foo", false},
		{"neither", "Baz", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := sets.captures(testCase.ref); got != testCase.want {
				t.Fatalf("captures(%q) = %v, want %v", testCase.ref, got, testCase.want)
			}
		})
	}
}

func TestCapturesBare(t *testing.T) {
	t.Parallel()

	file := mustParse(t, "package p\n\nfunc F() int { return Foo(1) + local }\n\nvar local = 1\n")
	taken := Binding{Captures: CaptureSets{Taken: map[string]bool{"Foo": true, "local": true}}}
	testCases := []struct {
		name string
		lit  string
		bind Binding
		want bool
	}{
		{"unresolved and taken", "Foo", taken, true},
		{"unresolved and free", "Foo", Binding{}, false},
		{"resolved in the file", "local", taken, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pos := -1
			for idx, tok := range file.body {
				if tok.lit == testCase.lit {
					pos = idx
					break
				}
			}
			if pos < 0 {
				t.Fatalf("no token %q", testCase.lit)
			}
			if got := capturesBare(file, pos, testCase.bind); got != testCase.want {
				t.Fatalf("capturesBare() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestRelocationUnsafe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want bool
	}{
		{"plain declarations", "func F() int { return 1 }\n", false},
		{"TestMain is not load work", "func TestMain(m *testing.M) { m.Run() }\n", false},
		{"init is not unsafe by itself", "func init() {}\n", false},
		{"dot import", "import . \"strings\"\n\nvar V = ToUpper(\"a\")\n", true},
		{"cgo", "import \"C\"\n\nvar V = C.VAL\n", true},
		{"predeclared name", "func max(a, b int) int { return a }\n", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := mustParse(t, "package p\n\n"+testCase.body)
			if got := RelocationUnsafe(file, file); got != testCase.want {
				t.Fatalf("RelocationUnsafe() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestRelocationUnsafe_OldFileImports(t *testing.T) {
	t.Parallel()

	cgo := mustParse(t, "package p\n\nimport \"C\"\n")
	plain := mustParse(t, "package p\n")
	if !RelocationUnsafe(cgo, plain) || !RelocationUnsafe(plain, cgo) {
		t.Fatal("an import C on either side must be unsafe")
	}
}

func TestLoads(t *testing.T) {
	t.Parallel()

	testCases := loadsCases()
	runLoadCases(t, testCases)
}

func loadsCases() []loadCase {
	return slices.Concat(loadsBasicCases(), loadsExtendedCases())
}

func loadsBasicCases() []loadCase {
	return []loadCase{
		{"init", "func init() {}\n", true},
		{
			"blank var with a call",
			"var _ = register()\n\nfunc register() int { return 1 }\n",
			true,
		},
		{
			"blank var with a nested call",
			"var _ = []int{register()}\n\nfunc register() int { return 1 }\n",
			true,
		},
		{
			"blank var pair with a call",
			"var _, _ = pair()\n\nfunc pair() (int, int) { return 1, 2 }\n",
			true,
		},
		{"blank interface assertion", "type T struct{}\n\nvar _ interface{} = (*T)(nil)\n", false},
		{"blank pointer conversion", "type T struct{}\n\nvar _ = (*T)(nil)\n", false},
		{
			"pointer conversion of a package var",
			"var s = []byte{1}\n\nvar _ = (*[4]byte)(s)\n",
			true,
		},
		{
			"pointer conversion of a shadowed nil",
			"var nil = []byte{1}\n\nvar _ = (*[4]byte)(nil)\n",
			true,
		},
		{
			"dot import frees the base name",
			"import . \"example.com/m/reg\"\n\nvar _ = reg.M\n",
			true,
		},
		{"blank slice conversion", "var _ = []byte(\"x\")\n", false},
		{"blank map conversion", "var _ = map[string]int(nil)\n", false},
		{"blank func conversion", "var _ = (func())(nil)\n", false},
		{"blank chan conversion", "var _ = (chan int)(nil)\n", false},
		{"blank interface conversion", "var _ = interface{}(nil)\n", false},
		{"blank struct conversion", "var _ = struct{}(struct{}{})\n", false},
		{"blank universe conversion", "var x = 1\n\nvar _ = []any{string(rune(x))}\n", false},
	}
}

func loadsExtendedCases() []loadCase {
	return []loadCase{
		{"blank parenthesised universe conversion", "var x = 1\n\nvar _ = (int)(x)\n", false},
		{
			"blank call then a conversion",
			"var _ = []any{f(), []byte(\"x\")}\n\nfunc f() int { return 1 }\n",
			true,
		},
		{
			"blank conversion then a call",
			"var _ = []any{[]byte(\"x\"), f()}\n\nfunc f() int { return 1 }\n",
			true,
		},
		{"blank deref of nil conversion", "type T struct{}\n\nvar _ = (*T)(nil)\n", false},
		{"blank parenthesised literal call", "var _ = (func() int { return reg() })()\n", true},
		{
			"blank parenthesised selector call",
			"import \"reg\"\n\nvar _ = (reg.Add)(\"a\")\n",
			true,
		},
		{
			"blank literal call in a composite",
			"type S struct{ B bool }\n\nvar _ = S{B: (func() bool { return reg() })()}\n",
			true,
		},
		{
			"blank function literal",
			"var _ = func() int { return register() }\n\nfunc register() int { return 1 }\n",
			false,
		},
		{
			"named var with a call",
			"var X = register()\n\nfunc register() int { return 1 }\n",
			true,
		},
		{"named var with a conversion", "var X = int64(1)\n", false},
		{"named var pair with a call", "var a, b = pair()\n", true},
		{"blank var without a value", "var _ int\n", false},
		{"constant blank", "const _ = 1\n", false},
		{"blank import", "import _ \"example.com/reg\"\n", true},
		{"blank import of embed", "import _ \"embed\"\n", false},
		{"named import", "import reg \"example.com/reg\"\n\nvar _ = reg.X\n", false},
		{"dot import", "import . \"example.com/reg\"\n", false},
	}
}

func TestLoads_Allowlist(t *testing.T) {
	t.Parallel()

	const reg = "import \"example.com/m/reg\"\n\n"
	testCases := []loadCase{
		{"channel receive", reg + "var _ = <-reg.Ch\n", true},
		{"method value on a variable", reg + "var _ = reg.P.M\n", true},
		{"type assertion", reg + "var _ = reg.V.(int)\n", true},
		{"index", "var xs = []int{1}\n\nvar _ = xs[0]\n", true},
		{"slice", "var xs = []int{1}\n\nvar _ = xs[:]\n", true},
		{"dereference", "var p *int\n\nvar _ = *p\n", true},
		{
			"generic instantiation call",
			"func take[T any]() int { return 1 }\n\nvar _ = take[int]()\n",
			true,
		},
		{"receive nested in a composite", reg + "var _ = []any{1, <-reg.Ch}\n", true},
		{"receive under a binary operator", reg + "var _ = 1 + <-reg.Ch\n", true},
		{"conversion of a call", "var _ = int(f())\n\nfunc f() int { return 1 }\n", true},
		{
			"selector shadowed by a package-level name",
			reg + "var reg struct{ N int }\n\nvar _ = reg.N\n",
			true,
		},
		{"parenthesised receive", reg + "var _ = (<-reg.Ch)\n", true},
		{"receive as the left operand", reg + "var _ = <-reg.Ch + 1\n", true},
		{"selector on an undeclared name", "var _ = p.M\n", true},
		{"call as a map key", "var _ = map[int]int{f(): 1}\n\nfunc f() int { return 1 }\n", true},
		{"call under a unary operator", "var _ = -f()\n\nfunc f() int { return 1 }\n", true},
		{"slice to array conversion", reg + "var _ = [4]byte(reg.S)\n", true},
		{"slice to array pointer conversion", reg + "var _ = (*[4]byte)(reg.S)\n", true},
		{"pointer conversion of a value", reg + "type T struct{}\n\nvar _ = (*T)(reg.P)\n", true},
		{"slice conversion", reg + "var _ = []byte(reg.S)\n", false},
		{"division by a variable", reg + "var _ = 1 / reg.Zero\n", true},
		{"remainder by a variable", reg + "var _ = 1 % reg.Zero\n", true},
		{"left shift by a variable", reg + "var _ = 1 << reg.Neg\n", true},
		{"right shift by a variable", reg + "var _ = 1 >> reg.Neg\n", true},
		{"division by a literal", reg + "var _ = reg.X / 2\n", false},
		{"shift by a literal", reg + "var _ = reg.X << 2\n", false},
		{
			"aliased import frees the base name",
			"import r \"example.com/m/reg\"\n\nvar _ = reg.M\n",
			true,
		},
		{"aliased import selector", "import r \"example.com/m/reg\"\n\nvar _ = r.X\n", false},
		{"package-qualified identifier", reg + "var _ = reg.Ch\n", false},
		{
			"iota constants and a mixed var block",
			"const (\n\tK0 = iota\n\tK1\n)\n\nvar (\n\tv1 = K1\n\tv2 string = \"x\"\n" +
				"\t_ = v1\n\t_ = v2\n)\n",
			false,
		},
		{"unary and binary over inert operands", "var a = 1\n\nvar _ = -a + 2*a\n", false},
		{"address of a composite", "type T struct{ N int }\n\nvar _ = &T{N: 1}\n", false},
		{"keyed map literal", "var _ = map[string]int{\"a\": 1}\n", false},
	}
	runLoadCases(t, testCases)
}

func TestLoads_Errors(t *testing.T) {
	t.Parallel()

	testCases := []loadCase{
		{"errors.New", "import \"errors\"\n\nvar E = errors.New(\"x\")\n", false},
		{"aliased errors.New", "import e \"errors\"\n\nvar E = e.New(\"x\")\n", false},
		{"dot-imported errors.New", "import . \"errors\"\n\nvar E = New(\"x\")\n", true},
		{"errors.Join", "import \"errors\"\n\nvar E = errors.Join(nil, nil)\n", true},
		{
			"errors.New with a call argument",
			"import \"errors\"\n\nvar E = errors.New(f())\n",
			true,
		},
		{"another package's New", "import \"other\"\n\nvar E = other.New(\"x\")\n", true},
		{
			"another package's New imported before errors",
			"import (\n\t\"other\"\n\t\"errors\"\n)\n\nvar E = other.New(\"x\")\n" +
				"var F = errors.New(\"y\")\n",
			true,
		},
		{
			"another package named errors",
			"import \"example.com/errors\"\n\nvar E = errors.New(\"x\")\n",
			true,
		},
		{
			"errors imported after another package",
			"import (\n\t\"other\"\n\t\"errors\"\n)\n\nvar E = errors.New(\"x\")\n" +
				"var _ = other.X\n",
			false,
		},
	}
	runLoadCases(t, testCases)
}

type loadCase struct {
	name string
	body string
	want bool
}

func runLoadCases(t *testing.T, testCases []loadCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := mustParse(t, "package p\n\n"+testCase.body).Loads; got != testCase.want {
				t.Fatalf("Loads = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsInert_ShadowedUniverse(t *testing.T) {
	t.Parallel()

	fun := &ast.Ident{Name: "string"}
	if !isInert(fun, loadEnv{}) || isInert(fun, loadEnv{top: map[string]bool{"string": true}}) {
		t.Fatal("a predeclared type is a conversion unless the package shadows it")
	}
}
