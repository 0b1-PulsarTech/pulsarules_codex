package pass

import (
	"slices"
	"testing"
)

func configuredCtx(analyzerID string, extensions any) *Context {
	return &Context{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
		analyzerID: {Enabled: true, Params: map[string]any{ParamExtensions: extensions}},
	}}}
}

func TestExtensionFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *Context
		fc   FileChange
		want bool
	}{
		{
			name: "no config falls back to the given default",
			ctx:  &Context{},
			fc:   FileChange{Extension: ".go"},
			want: true,
		},
		{
			name: "extension absent from the default is rejected",
			ctx:  &Context{},
			fc:   FileChange{Extension: ".dart"},
			want: false,
		},
		{
			name: "configured override widens the set",
			ctx:  configuredCtx("file-size", []string{".dart"}),
			fc:   FileChange{Extension: ".dart"},
			want: true,
		},
		{
			name: "configured override narrows out the fallback's own entries",
			ctx:  configuredCtx("file-size", []string{".dart"}),
			fc:   FileChange{Extension: ".go"},
			want: false,
		},
		{
			name: "a bare spelling without the leading dot still matches",
			ctx:  configuredCtx("file-size", []string{"go"}),
			fc:   FileChange{Extension: ".go"},
			want: true,
		},
		{
			name: "matching is case-insensitive",
			ctx:  configuredCtx("file-size", []string{".GO"}),
			fc:   FileChange{Extension: ".go"},
			want: true,
		},
		{
			name: "a config set under another analyzer id is not consulted (it must not " +
				"leak across shared call sites)",
			ctx:  configuredCtx("typographic-markers", []string{".dart"}),
			fc:   FileChange{Extension: ".dart"},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			filter := ExtensionFilter(testCase.ctx, "file-size", []string{".go"})
			if got := filter(testCase.fc); got != testCase.want {
				t.Errorf("filter(%+v) = %v, want %v", testCase.fc, got, testCase.want)
			}
		})
	}
}

func TestIsTestPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		want bool
	}{
		{"go test suffix", "internal/foo/bar_test.go", true},
		{"dart test suffix", "lib/foo_test.dart", true},
		{"js dot-test convention", "src/foo.test.js", true},
		{"ts dot-spec convention", "src/foo.spec.ts", true},
		{"underscore spec convention", "src/foo_spec.rb", true},
		{"production go file", "internal/foo/bar.go", false},
		{"a directory merely named test is not a marker", "internal/test/bar.go", false},
		{"matching is case-insensitive", "internal/foo/BAR_TEST.GO", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := IsTestPath(testCase.path); got != testCase.want {
				t.Errorf("IsTestPath(%q) = %v, want %v", testCase.path, got, testCase.want)
			}
		})
	}
}

func TestDefaultSourceExtensions_ClonedEachCall(t *testing.T) {
	t.Parallel()

	first := DefaultSourceExtensions()
	first[0] = "mutated"
	second := DefaultSourceExtensions()
	if second[0] == "mutated" {
		t.Fatal("DefaultSourceExtensions must clone, not share, its backing slice")
	}
}

func TestDefaultSourceExtensions_IncludesNix(t *testing.T) {
	t.Parallel()

	if !slices.Contains(DefaultSourceExtensions(), ".nix") {
		t.Fatal("DefaultSourceExtensions must include .nix, which langspec registers")
	}
}
