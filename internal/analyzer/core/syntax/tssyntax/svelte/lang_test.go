package svelte

import (
	"slices"
	"testing"
)

func TestParseAttrs(t *testing.T) {
	t.Parallel()

	tsLang := attribute{name: "lang", value: "ts"}
	srcA := attribute{name: "src", value: "a"}
	testCases := []struct {
		name  string
		attrs string
		want  []attribute
	}{
		{name: "empty"},
		{name: "bare", attrs: " module", want: []attribute{{name: "module"}}},
		{name: "quoted", attrs: ` lang="ts"`, want: []attribute{tsLang}},
		{name: "single quoted", attrs: ` lang='ts'`, want: []attribute{tsLang}},
		{name: "unquoted", attrs: ` lang=ts`, want: []attribute{tsLang}},
		{name: "spaced equals", attrs: ` lang = "ts" `, want: []attribute{tsLang}},
		{
			name:  "several",
			attrs: ` context="module" lang="ts" defer`,
			want: []attribute{
				{name: "context", value: "module"},
				{name: "lang", value: "ts"},
				{name: "defer"},
			},
		},
		{
			name:  "quoted value with spaces",
			attrs: ` title="a module b" x`,
			want:  []attribute{{name: "title", value: "a module b"}, {name: "x"}},
		},
		{
			name:  "expression value",
			attrs: ` on:click={() => a > b} x`,
			want:  []attribute{{name: "on:click", value: "{() => a > b}"}, {name: "x"}},
		},
		{name: "self closing slash", attrs: ` src=a /`, want: []attribute{srcA}},
		{name: "open quote", attrs: ` lang="ts`, want: []attribute{tsLang}},
		{name: "equals without a value", attrs: ` lang=`, want: []attribute{{name: "lang"}}},
		{name: "stray equals", attrs: ` =a b`, want: []attribute{{name: "a"}, {name: "b"}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := parseAttrs(testCase.attrs); !slices.Equal(got, testCase.want) {
				t.Fatalf("attributes %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestIsTypeScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		attrs string
		want  bool
	}{
		{attrs: ` lang="ts"`, want: true},
		{attrs: ` lang='ts'`, want: true},
		{attrs: ` lang=ts`, want: true},
		{attrs: ` lang = "typescript" `, want: true},
		{attrs: ` type="text/typescript"`, want: true},
		{attrs: ` context="module" lang="ts"`, want: true},
		{attrs: ` lang="js"`},
		{attrs: ` type="module"`},
		{attrs: ` context="module"`},
		{attrs: ` data-lang="ts"`},
		{attrs: ` lang`},
		{attrs: ``},
	}
	for _, testCase := range testCases {
		t.Run(testCase.attrs, func(t *testing.T) {
			t.Parallel()

			if got := isTypeScript(testCase.attrs); got != testCase.want {
				t.Fatalf("typescript %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsJavaScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		attrs string
		want  bool
	}{
		{attrs: ``, want: true},
		{attrs: ` lang="ts"`, want: true},
		{attrs: ` type="module"`, want: true},
		{attrs: ` type="text/javascript"`, want: true},
		{attrs: ` type="application/javascript"`, want: true},
		{attrs: ` type="text/typescript"`, want: true},
		{attrs: ` type="ts"`, want: true},
		{attrs: ` type="text/babel"`, want: true},
		{attrs: ` type=""`, want: true},
		{attrs: ` type="application/ld+json"`},
		{attrs: ` type="importmap"`},
		{attrs: ` type="text/template"`},
		{attrs: ` type="speculationrules"`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.attrs, func(t *testing.T) {
			t.Parallel()

			if got := isJavaScript(testCase.attrs); got != testCase.want {
				t.Fatalf("javascript %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsModule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		attrs string
		want  bool
	}{
		{attrs: ` context="module"`, want: true},
		{attrs: ` module`, want: true},
		{attrs: ` module lang="ts"`, want: true},
		{attrs: ` context="other"`},
		{attrs: ` type="module"`},
		{attrs: ` title="a module b"`},
		{attrs: ``},
	}
	for _, testCase := range testCases {
		t.Run(testCase.attrs, func(t *testing.T) {
			t.Parallel()

			if got := isModule(testCase.attrs); got != testCase.want {
				t.Fatalf("module %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestUnquote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		raw  string
		want string
	}{
		{raw: `"ts" x`, want: "ts"},
		{raw: `'ts'`, want: "ts"},
		{raw: `"open`, want: "open"},
		{raw: `bare />`, want: "bare"},
		{raw: `bare`, want: "bare"},
		{raw: `{a > b} x`, want: "{a > b}"},
		{raw: ``, want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			t.Parallel()

			if got := unquote(testCase.raw); got != testCase.want {
				t.Fatalf("unquote %q, want %q", got, testCase.want)
			}
		})
	}
}
