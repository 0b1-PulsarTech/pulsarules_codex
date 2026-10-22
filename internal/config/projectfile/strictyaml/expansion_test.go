package strictyaml

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

func mergeBomb(levels int) string {
	var content strings.Builder
	content.WriteString("version: 1\nanalyzers:\n  a0: &a0 {enabled: true}\n")
	for level := 1; level <= levels; level++ {
		refs := strings.TrimSuffix(strings.Repeat(fmt.Sprintf("*a%d, ", level-1), 10), ", ")
		fmt.Fprintf(&content, "  a%d: &a%d {<<: [%s]}\n", level, level, refs)
	}
	return content.String()
}

func memoChain(levels int) string {
	var content strings.Builder
	content.WriteString("version: 1\nskip_dirs: &B [" + strings.Repeat("x,", 1000) + "]\n")
	content.WriteString("analyzers:\n  file-size:\n    params:\n      deep: ")
	for range levels {
		content.WriteString("{b: [*B, *B, *B], n: ")
	}
	content.WriteString("{}" + strings.Repeat("}", levels) + "\n")
	return content.String()
}

func manyAnchors(count, items int) string {
	var content strings.Builder
	content.WriteString("version: 1\nanalyzers:\n  file-size:\n    params:\n")
	for idx := range count {
		fmt.Fprintf(&content, "      k%d: &a%d [%s]\n", idx, idx, strings.Repeat("v,", items))
	}
	content.WriteString("      refs: [")
	for idx := range count {
		fmt.Fprintf(&content, "*a%d,", idx)
	}
	content.WriteString("]\n")
	return content.String()
}

type expansionCase struct {
	name    string
	content string
	want    string
}

func expansionCases() []expansionCase {
	const head = ".pulsarules.yaml: line "
	const volume = ": document expands past 100000 nodes"
	const loop = ": alias refers to its own anchor"
	const refsPath = "analyzers.file-size.params.refs"
	deepPath := "analyzers.file-size.params.deep" + strings.Repeat(".n", 7)
	rootAlias := "&r {version: 1, analyzers: {file-size: {params: {x: *r}}}}\n"
	selfMerge := "version: 1\nanalyzers: &a {file-size: {<<: *a}}\n"
	return []expansionCase{
		{"shared merge chain within budget", mergeBomb(3), ""},
		{"seven level merge bomb", mergeBomb(7), head + "8: analyzers.a5.<<[3]" + volume},
		{"eight level merge bomb", mergeBomb(8), head + "8: analyzers.a5.<<[3]" + volume},
		{"two hundred levels", mergeBomb(200), head + "8: analyzers.a5.<<[3]" + volume},
		{"shared list reused down a deep chain", memoChain(40), head + "6: " + deepPath + volume},
		{"anchors then aliases", manyAnchors(100, 600), head + "105: " + refsPath + volume},
		{
			"alias inside its own anchor",
			"version: 1\nanalyzers: &loop\n  x: *loop\n",
			head + "3: analyzers.x" + loop,
		},
		{"root alias in a mapping", rootAlias, head + "1: analyzers.file-size.params.x" + loop},
		{
			"root alias as a merge",
			"&r {version: 1, analyzers: {<<: *r}}\n",
			head + "1: analyzers.<<" + loop,
		},
		{"list holding itself", "version: 1\ntests: &t [a, *t]\n", head + "2: tests[1]" + loop},
		{"mapping merging itself", selfMerge, head + "2: analyzers.file-size.<<" + loop},
		{
			"alias list that fans out",
			"version: 1\ntests: &t [" + strings.Repeat("a, ", 400) + "]\n" +
				"skip_dirs: [" + strings.Repeat("*t, ", 400) + "]\n",
			head + "3: skip_dirs[249]" + volume,
		},
	}
}

func paramsYAML(body string) string {
	return "version: 1\nanalyzers:\n  file-size:\n    params:\n" + body + "\n"
}

func nestedValue(levels int, key string) string {
	return "      max_lines: " + strings.Repeat("{"+key+": ", levels) + "1" +
		strings.Repeat("}", levels)
}

func aliasChain(last int) string {
	var content strings.Builder
	content.WriteString("      k0: &a0 {x: 1}\n")
	for idx := 1; idx <= last; idx++ {
		fmt.Fprintf(&content, "      k%d: &a%d {x: *a%d}\n", idx, idx, idx-1)
	}
	return paramsYAML(strings.TrimSuffix(content.String(), "\n"))
}

