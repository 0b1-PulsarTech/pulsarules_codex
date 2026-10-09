package pathalias

import (
	"slices"
	"testing"
)

func TestMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, pattern, spec, wantStar string
		want                          bool
	}{
		{name: "exact", pattern: "$lib", spec: "$lib", want: true},
		{name: "exact miss", pattern: "$lib", spec: "$lib/x"},
		{name: "prefix star", pattern: "#lib/*", spec: "#lib/a/b", wantStar: "a/b", want: true},
		{name: "prefix miss", pattern: "#lib/*", spec: "#libx/a"},
		{name: "suffix star", pattern: "@/*.js", spec: "@/a.js", wantStar: "a", want: true},
		{name: "suffix miss", pattern: "@/*.js", spec: "@/a.ts"},
		{name: "too short", pattern: "ab*cd", spec: "abc"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			star, ok := Match(testCase.pattern, testCase.spec)
			if ok != testCase.want || star != testCase.wantStar {
				t.Fatalf("Match = %q, %v", star, ok)
			}
		})
	}
}

func TestAlias_ResolveAndSpelling(t *testing.T) {
	t.Parallel()

	lib := Alias{Spec: "#lib/*", Target: "web/src/lib/*"}
	root := Alias{Spec: "@/*", Target: "*"}
	exact := Alias{Spec: "$cfg", Target: "web/config"}
	testCases := []struct {
		name         string
		alias        Alias
		spec, target string
		wantResolve  string
		wantSpelling string
	}{
		{name: "star", alias: lib, spec: "#lib/a/b", wantResolve: "web/src/lib/a/b"},
		{name: "no match", alias: lib, spec: "other"},
		{name: "spelling under", alias: lib, target: "web/src/lib/a", wantSpelling: "#lib/a"},
		{name: "spelling outside", alias: lib, target: "web/src/other/a"},
		{name: "spelling is the dir itself", alias: lib, target: "web/src/lib"},
		{name: "root alias", alias: root, target: "a/b", wantSpelling: "@/a/b"},
		{name: "root alias outside", alias: root, target: "../a"},
		{
			name: "exact", alias: exact, spec: "$cfg", target: "web/config",
			wantResolve: "web/config", wantSpelling: "$cfg",
		},
		{name: "exact other target", alias: exact, target: "web/config/x"},
		{
			name: "middle star target", alias: Alias{Spec: "#a/*", Target: "src/*.ts"},
			spec: "#a/x", target: "src/x.ts", wantResolve: "src/x.ts",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			resolved, _ := testCase.alias.Resolve(testCase.spec)
			if resolved != testCase.wantResolve {
				t.Fatalf("Resolve = %q, want %q", resolved, testCase.wantResolve)
			}
			spelling, _ := testCase.alias.Spelling(testCase.target)
			if spelling != testCase.wantSpelling {
				t.Fatalf("Spelling = %q, want %q", spelling, testCase.wantSpelling)
			}
		})
	}
}

func TestSpecs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		aliases []Alias
		want    []string
	}{
		{
			name:    "in order",
			aliases: []Alias{{Spec: "a", Target: "a"}, {Spec: "b/*", Target: "b/*"}},
			want:    []string{"a", "b/*"},
		},
		{
			name: "repeated spec once",
			aliases: []Alias{
				{Spec: "a", Target: "a"}, {Spec: "a", Target: "a", Origin: OriginPaths},
			},
			want: []string{"a"},
		},
		{
			name: "unusable targets are not local",
			aliases: []Alias{
				{Spec: "react", Target: "node_modules/@types/react"},
				{Spec: "virtual:x", Target: "src/x.d.ts"},
				{Spec: "ok", Target: "src/ok.ts"},
			},
			want: []string{"ok"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := Specs(testCase.aliases); !slices.Equal(got, testCase.want) {
				t.Fatalf("Specs = %v, want %v", got, testCase.want)
			}
		})
	}
}
