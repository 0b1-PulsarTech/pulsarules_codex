package docshape

import (
	"go/ast"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type classifyCase struct {
	name     string
	dialect  lexer.Dialect
	own      []string
	adjacent []string
	text     []string
	want     Shape
}

func goText(testCase classifyCase) []string {
	if testCase.text != nil || testCase.dialect != lexer.DialectGo {
		return testCase.text
	}
	group := &ast.CommentGroup{}
	for _, line := range testCase.own {
		group.List = append(group.List, &ast.Comment{Text: line})
	}
	return strings.Split(strings.TrimSpace(group.Text()), "\n")
}

func TestClassify(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(
		directiveCases(), deprecatedAndMarkerCases(), dialectDifferenceCases(),
		blockAndMarkerCases(),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Classify(testCase.dialect, testCase.own, testCase.adjacent, goText(testCase))
			if got.Exempt != testCase.want.Exempt || got.Mixed != testCase.want.Mixed ||
				!slices.Equal(got.Prose, testCase.want.Prose) {
				t.Fatalf("Classify = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func directiveCases() []classifyCase {
	goDialect, tsDialect := lexer.DialectGo, lexer.DialectTS
	return []classifyCase{
		{
			name:    "go prose",
			dialect: goDialect,
			own:     []string{"// Does it."},
			want:    Shape{Prose: []string{"Does it."}},
		},
		{
			name:    "ts prose",
			dialect: tsDialect,
			own:     []string{"/** Does it. */"},
			want:    Shape{Prose: []string{"Does it."}},
		},
		{name: "go empty", dialect: goDialect, own: []string{"//"}, want: Shape{Exempt: true}},
		{name: "ts empty", dialect: tsDialect, own: []string{"/** */"}, want: Shape{Exempt: true}},
		{
			name: "go directive only", dialect: goDialect, own: []string{"//nolint:foo"},
			want: Shape{Exempt: true},
		},
		{
			name: "ts directive only", dialect: tsDialect, own: []string{"/** eslint-disable */"},
			want: Shape{Exempt: true},
		},
		{
			name: "go directive beside prose", dialect: goDialect,
			own:  []string{"// Prose.", "//go:noinline"},
			want: Shape{Prose: []string{"Prose."}, Mixed: true},
		},
		{
			name: "ts directive beside prose", dialect: tsDialect,
			own:  []string{"/**", " * Prose.", " * @ts-ignore", " */"},
			want: Shape{Prose: []string{"Prose."}, Mixed: true},
		},
		{
			name: "adjacent directive beside prose", dialect: tsDialect,
			own:      []string{"/** Prose. */"},
			adjacent: []string{"// eslint-disable-next-line"},
			want:     Shape{Prose: []string{"Prose."}, Mixed: true},
		},
		{
			name: "adjacent plain comment", dialect: tsDialect,
			own:      []string{"/** Prose. */"},
			adjacent: []string{"// note"},
			want:     Shape{Prose: []string{"Prose."}},
		},
		{
			name: "adjacent directive on an empty doc", dialect: tsDialect,
			own:      []string{"/** */"},
			adjacent: []string{"// eslint-disable"},
			want:     Shape{Exempt: true},
		},
	}
}

func deprecatedAndMarkerCases() []classifyCase {
	goDialect, tsDialect := lexer.DialectGo, lexer.DialectTS
	return []classifyCase{
		{
			name: "go deprecated", dialect: goDialect, own: []string{"// Deprecated: use x."},
			want: Shape{Prose: []string{"Deprecated: use x."}, Exempt: true},
		},
		{
			name: "ts deprecated", dialect: tsDialect, own: []string{"/** @deprecated use x */"},
			want: Shape{Prose: []string{"@deprecated use x"}, Exempt: true},
		},
		{
			name: "deprecated at the line limit", dialect: goDialect,
			own:  []string{"// Deprecated: a", "// b", "// c"},
			want: Shape{Prose: []string{"Deprecated: a", "b", "c"}, Exempt: true},
		},
		{
			name: "deprecated over the line limit", dialect: goDialect,
			own:  []string{"// Deprecated: a", "// b", "// c", "// d"},
			want: Shape{Prose: []string{"Deprecated: a", "b", "c", "d"}},
		},
		{
			name: "deprecated with a blank inside", dialect: goDialect,
			own:  []string{"// Deprecated: a", "//", "// b"},
			want: Shape{Prose: []string{"Deprecated: a", "b"}},
		},
		{
			name: "deprecated after prose", dialect: goDialect,
			own:  []string{"// Does it.", "// Deprecated: a"},
			want: Shape{Prose: []string{"Does it.", "Deprecated: a"}},
		},
		{
			name: "deprecated ignores blank edges", dialect: tsDialect,
			own:  []string{"/**", " *", " * @deprecated x", " *", " */"},
			want: Shape{Prose: []string{"@deprecated x"}, Exempt: true},
		},
		{
			name: "go simplification", dialect: goDialect, own: []string{"// simplification: x"},
			want: Shape{Prose: []string{"simplification: x"}, Exempt: true},
		},
		{
			name: "ts simplification", dialect: tsDialect,
			own:  []string{"/**", " * simplification: x", " */"},
			want: Shape{Prose: []string{"simplification: x"}, Exempt: true},
		},
		{
			name: "simplification after prose", dialect: goDialect,
			own:  []string{"// Does it.", "// simplification: later"},
			want: Shape{Prose: []string{"Does it.", "simplification: later"}},
		},
		{
			name: "wrong case simplification", dialect: goDialect,
			own:  []string{"// Simplification: x"},
			want: Shape{Prose: []string{"Simplification: x"}},
		},
		{name: "no lines", dialect: goDialect, want: Shape{Exempt: true}},
	}
}

func dialectDifferenceCases() []classifyCase {
	goDialect, tsDialect := lexer.DialectGo, lexer.DialectTS
	return []classifyCase{
		{
			name:    "go at-deprecated is prose",
			dialect: goDialect,
			own: []string{
				"// @deprecated use x",
			},
			want: Shape{Prose: []string{"@deprecated use x"}},
		},
		{
			name: "go stars line is prose", dialect: goDialect,
			own: []string{"// ***"}, want: Shape{Prose: []string{"***"}},
		},
		{
			name: "go empty star block is prose", dialect: goDialect,
			own: []string{"/** */"}, want: Shape{Prose: []string{"*"}},
		},
		{
			name: "go closer-only comments stay blank", dialect: goDialect,
			own: []string{"//*/", "// */"}, want: Shape{Exempt: true},
		},
		{
			name: "go ignore line is prose", dialect: goDialect,
			own: []string{"// ignore: unused"}, want: Shape{Prose: []string{"ignore: unused"}},
		},
		{
			name: "go eslint line is prose", dialect: goDialect,
			own: []string{"// eslint-foo"}, want: Shape{Prose: []string{"eslint-foo"}},
		},
		{
			name: "go export is a directive", dialect: goDialect,
			own: []string{"//export Foo"}, want: Shape{Exempt: true},
		},
		{
			name:    "go directive beside deprecated does not count toward the cap",
			dialect: goDialect,
			own:     []string{"// Deprecated: a", "// b", "// c", "//go:noinline"},
			want:    Shape{Exempt: true, Mixed: true, Prose: []string{"Deprecated: a", "b", "c"}},
		},
		{
			name: "ts directive beside deprecated counts toward the cap", dialect: tsDialect,
			own:  []string{"/**", " * @deprecated a", " * b", " * c", " * @ts-ignore", " */"},
			want: Shape{Prose: []string{"@deprecated a", "b", "c"}, Mixed: true},
		},
		{
			name:    "go directive before simplification",
			dialect: goDialect,
			own: []string{
				"//go:noinline",
				"// simplification: x",
			},
			want: Shape{Exempt: true, Mixed: true, Prose: []string{"simplification: x"}},
		},
		{
			name:    "go blank before simplification",
			dialect: goDialect,
			own: []string{
				"//",
				"// simplification: x",
			},
			want: Shape{Prose: []string{"simplification: x"}, Exempt: true},
		},
		{
			name: "ts go directive is prose", dialect: tsDialect,
			own: []string{"/** //go:noinline */"}, want: Shape{Prose: []string{"//go:noinline"}},
		},
		{
			name:    "ts adjacent banner is not a directive",
			dialect: tsDialect,
			own: []string{
				"/** Prose. */",
			},
			adjacent: []string{"// Code generated by x. DO NOT EDIT."},
			want:     Shape{Prose: []string{"Prose."}},
		},
	}
}

func blockAndMarkerCases() []classifyCase {
	goDialect, tsDialect := lexer.DialectGo, lexer.DialectTS
	return []classifyCase{
		{
			name: "ts stars line is blank", dialect: tsDialect,
			own: []string{"/** *** */"}, want: Shape{Exempt: true},
		},
		{
			name: "ts deprecated marker", dialect: tsDialect,
			own:  []string{"/** Deprecated: use x. */"},
			want: Shape{Exempt: true, Prose: []string{"Deprecated: use x."}},
		},
		{
			name: "go block interior directive is text", dialect: goDialect,
			own:  []string{"/*", "real prose.", "//go:noinline", "*/"},
			text: []string{"real prose.", "//go:noinline"},
			want: Shape{Prose: []string{"real prose.", "//go:noinline"}},
		},
		{
			name: "go block interior slash is text", dialect: goDialect,
			own:  []string{"/*", "Deprecated: a", "//", "more.", "*/"},
			text: []string{"Deprecated: a", "//", "more."},
			want: Shape{Exempt: true, Prose: []string{"Deprecated: a", "//", "more."}},
		},
		{
			name: "unknown dialect gets no rules", dialect: lexer.Dialect("zz"),
			own:  []string{"// Deprecated: a"},
			want: Shape{Prose: []string{"// Deprecated: a"}},
		},
	}
}

func TestClassify_Prose(t *testing.T) {
	t.Parallel()

	own := []string{"/**", " * Does it.", " * @ts-ignore", " *", " * More.", " */"}
	want := []string{"Does it.", "More."}
	if got := Classify(lexer.DialectTS, own, nil, nil).Prose; !slices.Equal(got, want) {
		t.Fatalf("Prose = %q, want %q", got, want)
	}
}