func sizeCases() []expansionCase {
	const head = ".pulsarules.yaml: line "
	const tooLong = ": value longer than 1024 bytes"
	const tooBig = ".pulsarules.yaml: config is larger than 256 KiB"
	atLimit := "version: 1\n#" + strings.Repeat("x", MaxFileBytes-13) + "\n"
	value := func(size int) string {
		return paramsYAML(`      max_lines: "` + strings.Repeat("x", size) + `"`)
	}
	key := strings.Repeat("K", 1025)
	return []expansionCase{
		{"a file exactly at the size limit", atLimit, ""},
		{"a file one byte over the size limit", atLimit + "#", tooBig},
		{"a scalar of exactly 1024 bytes", value(1024), ""},
		{
			"a value one byte over the scalar limit",
			value(1025),
			head + "5: analyzers.file-size.params.max_lines" + tooLong,
		},
		{
			"a key of exactly 1024 bytes",
			"version: 1\nanalyzers:\n  " + key[:1024] + ": {}\n",
			"",
		},
		{
			"a key one byte over the scalar limit",
			"version: 1\nanalyzers:\n  ? " + key + "\n  : {}\n",
			head + "3: ..." + strings.Repeat("K", 80) + tooLong,
		},
	}
}

func depthCases() []expansionCase {
	const head = ".pulsarules.yaml: line "
	const tooDeep = ": nested deeper than 64 levels"
	const tooBig = ".pulsarules.yaml: config is larger than 256 KiB"
	deepTail := "..." + strings.Repeat(".a", 40)
	return []expansionCase{
		{
			"a hundred byte key nested nine thousand deep",
			paramsYAML(nestedValue(9000, strings.Repeat("k", 100))),
			tooBig,
		},
		{"nesting at exactly 64 levels", paramsYAML(nestedValue(60, "a")), ""},
		{
			"nesting one level over the limit",
			paramsYAML(nestedValue(61, "a")),
			head + "5: " + deepTail + tooDeep,
		},
		{
			"nine thousand levels of one byte keys",
			paramsYAML(nestedValue(9000, "a")),
			head + "5: " + deepTail + tooDeep,
		},
		{"an alias chain at the depth limit", aliasChain(59), ""},
		{
			"an alias chain one level over the limit",
			aliasChain(60),
			head + "65: analyzers.file-size.params.k60.x" + tooDeep,
		},
	}
}

func flowMap(count int, key func(idx int) string) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = key(idx) + ": 1"
	}
	return paramsYAML("      max_lines: 1\n      extra: {" + strings.Join(items, ", ") + "}")
}

func breadthCases() []expansionCase {
	const head = ".pulsarules.yaml: line "
	const extra = "analyzers.file-size.params.extra"
	const tooMany = head + "6: " + extra + ": more than 256 keys"
	distinct := func(idx int) string { return fmt.Sprintf("k%d", idx) }
	sameKey := func(int) string { return "k" }
	list := func(count int) string {
		return "version: 1\ntests: [" + strings.Repeat("t,", count) + "]\n"
	}
	return []expansionCase{
		{"a mapping of exactly 256 keys", flowMap(256, distinct), ""},
		{"a mapping of 257 keys", flowMap(257, distinct), tooMany},
		{"a mapping of 20000 distinct keys", flowMap(20000, distinct), tooMany},
		{"a sequence of exactly 1024 entries", list(1024), ""},
		{"a sequence of 1025 entries", list(1025), head + "2: tests: more than 1024 entries"},
		{"a sequence of 100000 entries", list(100_000), head + "2: tests: more than 1024 entries"},
		{
			"thirty thousand copies of one key",
			flowMap(30000, sameKey),
			tooMany,
		},
	}
}

