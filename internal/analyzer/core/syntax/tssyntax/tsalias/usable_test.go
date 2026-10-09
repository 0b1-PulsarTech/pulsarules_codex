package tsalias

import "testing"

func TestAlias_Usable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		target string
		want   bool
	}{
		{name: "source file", target: "src/env.ts", want: true},
		{name: "wildcard directory", target: "src/lib/*", want: true},
		{name: "declaration", target: "src/virtual.d.ts"},
		{name: "module declaration", target: "src/virtual.d.mts"},
		{name: "commonjs declaration", target: "src/virtual.d.cts"},
		{name: "wildcard declaration", target: "src/x/*.d.ts"},
		{name: "dependency directory", target: "node_modules/@types/react"},
		{name: "nested dependency", target: "web/node_modules/pkg/index.js"},
		{name: "similar name", target: "src/node_modules_shim.ts", want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := (Alias{Target: testCase.target}).Usable(); got != testCase.want {
				t.Fatalf("Usable(%q) = %v", testCase.target, got)
			}
		})
	}
}

func TestAlias_Serves(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		alias    Alias
		importer string
		want     bool
	}{
		{name: "imports everywhere", alias: Alias{}, importer: "web/scripts/a.mjs", want: true},
		{
			name:     "paths for ts",
			alias:    Alias{Origin: OriginPaths},
			importer: "web/a.ts",
			want:     true,
		},
		{
			name:     "paths for svelte",
			alias:    Alias{Origin: OriginPaths},
			importer: "a.svelte",
			want:     true,
		},
		{name: "paths not for js", alias: Alias{Origin: OriginPaths}, importer: "a.js"},
		{name: "paths not for mjs", alias: Alias{Origin: OriginPaths}, importer: "a.mjs"},
		{name: "paths not for cjs", alias: Alias{Origin: OriginPaths}, importer: "a.cjs"},
		{
			name:     "kit inside src",
			alias:    Alias{Origin: OriginKit, Root: "web"},
			importer: "web/src/routes/a.js", want: true,
		},
		{
			name:     "kit outside src",
			alias:    Alias{Origin: OriginKit, Root: "web"},
			importer: "web/scripts/a.mjs",
		},
		{
			name:     "kit at the repo root",
			alias:    Alias{Origin: OriginKit, Root: "."},
			importer: "src/a.ts", want: true,
		},
		{
			name:     "kit sibling named like src",
			alias:    Alias{Origin: OriginKit, Root: "web"},
			importer: "web/src2/a.ts",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.alias.Serves(testCase.importer); got != testCase.want {
				t.Fatalf("Serves(%q) = %v", testCase.importer, got)
			}
		})
	}
}

func TestBest(t *testing.T) {
	t.Parallel()

	aliases := []Alias{
		{Spec: "@/*", Target: "src/*"},
		{Spec: "@/ui/*", Target: "ui/*"},
		{Spec: "@/ui/card", Target: "card.ts"},
		{Spec: "@/ui/*", Target: "dup/*"},
	}
	testCases := []struct {
		name, spec, want string
		found            bool
	}{
		{name: "shorter prefix", spec: "@/a", want: "src/*", found: true},
		{name: "longest prefix", spec: "@/ui/x", want: "ui/*", found: true},
		{name: "exact beats prefix", spec: "@/ui/card", want: "card.ts", found: true},
		{name: "none", spec: "other"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, found := Best(aliases, testCase.spec)
			if found != testCase.found || got.Target != testCase.want {
				t.Fatalf("Best(%q) = %+v, %v", testCase.spec, got, found)
			}
		})
	}
}

func TestSpellable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, resolved, spec string
		want                 bool
	}{
		{name: "no extension", resolved: "src/lib/db", spec: "$lib/db", want: true},
		{name: "js extension", resolved: "src/lib/db.js", spec: "#lib/db", want: true},
		{name: "ts written out", resolved: "src/env.ts", spec: "@/env.ts", want: true},
		{name: "ts missing from spec", resolved: "src/env.ts", spec: "@env"},
		{name: "tsx missing", resolved: "src/a.tsx", spec: "@a"},
		{name: "mts missing", resolved: "src/a.mts", spec: "@a"},
		{name: "cts missing", resolved: "src/a.cts", spec: "@a"},
		{name: "declaration", resolved: "src/a.d.ts", spec: "@a.d.ts"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := Spellable(testCase.resolved, testCase.spec); got != testCase.want {
				t.Fatalf("Spellable(%q, %q) = %v", testCase.resolved, testCase.spec, got)
			}
		})
	}
}
