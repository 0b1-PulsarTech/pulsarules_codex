package pass

import "testing"

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
			ctx:  configuredCtx("top-of-file", []string{".dart"}),
			fc:   FileChange{Extension: ".dart"},
			want: true,
		},
		{
			name: "configured override narrows out the fallback's own entries",
			ctx:  configuredCtx("top-of-file", []string{".dart"}),
			fc:   FileChange{Extension: ".go"},
			want: false,
		},
		{
			name: "a bare spelling without the leading dot still matches",
			ctx:  configuredCtx("top-of-file", []string{"go"}),
			fc:   FileChange{Extension: ".go"},
			want: true,
		},
		{
			name: "matching is case-insensitive",
			ctx:  configuredCtx("top-of-file", []string{".GO"}),
			fc:   FileChange{Extension: ".go"},
			want: true,
		},
		{
			name: "a config set under another analyzer id is not consulted (it must not " +
				"leak across shared call sites)",
			ctx:  configuredCtx("big-comment", []string{".dart"}),
			fc:   FileChange{Extension: ".dart"},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			filter := ExtensionFilter(testCase.ctx, "top-of-file", []string{".go"})
			if got := filter(testCase.fc); got != testCase.want {
				t.Errorf("filter(%+v) = %v, want %v", testCase.fc, got, testCase.want)
			}
		})
	}
}
