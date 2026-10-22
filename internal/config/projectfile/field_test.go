package projectfile

import (
	"fmt"
	"strings"
	"testing"
)

func TestFieldPathCovered(t *testing.T) {
	t.Parallel()

	root := Root()
	root.Key("skip_dirs").Item(3).mark()
	root.Key("analyzers").Key("x").mark()
	root.Key("a[0]").Key("p.q").mark()
	testCases := []struct {
		name string
		at   *Field
		want bool
	}{
		{"the same field", root.Key("analyzers").Key("x"), true},
		{"a child by key", root.Key("analyzers").Key("x").Key("params"), true},
		{"a child by index", root.Key("skip_dirs").Item(3).Item(0), true},
		{"the list holding a skipped item", root.Key("skip_dirs"), true},
		{"an unrelated field", root.Key("tests"), false},
		{"a sibling item", root.Key("skip_dirs").Item(4), false},
		{"a longer name with the same prefix", root.Key("analyzers").Key("xy"), false},
		{"a key that spells an index", root.Key("skip_dirs[3]"), false},
		{"a key that spells a dotted child", root.Key("analyzers.x"), false},
		{"a key made of brackets", root.Key("[").Key("[[["), false},
		{"an item beside a bracketed key", root.Key("a").Item(0), false},
		{"a dotted key beside its halves", root.Key("a[0]").Key("p"), false},
		{"the dotted key itself", root.Key("a[0]").Key("p.q"), true},
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

	if Root().Key("tests").covered(new(int)) {
		t.Fatal("an unmarked path is covered")
	}
}

func TestFieldPathString(t *testing.T) {
	t.Parallel()

	root := Root()
	testCases := []struct {
		name string
		at   *Field
		want string
	}{
		{"the root", root, ""},
		{"a key", root.Key("tests"), "tests"},
		{"keys", root.Key("analyzers").Key("x").Key("params"), "analyzers.x.params"},
		{"an item", root.Key("tests").Item(2), "tests[2]"},
		{"an item first", root.Item(0).Key("a"), "[0].a"},
		{"an empty key", root.Key("a").Key(""), "a.<empty key>"},
		{"brackets in a key", root.Key("[").Key("a.b"), "[.a.b"},
		{"a control byte", root.Key("a").Key("\x01\x7f"), "a.\x01\x7f"},
		{"a C1 character", root.Key("\u0085"), "\u0085"},
		{"a backslash", root.Key("a").Key(`b\`), `a.b\`},
		{"a printable rune", root.Key("caf\u00e9"), "caf\u00e9"},
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

	root := Root()
	first := root.Key("a").Item(1)
	if root.Key("a").Item(1) != first {
		t.Fatal("the same segments gave two nodes")
	}
	if root.Key("[1]") == root.Item(1) {
		t.Fatal("a key spelled like an index shares its node")
	}
}

func parseWork(t *testing.T, content string) (walked, problems, nodes int) {
	t.Helper()
	parsed, err := decode(Name, []byte(content))
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

func TestClipValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
		want  string
	}{
		{"empty", "", ""},
		{"short", "abc", "abc"},
		{"exactly the clip", strings.Repeat("a", 80), strings.Repeat("a", 80)},
		{"one over", strings.Repeat("a", 81), strings.Repeat("a", 80) + "..."},
		{
			"a rune across the cut",
			strings.Repeat("a", 79) + "\u00e9" + "tail",
			strings.Repeat("a", 79) + "...",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := ClipValue(testCase.value); got != testCase.want {
				t.Fatalf("clipValue() = %q, want %q", got, testCase.want)
			}
		})
	}
}
