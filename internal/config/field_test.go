package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestFieldPathCovered(t *testing.T) {
	t.Parallel()

	root := newRoot()
	root.key("skip_dirs").item(3).mark()
	root.key("analyzers").key("x").mark()
	root.key("a[0]").key("p.q").mark()
	testCases := []struct {
		name string
		at   *Field
		want bool
	}{
		{"the same field", root.key("analyzers").key("x"), true},
		{"a child by key", root.key("analyzers").key("x").key("params"), true},
		{"a child by index", root.key("skip_dirs").item(3).item(0), true},
		{"the list holding a skipped item", root.key("skip_dirs"), true},
		{"an unrelated field", root.key("tests"), false},
		{"a sibling item", root.key("skip_dirs").item(4), false},
		{"a longer name with the same prefix", root.key("analyzers").key("xy"), false},
		{"a key that spells an index", root.key("skip_dirs[3]"), false},
		{"a key that spells a dotted child", root.key("analyzers.x"), false},
		{"a key made of brackets", root.key("[").key("[[["), false},
		{"an item beside a bracketed key", root.key("a").item(0), false},
		{"a dotted key beside its halves", root.key("a[0]").key("p"), false},
		{"the dotted key itself", root.key("a[0]").key("p.q"), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.at.covered(new(int)); got != testCase.want {
				t.Fatalf("covered(%q) = %v, want %v", testCase.at, got, testCase.want)
			}
		})
	}
}

func TestFieldPathCoveredWhenEmpty(t *testing.T) {
	t.Parallel()

	if newRoot().key("tests").covered(new(int)) {
		t.Fatal("an unmarked path is covered")
	}
}

