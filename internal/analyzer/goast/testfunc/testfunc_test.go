package testfunc

import (
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func lastFunc(t *testing.T, source string) (*ast.File, *ast.FuncDecl) {
	t.Helper()
	cache := astcache.New()
	file, err := cache.Parse("p_test.go", []byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("last decl is not a func")
	}
	return file, fn
}

type coversCase struct {
	name       string
	source     string
	isTestFile bool
	want       bool
}

func TestCoversFunc(t *testing.T) {
	t.Parallel()

	cases := append(runnableCases(), helperCases()...)
	for _, testCase := range append(cases, receiverCases()...) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, fn := lastFunc(t, testCase.source)
			fc := pass.FileChange{IsTest: testCase.isTestFile}
			if got := Covers(fc, file, fn); got != testCase.want {
				t.Fatalf("CoversFunc = %v, want %v", got, testCase.want)
			}
		})
	}
}

func runnableCases() []coversCase {
	return []coversCase{
		{
			"runnable test",
			"package p\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n",
			true,
			true,
		},
		{
			"TestMain",
			"package p\nimport \"testing\"\nfunc TestMain(m *testing.M) {}\n",
			true,
			true,
		},
		{
			"benchmark",
			"package p\nimport \"testing\"\nfunc BenchmarkA(b *testing.B) {}\n",
			true,
			true,
		},
		{"fuzz", "package p\nimport \"testing\"\nfunc FuzzA(f *testing.F) {}\n", true, true},
		{"example", "package p\nfunc ExampleA() {}\n", true, true},
		{"example with a suffix", "package p\nfunc ExampleA_b() {}\n", true, true},
		{"lowercase suffix is not a test", "package p\nfunc Testimony() {}\n", true, false},
		{"test with the wrong param", "package p\nfunc TestA(x int) {}\n", true, false},
		{
			"test with two params is a helper",
			"package p\nimport \"testing\"\nfunc TestA(t *testing.T, x int) {}\n",
			true,
			true,
		},
		{"test with no params", "package p\nfunc TestA() {}\n", true, false},
	}
}

func receiverCases() []coversCase {
	return []coversCase{
		{"helper by field t", "package p\nfunc (s S) h() { s.t.Helper() }\n", true, true},
		{"helper by field T", "package p\nfunc (s S) h() { s.T.Helper() }\n", true, true},
		{"helper by T() call", "package p\nfunc (s S) h() { s.T().Helper() }\n", true, true},
		{
			"Helper on a non-testing menu type",
			"package p\nfunc h(m M) { m.Helper() }\n",
			true,
			false,
		},
		{
			"dot import handle",
			"package p\nimport . \"testing\"\nfunc h(t *T) {}\n",
			true,
			true,
		},
		{
			"dot import TB",
			"package p\nimport . \"testing\"\nfunc h(t TB) {}\n",
			true,
			true,
		},
		{"bare T without dot import", "package p\nfunc h(t *T) {}\n", true, false},
		{
			"dot import runnable test",
			"package p\nimport . \"testing\"\nfunc TestA(t *T) {}\n",
			true,
			true,
		},
		{"suite method", "package p\nfunc (s *suite) TestX() {}\n", true, true},
		{
			"suite method with a lowercase suffix",
			"package p\nfunc (s *suite) Testx() {}\n",
			true,
			false,
		},
		{
			"method with a result is not a suite method",
			"package p\nfunc (c *Client) TestConnection() error { return nil }\n",
			true,
			false,
		},
		{
			"suite method outside a test file",
			"package p\nfunc (s *suite) TestX() {}\n",
			false,
			false,
		},
	}
}

func helperCases() []coversCase {
	return []coversCase{
		{
			"helper by testing.TB",
			"package p\nimport \"testing\"\nfunc h(tb testing.TB) {}\n",
			true,
			true,
		},
		{
			"helper method",
			"package p\nimport \"testing\"\nfunc (s S) h(t *testing.T) {}\n",
			true,
			true,
		},
		{
			"helper by aliased testing import",
			"package p\nimport tst \"testing\"\nfunc h(t *tst.T) {}\n",
			true,
			true,
		},
		{"helper by require.TestingT", "package p\nfunc h(t require.TestingT) {}\n", true, true},
		{"helper by assert.TestingT", "package p\nfunc h(t assert.TestingT) {}\n", true, true},
		{
			"other package T is not a testing handle",
			"package p\nfunc h(t other.T) {}\n",
			true,
			false,
		},
		{
			"method with no testing param and no Helper call",
			"package p\nfunc (s S) h() {}\n",
			true,
			false,
		},
		{"plain func", "package p\nfunc plain() {}\n", true, false},
		{
			"runnable test outside a test file",
			"package p\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n",
			false,
			false,
		},
		{"Helper call with an argument", "package p\nfunc h(x X) { x.Helper(1) }\n", true, false},
		{"bodyless declaration", "package p\nfunc h()\n", true, false},
	}
}

func TestKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{"test", "package p\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n", "Test"},
		{"TestMain", "package p\nimport \"testing\"\nfunc TestMain(m *testing.M) {}\n", "Test"},
		{"example", "package p\nfunc ExampleA() {}\n", KindExample},
		{
			"method is never runnable",
			"package p\nimport \"testing\"\nfunc (S) TestA(t *testing.T) {}\n",
			"",
		},
		{"helper is no kind", "package p\nimport \"testing\"\nfunc h(t *testing.T) {}\n", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, fn := lastFunc(t, testCase.source)
			if got, _ := Kind(file, fn); got != testCase.want {
				t.Fatalf("Kind = %q, want %q", got, testCase.want)
			}
		})
	}
}
