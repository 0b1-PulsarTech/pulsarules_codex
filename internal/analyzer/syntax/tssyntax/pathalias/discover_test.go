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
				got := finder.For(testCase.file)
				if !slices.Equal(got, testCase.want) {
					t.Fatalf("aliases\n got %v\nwant %v", got, testCase.want)
				}
			}
		})
	}
}
