package strictyaml

import (
	"testing"
)

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
