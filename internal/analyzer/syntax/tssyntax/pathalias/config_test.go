package pathalias

import (
	"slices"
	"testing"
)

func readerOf(files map[string]string) func(string) ([]byte, bool) {
	return func(name string) ([]byte, bool) {
		body, ok := files[name]
		return []byte(body), ok
	}
}

type pathCase struct {
	name  string
	files map[string]string
	want  []Alias
}

func pathCases() []pathCase {
	return append(pathBaseCases(), pathEdgeCases()...)
}

func pathBaseCases() []pathCase {
	return []pathCase{
		{
			name: "paths without baseUrl are relative to the config",
			files: map[string]string{"app/tsconfig.json": `{"compilerOptions":
				{"paths": {"@/*": ["./src/*"], "ui": ["./ui/index.ts"]}}}`},
			want: []Alias{
				{
					Spec:   "@/*",
					Target: "app/src/*",
					Origin: OriginPaths,
				},
				{Spec: "ui", Target: "app/ui/index.ts", Origin: OriginPaths},
			},
		},
		{
			name: "baseUrl wins as the paths base",
			files: map[string]string{"tsconfig.json": `{"compilerOptions":
				{"baseUrl": "src", "paths": {"lib/*": ["lib/*"]}}}`},
			want: []Alias{{Spec: "lib/*", Target: "src/lib/*", Origin: OriginPaths}},
		},
		{
			name: "comments and trailing commas",
			files: map[string]string{"jsconfig.json": "{\n// c\n\"compilerOptions\": {\n" +
				"\"paths\": {\"x/*\": [\"./x/*\",],}, /* d */\n},\n}"},
			want: []Alias{{Spec: "x/*", Target: "x/*", Origin: OriginPaths}},
		},
		{
			name: "extends chain child overrides paths and parent gives baseUrl",
			files: map[string]string{
				"tsconfig.json": `{"extends": "./base/tsconfig", "compilerOptions":
					{"paths": {"a/*": ["a/*"]}}}`,
				"base/tsconfig.json": `{"compilerOptions":
					{"baseUrl": ".", "paths": {"b/*": ["b/*"]}}}`,
			},
			want: []Alias{{Spec: "a/*", Target: "base/a/*", Origin: OriginPaths}},
		},
		{
			name: "extends cycle stops",
			files: map[string]string{
				"tsconfig.json": `{"extends": "./a.json",
					"compilerOptions": {"paths": {"x": ["x"]}}}`,
				"a.json": `{"extends": "./tsconfig.json"}`,
			},
			want: []Alias{{Spec: "x", Target: "x", Origin: OriginPaths}},
		},
	}
}

func pathEdgeCases() []pathCase {
	return []pathCase{
		{
			name: "extends hop cap",
			files: map[string]string{
				"tsconfig.json": `{"extends": "./c1.json"}`,
				"c1.json":       `{"extends": "./c2.json"}`,
				"c2.json":       `{"extends": "./c3.json"}`,
				"c3.json":       `{"extends": "./c4.json"}`,
				"c4.json":       `{"extends": "./c5.json"}`,
				"c5.json":       `{"compilerOptions": {"paths": {"deep": ["deep"]}}}`,
			},
		},
		{
			name: "package extends is ignored",
			files: map[string]string{"tsconfig.json": `{"extends": "@app/tsconfig",
				"compilerOptions": {"paths": {"x": ["x"]}}}`},
			want: []Alias{{Spec: "x", Target: "x", Origin: OriginPaths}},
		},
		{
			name: "array extends is ignored",
			files: map[string]string{"tsconfig.json": `{"extends": ["./a.json"],
				"compilerOptions": {"paths": {"x": ["x"]}}}`},
			want: []Alias{{Spec: "x", Target: "x", Origin: OriginPaths}},
		},
		{
			name: "star-only, empty and escaping targets are skipped",
			files: map[string]string{"tsconfig.json": `{"compilerOptions": {"paths":
				{"*": ["x/*"], "e": [], "out": ["../../o"]}}}`},
		},
		{name: "invalid json", files: map[string]string{"tsconfig.json": `{`}},
		{name: "missing files"},
	}
}

func TestFinder_PathAliases(t *testing.T) {
	t.Parallel()

	testCases := pathCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			finder := NewFinder(readerOf(testCase.files))
			dir := "."
			if _, nested := testCase.files["app/tsconfig.json"]; nested {
				dir = "app"
			}
			got := finder.pathAliases(dir)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("aliases\n got %v\nwant %v", got, testCase.want)
			}
		})
	}
}