func TestFieldPathString(t *testing.T) {
	t.Parallel()

	root := newRoot()
	testCases := []struct {
		name string
		at   *Field
		want string
	}{
		{"the root", root, ""},
		{"a key", root.key("tests"), "tests"},
		{"keys", root.key("analyzers").key("x").key("params"), "analyzers.x.params"},
		{"an item", root.key("tests").item(2), "tests[2]"},
		{"an item first", root.item(0).key("a"), "[0].a"},
		{"an empty key", root.key("a").key(""), "a.<empty key>"},
		{"brackets in a key", root.key("[").key("a.b"), "[.a.b"},
		{"a control byte", root.key("a").key("\x01\x7f"), "a.\x01\x7f"},
		{"a C1 character", root.key("\u0085"), "\u0085"},
		{"a backslash", root.key("a").key(`b\`), `a.b\`},
		{"a printable rune", root.key("caf\u00e9"), "caf\u00e9"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.at.String(); got != testCase.want {
				t.Fatalf("String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFieldPathInterns(t *testing.T) {
	t.Parallel()

	root := newRoot()
	first := root.key("a").item(1)
	if root.key("a").item(1) != first {
		t.Fatal("the same segments gave two nodes")
	}
	if root.key("[1]") == root.item(1) {
		t.Fatal("a key spelled like an index shares its node")
	}
}

func parseWork(t *testing.T, content string) (walked, problems, nodes int) {
	t.Helper()
	parsed, err := Decode(Name, []byte(content))
	if err != nil {
		t.Fatalf("decode() = %v", err)
	}
	if parsed.file != nil {
		parsed.file.check(testRules(), parsed.top, parsed.problems)
	}
	root, err := readDocument(Name, []byte(content))
	if err != nil {
		t.Fatalf("readDocument() = %v", err)
	}
	return parsed.problems.walked, parsed.problems.total, distinctNodes(root)
}

func overridesOf(count int, entry func() string) string {
	return "version: 1\noverrides:\n" + strings.Repeat("  - "+entry()+"\n", count)
}

type linearCase struct {
	name    string
	base    int
	content func(items int) string
}

func TestParseStaysLinearInProblems(t *testing.T) {
	t.Parallel()

	const stepsPerNode = 4
	for _, testCase := range linearCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			for _, items := range []int{testCase.base, 4 * testCase.base} {
				walked, problems, nodes := parseWork(t, testCase.content(items))
				if work := walked + problems; work > stepsPerNode*nodes {
					t.Errorf("%d items: %d walked steps for %d yaml nodes, want a linear walk",
						items, work, nodes)
				}
				if walked < problems {
					t.Errorf("%d items: %d walked steps for %d problems, want the walk counted",
						items, walked, problems)
				}
			}
		})
	}
}

func linearCases() []linearCase {
	return []linearCase{
		{"keys in one mapping", 64, func(items int) string {
			keys := make([]string, items)
			for idx := range keys {
				keys[idx] = fmt.Sprintf("x%d: {}", idx)
			}
			return overridesOf(50, func() string {
				return "analyzers: {" + strings.Join(keys, ", ") + "}"
			})
		}},
		{"unknown params keys", 64, func(items int) string {
			keys := make([]string, items)
			for idx := range keys {
				keys[idx] = fmt.Sprintf("k%d: 1", idx)
			}
			head := "version: 1\nanalyzers:\n  file-size:\n    params: {"
			return head + strings.Join(keys, ", ") + "}\n"
		}},
	}
}

func pathKeyCases() []parseCase {
	const head = ".pulsarules.yaml: analyzers.file-size.params."
	params := func(body string) string {
		return "version: 1\nanalyzers:\n  file-size:\n    params: {" + body + "}\n"
	}
	return []parseCase{
		{
			"a params key made of brackets",
			params(`"[[[": 1`),
			[]string{head + "[[[: unknown param"},
		},
		{
			"a dotted params key",
			params(`"a.b": 1`),
			[]string{head + "a.b: unknown param"},
		},
		{
			"a dotted key beside a skipped parent",
			"version: 1\nanalyzers:\n  file-size: [x]\n  a.b: {}\n  file-size.params: {}\n",
			[]string{
				".pulsarules.yaml: line 3: analyzers.file-size: want a mapping",
				".pulsarules.yaml: analyzers.a.b: unknown analyzer",
				".pulsarules.yaml: analyzers.file-size.params: unknown analyzer",
			},
		},
		{
			"both odd keys beside each other",
			params(`"[": 1, "a.b": 2, "a[0]": 3`),
			[]string{
				head + "[: unknown param",
				head + "a.b: unknown param",
				head + "a[0]: unknown param",
			},
		},
	}
}

func TestValidateGlobStars(t *testing.T) {
	t.Parallel()

	const tooMany = "glob has more than 4 * (** and lone * segments are free)"
	stars := func(count int) string { return strings.Repeat("*a", count) }
	runValidateCases(t, []validateCase{
		{"four stars in one segment", "version: 1\ntests: ['" + stars(4) + "']\n", ""},
		{
			"five stars in one segment",
			"version: 1\ntests: ['" + stars(5) + "']\n",
			"tests[0]: " + tooMany,
		},
		{
			"stars spread over segments",
			"version: 1\ntests: ['*a/*a/*a/*a/*a']\n",
			"tests[0]: " + tooMany,
		},
		{"double-star segments are free", "version: 1\ntests: ['**/**/**/**/**/**/*a']\n", ""},
		{"lone star segments are free", "version: 1\ntests: ['*/*/*/*/*.dart']\n", ""},
		{"many lone star segments", "version: 1\ntests: ['*/*/*/*/*/*/*/*/*/*/*a']\n", ""},
		{
			"lone star segments beside four counted stars",
			"version: 1\ntests: ['*/" + stars(4) + "/*']\n",
			"",
		},
		{
			"a counted star beside lone star segments",
			"version: 1\ntests: ['*/*/" + stars(5) + "/*']\n",
			"tests[0]: " + tooMany,
		},
		{
			"a segment of two stars is not lone",
			"version: 1\ntests: ['*a/**b/*c/*d/*e']\n",
			"tests[0]: " + tooMany,
		},
		{
			"an override glob",
			"version: 1\noverrides: [{files: ['a', '" + stars(5) + "']}]\n",
			"overrides[0].files[1]: " + tooMany,
		},
		{
			"a star run counts each star",
			"version: 1\ntests: ['*****']\n",
			"tests[0]: " + tooMany,
		},
	})
}

func TestValidateGlobBraces(t *testing.T) {
	t.Parallel()

	const refused = "brace alternatives are not supported; list each glob"
	runValidateCases(t, []validateCase{
		{
			"alternation in tests",
			"version: 1\ntests: ['web/src/**/*.{ts,tsx}']\n",
			"tests[0]: " + refused,
		},
		{
			"an opening brace in an override",
			"version: 1\noverrides: [{files: ['a', 'b{']}]\n",
			"overrides[0].files[1]: " + refused,
		},
		{
			"a closing brace in an override",
			"version: 1\noverrides: [{files: ['}b']}]\n",
			"overrides[0].files[0]: " + refused,
		},
		{"a brace inside a class is a literal", "version: 1\ntests: ['lib/[{]x.go']\n", ""},
		{"a closing brace inside a class is a literal", "version: 1\ntests: ['lib/[}]x']\n", ""},
		{
			"a brace after a class is refused",
			"version: 1\ntests: ['lib/[a]{x,y}']\n",
			"tests[0]: " + refused,
		},
		{
			"an escaped bracket does not open a class",
			"version: 1\ntests: ['lib/\\[{x']\n",
			"tests[0]: " + refused,
		},
		{
			"the brace message wins over the star cap",
			"version: 1\ntests: ['{a,b}/*a*a*a*a*a']\n",
			"tests[0]: " + refused,
		},
		{"an escaped opening brace is a literal", `version: 1` + "\ntests: ['a\\{b']\n", ""},
		{"an escaped closing brace is a literal", `version: 1` + "\ntests: ['a\\}b']\n", ""},
		{
			"an escaped backslash does not escape the brace",
			"version: 1\ntests: ['a\\\\{b']\n",
			"tests[0]: " + refused,
		},
	})
}
