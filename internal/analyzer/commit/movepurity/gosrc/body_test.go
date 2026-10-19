package gosrc

import (
	"go/token"
	"slices"
	"testing"
)

func TestIsQualified(t *testing.T) {
	t.Parallel()

	ident := func(lit string) goToken { return goToken{kind: token.IDENT, lit: lit} }
	period := goToken{kind: token.PERIOD}
	testCases := []struct {
		name string
		body []goToken
		pos  int
		want bool
	}{
		{"qualified", []goToken{ident("q"), period, ident("X")}, 0, true},
		{"other qualifier", []goToken{ident("r"), period, ident("X")}, 0, false},
		{"truncated", []goToken{ident("q"), period}, 0, false},
		{
			"chained selector",
			[]goToken{ident("a"), period, ident("q"), period, ident("X")},
			2,
			false,
		},
		{
			"not followed by an ident",
			[]goToken{ident("q"), period, {kind: token.LPAREN}},
			0,
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isQualified(testCase.body, testCase.pos, "q"); got != testCase.want {
				t.Fatalf("isQualified() = %v, want %v", got, testCase.want)
			}
		})
	}
}

type bodyCase struct {
	name   string
	oldSrc string
	newSrc string
	rules  []QualRule
	bind   Binding
	want   bool
}

func bodyRow(name, oldBody, newBody string) bodyCase {
	return bodyCase{name: name, oldSrc: oldBody, newSrc: newBody}
}

func (row bodyCase) equivalent() bodyCase {
	row.want = true
	return row
}

func (row bodyCase) withRules(rules ...QualRule) bodyCase {
	row.rules = rules
	return row
}

func (row bodyCase) withBind(bind Binding) bodyCase {
	row.bind = bind
	return row
}

