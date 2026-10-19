package pathalias

import (
	"slices"
	"testing"
)

type forCase struct {
	name  string
	files map[string]string
	file  string
	want  []Alias
}

func forCases() []forCase {
	return []forCase{
		{
			name: "package imports with conditions",
			files: map[string]string{"web/package.json": `{"imports": {
				"#lib/*": "./src/lib/*",
				"#env": {"types": {"x": 1}, "default": "./src/env.ts", "node": "./n.ts"},
				"#dep": "left-pad", "bad": "./bad", "#num": 3, "#empty": {}, "#arr": []}}`},
			file: "web/src/a/b.ts",
			want: []Alias{
				{Spec: "#env", Target: "web/src/env.ts"},
				{Spec: "#lib/*", Target: "web/src/lib/*"},
			},
		},
		{
			name: "nearest package.json wins",
			files: map[string]string{
				"package.json":     `{"imports": {"#root/*": "./r/*"}}`,
				"pkg/package.json": `{"imports": {"#pkg/*": "./p/*"}}`,
			},
			file: "pkg/deep/x.ts",
			want: []Alias{{Spec: "#pkg/*", Target: "pkg/p/*"}},
		},
		{
			name:  "root package.json",
			files: map[string]string{"package.json": `{"imports": {"#r": "./r.ts"}}`},
			file:  "x.ts",
			want:  []Alias{{Spec: "#r", Target: "r.ts"}},
		},
		{
			name: "no package.json means no aliases",
			files: map[string]string{"tsconfig.json": `{"compilerOptions":
				{"paths": {"x": ["x"]}}}`},
			file: "a/b.ts",
		},
		{
			name:  "invalid package.json",
			files: map[string]string{"package.json": `{`},
			file:  "a.ts",
		},
		{
			name: "sveltekit lib comes from the dependency",
			files: map[string]string{"web/package.json": `{"devDependencies":
				{"@sveltejs/kit": "^2"}}`},
			file: "web/src/routes/a.svelte",
			want: []Alias{kitAlias("web", "$lib/*", "web/src/lib/*")},
		},
		{
			name: "sveltekit lib keeps the paths and the kit spelling apart",
			files: map[string]string{
				"package.json":  `{"dependencies": {"@sveltejs/kit": "^2"}}`,
				"tsconfig.json": `{"compilerOptions": {"paths": {"$lib/*": ["custom/*"]}}}`,
			},
			file: "a.ts",
			want: []Alias{
				{Spec: "$lib/*", Target: "custom/*", Origin: OriginPaths},
				kitAlias(".", "$lib/*", "src/lib/*"),
			},
		},
		{
			name: "types condition is skipped for a runtime target",
			files: map[string]string{"package.json": `{"imports": {
				"#x/*": {"types": "./x/*.d.ts", "default": "./x/*.js"},
				"#y": {"import": {"types": "./y.d.ts", "default": "./y.js"}},
				"#z": {"types": "./z.d.ts"}}}`},
			file: "a.ts",
			want: []Alias{
				{Spec: "#x/*", Target: "x/*.js"},
				{Spec: "#y", Target: "y.js"},
			},
		},
	}
}

func TestFinder_For(t *testing.T) {
	t.Parallel()

	testCases := forCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			finder := NewFinder(readerOf(testCase.files))
			for range 2 {
				_, got := finder.Lookup(testCase.file)
				if !slices.Equal(got, testCase.want) {
					t.Fatalf("aliases\n got %v\nwant %v", got, testCase.want)
				}
			}
		})
	}
}

func TestFinderLookupRoot(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"web/package.json": `{"imports": {"#a": "./src/a.ts"}}`,
		"other/readme.md":  "no manifest",
	}
	testCases := []struct {
		name        string
		lookups     []string
		wantRoot    string
		wantAliases int
	}{
		{"file under a manifest", []string{"web/src/deep/x.ts"}, "web", 1},
		{"file without a manifest", []string{"other/x.ts"}, "", 0},
		{
			"second file of a project is served from the cache",
			[]string{"web/a.ts", "web/b/c.ts"},
			"web",
			1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			finder := NewFinder(readerOf(files))
			for _, file := range testCase.lookups {
				root, aliases := finder.Lookup(file)
				if root != testCase.wantRoot || len(aliases) != testCase.wantAliases {
					t.Fatalf("Lookup(%q) = %q, %v; want %q and %d aliases",
						file, root, aliases, testCase.wantRoot, testCase.wantAliases)
				}
			}
		})
	}
}

func TestUnique(t *testing.T) {
	t.Parallel()

	first := Alias{Spec: "$x", Target: "a", Origin: OriginPaths}
	testCases := []struct {
		name  string
		input []Alias
		want  []Alias
	}{
		{name: "empty"},
		{
			name:  "same origin and spec keeps the first",
			input: []Alias{first, {Spec: "$x", Target: "b", Origin: OriginPaths}},
			want:  []Alias{first},
		},
		{
			name:  "same spec from another origin stays",
			input: []Alias{first, {Spec: "$x", Target: "b", Origin: OriginKit}},
			want:  []Alias{first, {Spec: "$x", Target: "b", Origin: OriginKit}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := unique(testCase.input); !slices.Equal(got, testCase.want) {
				t.Fatalf("unique = %v, want %v", got, testCase.want)
			}
		})
	}
}
