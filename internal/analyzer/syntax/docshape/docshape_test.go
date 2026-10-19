package docshape

import (
	"slices"
	"testing"
)

type classifyCase struct {
	name     string
	own      []string
	adjacent []string
	want     Shape
}

func TestClassify(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(directiveCases(), deprecatedAndMarkerCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Classify(testCase.own, testCase.adjacent); got != testCase.want {
				t.Fatalf("Classify = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func directiveCases() []classifyCase {
	return []classifyCase{
		{name: "go prose", own: []string{"// Does it."}, want: Shape{}},
		{name: "ts prose", own: []string{"/** Does it. */"}, want: Shape{}},
		{name: "go empty", own: []string{"//"}, want: Shape{Exempt: true}},
		{name: "ts empty", own: []string{"/** */"}, want: Shape{Exempt: true}},
		{name: "go directive only", own: []string{"//nolint:foo"}, want: Shape{Exempt: true}},
		{
			name: "ts directive only",
			own:  []string{"/** eslint-disable */"},
			want: Shape{Exempt: true},
		},
		{
			name: "go directive beside prose",
			own:  []string{"// Prose.", "//go:noinline"},
			want: Shape{Mixed: true},
		},
		{
			name: "ts directive beside prose",
			own:  []string{"/**", " * Prose.", " * @ts-ignore", " */"},
			want: Shape{Mixed: true},
		},
		{
			name:     "adjacent directive beside prose",
			own:      []string{"/** Prose. */"},
			adjacent: []string{"// eslint-disable-next-line"},
			want:     Shape{Mixed: true},
		},
		{
			name:     "adjacent plain comment",
			own:      []string{"/** Prose. */"},
			adjacent: []string{"// note"},
			want:     Shape{},
		},
		{
			name:     "adjacent directive on an empty doc",
			own:      []string{"/** */"},
			adjacent: []string{"// eslint-disable"},
			want:     Shape{Exempt: true},
		},
	}
}

func deprecatedAndMarkerCases() []classifyCase {
	return []classifyCase{
		{name: "go deprecated", own: []string{"// Deprecated: use x."}, want: Shape{Exempt: true}},
		{
			name: "ts deprecated",
			own:  []string{"/** @deprecated use x */"},
			want: Shape{Exempt: true},
		},
		{
			name: "deprecated at the line limit",
			own:  []string{"// Deprecated: a", "// b", "// c"},
			want: Shape{Exempt: true},
		},
		{
			name: "deprecated over the line limit",
			own:  []string{"// Deprecated: a", "// b", "// c", "// d"},
			want: Shape{},
		},
		{
			name: "deprecated with a blank inside",
			own:  []string{"// Deprecated: a", "//", "// b"},
			want: Shape{},
		},
		{
			name: "deprecated after prose",
			own:  []string{"// Does it.", "// Deprecated: a"},
			want: Shape{},
		},
		{
			name: "deprecated ignores blank edges",
			own:  []string{"/**", " *", " * @deprecated x", " *", " */"},
			want: Shape{Exempt: true},
		},
		{
			name: "go simplification",
			own:  []string{"// simplification: x"},
			want: Shape{Exempt: true},
		},
		{
			name: "ts simplification",
			own:  []string{"/**", " * simplification: x", " */"},
			want: Shape{Exempt: true},
		},
		{
			name: "simplification after prose",
			own:  []string{"// Does it.", "// simplification: later"},
			want: Shape{},
		},
		{name: "wrong case simplification", own: []string{"// Simplification: x"}, want: Shape{}},
		{name: "no lines", want: Shape{Exempt: true}},
	}
}

func TestProse(t *testing.T) {
	t.Parallel()

	own := []string{"/**", " * Does it.", " * @ts-ignore", " *", " * More.", " */"}
	want := []string{"Does it.", "More."}
	if got := Prose(own); !slices.Equal(got, want) {
		t.Fatalf("Prose = %q, want %q", got, want)
	}
}