func runBodyCases(t *testing.T, testCases []bodyCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			oldFile := mustParse(t, "package p\n\n"+testCase.oldSrc)
			newFile := mustParse(t, "package p\n\n"+testCase.newSrc)
			got := BodiesEquivalent(oldFile, newFile, testCase.rules, testCase.bind)
			if got != testCase.want {
				t.Fatalf("BodiesEquivalent() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func ret(call string) string { return "func F() int { return " + call + " }\n" }

func TestBodiesEquivalent_Tokens(t *testing.T) {
	t.Parallel()

	runBodyCases(t, []bodyCase{
		bodyRow("identical", ret("1"), ret("1")).equivalent(),
		bodyRow("reflowed", ret("1"), "func F() int {\n\treturn 1\n}\n").equivalent(),
		bodyRow("another literal", ret("1"), ret("2")),
		bodyRow("a longer new body", "func F() {}\n", "func F() {}\n\nfunc G() {}\n"),
		bodyRow("a longer old body", "func F() {}\n\nfunc G() {}\n", "func F() {}\n"),
	})
}

func TestBodiesEquivalent_Rules(t *testing.T) {
	t.Parallel()

	always := func(string) bool { return true }
	never := func(string) bool { return false }
	dropQ := QualRule{OldQual: "q", Declared: always}
	runBodyCases(t, []bodyCase{
		bodyRow("qualifier dropped", ret("q.Foo()"), ret("Foo()")).equivalent().withRules(dropQ),
		bodyRow("qualifier added", ret("Foo()"), ret("q.Foo()")).equivalent().
			withRules(QualRule{NewQual: "q", Declared: always}),
		bodyRow("qualifier swapped", ret("q.Foo()"), ret("r.Foo()")).equivalent().
			withRules(QualRule{OldQual: "q", NewQual: "r", Declared: always}),
		bodyRow("an undeclared name", ret("q.Foo()"), ret("Foo()")).
			withRules(QualRule{OldQual: "q", Declared: never}),
		bodyRow("an unexported name", ret("q.foo()"), ret("foo()")).withRules(dropQ),
		bodyRow("a different name", ret("q.Foo()"), ret("Bar()")).withRules(dropQ),
	})
}

func TestBodiesEquivalent_Binding(t *testing.T) {
	t.Parallel()

	rebound := Binding{
		OldPaths: map[string]string{"q": "m/a"},
		NewPaths: map[string]string{"q": "m/b"},
		Moves: []MoveInfo{
			{OldFull: "m/a", NewFull: "m/b", Decls: map[string]bool{"Foo": true}},
		},
	}
	taken := func(name string) Binding {
		return Binding{Captures: CaptureSets{Taken: map[string]bool{name: true}}}
	}
	runBodyCases(t, []bodyCase{
		bodyRow("rebound to the move target", ret("q.Foo()"), ret("q.Foo()")).equivalent().
			withBind(rebound),
		bodyRow("rebound, name not moved", ret("q.Bar()"), ret("q.Bar()")).
			withBind(rebound),
		bodyRow("a qualifier of unknown origin", ret("q.Foo()"), ret("q.Foo()")).
			withBind(Binding{IsUnknown: true}),
		bodyRow("a bare name another file owns", ret("Foo()"), ret("Foo()")).
			withBind(taken("Foo")),
		bodyRow("a bare name nobody owns", ret("Foo()"), ret("Foo()")).equivalent().
			withBind(taken("Bar")),
	})
}

func TestSameComment(t *testing.T) {
	t.Parallel()

	moves := []MoveInfo{{OldPkg: "core", NewPkg: "sources", Decls: map[string]bool{"Skip": true}}}
	comment := func(lit string) goToken { return goToken{kind: token.COMMENT, lit: lit} }
	testCases := []struct {
		name  string
		left  goToken
		right goToken
		want  bool
	}{
		{
			"qualifier follows the move",
			comment("// see core.Skip"),
			comment("// see sources.Skip"),
			true,
		},
		{"name not moved", comment("// see core.Other"), comment("// see sources.Other"), false},
		{
			"directive",
			comment("//go:generate core.Skip"),
			comment("//go:generate sources.Skip"),
			false,
		},
		{"line", comment("//line core.Skip"), comment("//line sources.Skip"), false},
		{"block line", comment("/*line core.Skip */"), comment("/*line sources.Skip */"), false},
		{"export", comment("//export core.Skip"), comment("//export sources.Skip"), false},
		{"extern", comment("//extern core.Skip"), comment("//extern sources.Skip"), false},
		{"digit before qualifier", comment("// 1core.Skip"), comment("// 1sources.Skip"), false},
		{"longer qualifier", comment("// xcore.Skip"), comment("// xsources.Skip"), false},
		{"longer name", comment("// core.SkipX"), comment("// sources.SkipX"), false},
		{"prose differs", comment("// see core.Skip"), comment("// use sources.Skip"), false},
		{"not a comment", goToken{kind: token.IDENT, lit: "core"}, comment("// sources"), false},
		{"want", comment("// want `core.Skip`"), comment("// want `sources.Skip`"), false},
		{"want tight", comment("//want \"core.Skip\""), comment("//want \"sources.Skip\""), false},
		{"want block", comment("/* want core.Skip */"), comment("/* want sources.Skip */"), false},
		{"want after tab", comment("//\twant core.Skip"), comment("//\twant sources.Skip"), false},
		{
			"want on a new line",
			comment("/*\nwant core.Skip\n*/"),
			comment("/*\nwant sources.Skip\n*/"),
			false,
		},
		{"error", comment("// ERROR \"core.Skip\""), comment("// ERROR \"sources.Skip\""), false},
		{"gc error", comment("// GC_ERROR core.Skip"), comment("// GC_ERROR sources.Skip"), false},
		{
			"error late",
			comment("// x: ERROR core.Skip"),
			comment("// x: ERROR sources.Skip"),
			false,
		},
		{"wanted prose", comment("// wanted core.Skip"), comment("// wanted sources.Skip"), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			oldFile := File{body: []goToken{testCase.left}}
			newFile := File{body: []goToken{testCase.right}}
			if got := newCommentMatcher().same(oldFile, newFile, [2]int{}, moves); got != testCase.want {
				t.Fatalf("commentMatcher.same() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSameComment_Example(t *testing.T) {
	t.Parallel()

	moves := []MoveInfo{{OldPkg: "core", NewPkg: "sources", Decls: map[string]bool{"Skip": true}}}
	example := func(name, last string) string {
		return "package p\n\nfunc " + name + "() {\n\t// Output:\n\t// " + last + "\n}\n"
	}
	testCases := []struct {
		name   string
		fn     string
		oldRef string
		newRef string
		want   bool
	}{
		{"example body", "ExampleSkip", "core.Skip", "sources.Skip", false},
		{"suffixed example", "ExampleSkip_second", "core.Skip", "sources.Skip", false},
		{"bare Example", "Example", "core.Skip", "sources.Skip", false},
		{"lowercase suffix", "Examplefoo", "core.Skip", "sources.Skip", true},
		{"ordinary function", "Helper", "core.Skip", "sources.Skip", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			oldFile := mustParse(t, example(testCase.fn, testCase.oldRef))
			newFile := mustParse(t, example(testCase.fn, testCase.newRef))
			at := [2]int{len(oldFile.body) - 3, len(newFile.body) - 3}
			if got := newCommentMatcher().same(oldFile, newFile, at, moves); got != testCase.want {
				t.Fatalf("commentMatcher.same() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSameComment_References(t *testing.T) {
	t.Parallel()

	declNew := map[string]bool{"New": true}
	toXa := MoveInfo{OldPkg: "util", NewPkg: "xa", Decls: declNew}
	toYb := MoveInfo{OldPkg: "util", NewPkg: "yb", Decls: declNew}
	toThings := MoveInfo{OldPkg: "other", NewPkg: "things", Decls: declNew}
	both := []MoveInfo{toXa, toYb}
	mixed := []MoveInfo{toXa, toThings}
	testCases := []struct {
		name  string
		moves []MoveInfo
		left  string
		right string
		want  bool
	}{
		{"first of two moves", both, "// see util.New", "// see xa.New", true},
		{"second of two moves", both, "// see util.New", "// see yb.New", true},
		{"target no move maps to", both, "// see util.New", "// see zz.New", false},
		{"each follows its move", mixed, "// util.New other.New", "// xa.New things.New", true},
		{"references cross-wired", mixed, "// util.New other.New", "// things.New xa.New", false},
		{"reference count differs", both, "// util.New", "// xa.New xa.New", false},
		{"name differs", both, "// util.New", "// xa.Make", false},
	}
	for _, testCase := range testCases {
		for _, reversed := range []bool{false, true} {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				moves := slices.Clone(testCase.moves)
				if reversed {
					slices.Reverse(moves)
				}
				oldFile := File{body: []goToken{{kind: token.COMMENT, lit: testCase.left}}}
				newFile := File{body: []goToken{{kind: token.COMMENT, lit: testCase.right}}}
				if got := newCommentMatcher().same(oldFile, newFile, [2]int{}, moves); got != testCase.want {
					t.Fatalf("commentMatcher.same() = %v, want %v", got, testCase.want)
				}
			})
		}
	}
}