func duplicateCases() []expansionCase {
	const head = ".pulsarules.yaml: line "
	const extra = "analyzers.file-size.params.extra"
	distinct := func(idx int) string { return fmt.Sprintf("k%d", idx) }
	sameKey := func(int) string { return "k" }
	return []expansionCase{
		{
			"one key twice in a params mapping",
			flowMap(2, sameKey),
			head + "6: " + extra + ".k: duplicate key, first at line 6",
		},
		{
			"a key repeated inside the limit",
			flowMap(256, func(idx int) string { return distinct(idx % 255) }),
			head + "6: " + extra + ".k0: duplicate key, first at line 6",
		},
		{
			"a key spelled as a string and as an int",
			"version: 1\nanalyzers:\n  file-size:\n    params:\n      \"1\": 1\n      1: 2\n",
			head + "6: analyzers.file-size.params.1: duplicate key, first at line 5",
		},
		{
			"two merge keys in one mapping",
			"version: 1\nanalyzers:\n  a: &a {}\n  b: {<<: *a, <<: *a}\n",
			head + "4: analyzers.b.<<: duplicate key, first at line 4",
		},
		{
			"a key twice inside a params value",
			paramsYAML("      max_lines: {a: 1, a: 2}"),
			head + "5: analyzers.file-size.params.max_lines.a: duplicate key, first at line 5",
		},
		{
			"an alias key beside a key spelled like the anchor",
			"version: 1\nskip_dirs: [&a x]\nanalyzers: {a: {}, *a: {}}\n",
			"",
		},
		{
			"the same key twice at the top",
			"version: 1\ntests: []\ntests: []\n",
			head + "3: tests: duplicate key, first at line 2",
		},
	}
}

func distinctNodes(node *yaml.Node) int {
	count := 1
	for _, child := range node.Content {
		count += distinctNodes(child)
	}
	return count
}

func requireOneVisitPerNode(t *testing.T, content string) {
	t.Helper()
	root, err := readDocument(fixtureName, []byte(content))
	if err != nil {
		return
	}
	walker := expander{extents: map[*yaml.Node]extent{}, active: map[*yaml.Node]bool{}}
	walker.size(root, Root(), 0)
	if nodes := distinctNodes(root); walker.visits > nodes {
		t.Errorf("walk visited %d nodes of a %d node document, want each at most once",
			walker.visits, nodes)
	}
}

func TestCheckExpansion(t *testing.T) {
	t.Parallel()

	allCases := slices.Concat(
		expansionCases(), sizeCases(), depthCases(), breadthCases(), duplicateCases(),
	)
	for _, testCase := range allCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFixture([]byte(testCase.content))
			requireOneVisitPerNode(t, testCase.content)
			assertPlainFixture(t, err)
			if testCase.want == "" {
				if err != nil {
					t.Fatalf("decodeFixture() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("decodeFixture() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

type clipPathCase struct {
	name string
	path string
	want string
}

func clipPathCases() []clipPathCase {
	return []clipPathCase{
		{"empty", "", ""},
		{"short", "a.b", "a.b"},
		{"exactly the clip", strings.Repeat("a", 80), strings.Repeat("a", 80)},
		{"one over", "b" + strings.Repeat("a", 80), "..." + strings.Repeat("a", 80)},
		{
			"a rune across the cut",
			strings.Repeat("\u00e9", 60) + "a",
			"..." + strings.Repeat("\u00e9", 39) + "a",
		},
		{
			"a short escape across the cut",
			strings.Repeat("\n", 50) + "a",
			"..." + strings.Repeat(`\n`, 39) + "a",
		},
		{
			"a hex escape across the cut",
			strings.Repeat("\x01", 30) + "ab",
			"..." + strings.Repeat(`\x01`, 19) + "ab",
		},
		{
			"a unicode escape across the cut",
			strings.Repeat("\u0085", 20),
			"..." + strings.Repeat(`\u0085`, 13),
		},
		{
			"a long escape across the cut",
			strings.Repeat("\U000e0001", 8) + "abcde",
			"..." + strings.Repeat(`\U000e0001`, 7) + "abcde",
		},
		{
			"a literal backslash before a rune",
			strings.Repeat("x", 30) + `\` + "\u00e9" + strings.Repeat("a", 78),
			"...\u00e9" + strings.Repeat("a", 78),
		},
		{
			"a literal backslash before a control byte",
			strings.Repeat("x", 30) + `\` + "\x01" + strings.Repeat("a", 76),
			"..." + `\x01` + strings.Repeat("a", 76),
		},
		{
			"literal backslash pairs",
			"zzzzz/" + strings.Repeat(`\x`, 300),
			"..." + strings.Repeat(`\x`, 40),
		},
	}
}

func TestClipPath(t *testing.T) {
	t.Parallel()

	testCases := clipPathCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := clipPath(testCase.path)
			if got != testCase.want || !utf8.ValidString(got) {
				t.Fatalf("clipPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}
