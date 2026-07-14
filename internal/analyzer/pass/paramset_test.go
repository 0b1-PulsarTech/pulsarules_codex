package pass

import "testing"

func TestParamSetInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		key      string
		fallback int
		want     int
	}{
		{"nil params keeps fallback", nil, "max_lines", 300, 300},
		{"missing key keeps fallback", ParamSet{}, "max_lines", 300, 300},
		{"int value wins", ParamSet{"max_lines": 180}, "max_lines", 300, 180},
		{"int64 value wins", ParamSet{"max_lines": int64(180)}, "max_lines", 300, 180},
		{"float64 value wins", ParamSet{"max_lines": float64(180)}, "max_lines", 300, 180},
		{"numeric string wins", ParamSet{"max_lines": "180"}, "max_lines", 300, 180},
		{
			"non-numeric string keeps fallback",
			ParamSet{"max_lines": "eight"},
			"max_lines",
			300,
			300,
		},
		{"zero value keeps fallback", ParamSet{"max_lines": 0}, "max_lines", 300, 300},
		{"negative value keeps fallback", ParamSet{"max_lines": -1}, "max_lines", 300, 300},
		{"wrong type keeps fallback", ParamSet{"max_lines": true}, "max_lines", 300, 300},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.params.Int(testCase.key, testCase.fallback)
			if got != testCase.want {
				t.Errorf("Int(%q) = %d, want %d", testCase.key, got, testCase.want)
			}
		})
	}
}

func TestAnalysisContextParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *Context
		id   string
		want int
	}{
		{"nil config returns nil ParamSet", &Context{}, "file-size", 300},
		{
			name: "missing analyzer returns nil ParamSet",
			ctx: &Context{
				Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{}},
			},
			id:   "file-size",
			want: 300,
		},
		{
			name: "present analyzer returns its params",
			ctx: &Context{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
				"file-size": {Params: map[string]any{"max_lines": 180}},
			}}},
			id:   "file-size",
			want: 180,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.ctx.Params(testCase.id).Int("max_lines", 300)
			if got != testCase.want {
				t.Errorf(
					"Params(%q).Int(max_lines) = %d, want %d",
					testCase.id,
					got,
					testCase.want,
				)
			}
		})
	}
}
