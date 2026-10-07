package testpolicy

import "testing"

type testCase struct {
	name string
	path string
	want bool
}

func assertIsTest(t *testing.T, testCases []testCase) {
	t.Helper()

	for _, policy := range []Policy{Defaults(), {}} {
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()
				if got := policy.IsTest(testCase.path); got != testCase.want {
					t.Errorf("IsTest(%q) = %v, want %v", testCase.path, got, testCase.want)
				}
			})
		}
	}
}

func TestDefaultsIsTestWeb(t *testing.T) {
	t.Parallel()

	testCases := []testCase{
		{"mjs under web test dir", "web/test/a.mjs", true},
		{"ts dot test", "src/a.test.ts", true},
		{"svelte dot spec", "src/a.spec.svelte", true},
		{"svelte route under test dir", "src/routes/test/+page.svelte", false},
		{"svelte under tests dir", "web/tests/A.svelte", false},
		{"ts route under test dir", "src/routes/test/+page.ts", false},
		{"ts server route under test dir", "src/routes/test/+page.server.ts", false},
		{"ts route dot test", "src/routes/test/+page.test.ts", true},
		{"mjs under web test dir stays", "web/test/foo.mjs", true},
		{"svelte dot test", "src/routes/test/a.test.svelte", true},
		{"ts under tests dir", "web/tests/e2e/x.ts", true},
		{"ts name with test prefix only", "src/test_utils.ts", false},
		{"plain ts", "src/a.ts", false},
		{"ts upper-case test dir keeps lowering", "web/Test/a.ts", true},
	}
	assertIsTest(t, testCases)
}

func TestDefaultsIsTestOther(t *testing.T) {
	t.Parallel()

	testCases := []testCase{
		{"js under dunder tests", "lib/__tests__/x.js", true},
		{"svelte under dunder tests dir", "src/__tests__/A.svelte", false},
		{"dart helper under test dir", "packages/x/test/helpers.dart", true},
		{"dart test suffix", "lib/a_test.dart", true},
		{"dart plain", "lib/a.dart", false},
		{"dart dot test is not a marker", "lib/a.test.dart", false},
		{"go test suffix", "a_test.go", true},
		{"go helper under tests dir", "tests/internal/apphost/boot.go", false},
		{"go helper under test dir", "test/helpers.go", false},
		{"go upper-case suffix", "pkg/BAR_TEST.GO", false},
		{"go mixed-case suffix", "pkg/Foo_Test.go", false},
		{"go golden suffix", "pkg/snap_test.golden.go", false},
		{"dart upper-case suffix", "lib/a_TEST.dart", false},
		{"dart integration_test dir", "app/integration_test/flow.dart", true},
		{"dart nested integration_test dir", "packages/x/integration_test/a/b.dart", true},
		{"dart upper-case test dir", "app/Test/w.dart", false},
		{"dart upper-case integration_test dir", "app/Integration_Test/x.dart", false},
		{"dart integration_test as file stem", "lib/integration_test_utils.dart", false},
		{"other extension dot test", "docs/a.test.md", true},
		{"other extension spec", "src/foo_spec.rb", true},
		{"other extension under test dir", "test/data.json", false},
		{"other extension plain", "src/foo.rb", false},
	}
	assertIsTest(t, testCases)
}
