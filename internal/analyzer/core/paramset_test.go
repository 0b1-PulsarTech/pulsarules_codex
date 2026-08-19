package core

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

func TestParamSetBool(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		key      string
		fallback bool
		want     bool
	}{
		{"nil params keeps fallback", nil, "enabled", true, true},
		{"missing key keeps fallback", ParamSet{}, "enabled", false, false},
		{"bool value wins", ParamSet{"enabled": true}, "enabled", false, true},
		{"parseable string wins", ParamSet{"enabled": "false"}, "enabled", true, false},
		{"unparseable string keeps fallback", ParamSet{"enabled": "nope"}, "enabled", true, true},
		{"wrong type keeps fallback", ParamSet{"enabled": 1}, "enabled", false, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.params.Bool(testCase.key, testCase.fallback)
			if got != testCase.want {
				t.Errorf("Bool(%q) = %v, want %v", testCase.key, got, testCase.want)
			}
		})
	}
}

func TestParamSetString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		key      string
		fallback string
		want     string
	}{
		{"nil params keeps fallback", nil, "mode", "default", "default"},
		{"missing key keeps fallback", ParamSet{}, "mode", "default", "default"},
		{"string value wins", ParamSet{"mode": "strict"}, "mode", "default", "strict"},
		{"wrong type keeps fallback", ParamSet{"mode": 1}, "mode", "default", "default"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.params.String(testCase.key, testCase.fallback)
			if got != testCase.want {
				t.Errorf("String(%q) = %q, want %q", testCase.key, got, testCase.want)
			}
		})
	}
}

func TestAnalysisContextParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *AnalysisContext
		id   string
		want int
	}{
		{"nil config returns nil ParamSet", &AnalysisContext{}, "file-size", 300},
		{
			name: "missing analyzer returns nil ParamSet",
			ctx:  &AnalysisContext{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{}}},
			id:   "file-size",
			want: 300,
		},
		{
			name: "present analyzer returns its params",
			ctx: &AnalysisContext{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
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
				t.Errorf("Params(%q).Int(max_lines) = %d, want %d", testCase.id, got, testCase.want)
			}
		})
	}
}

// TestParamSetSeverity asserts the conventional "severity" key maps to each
// Severity, and that an absent or unrecognized value falls back rather than
// silently downgrading a finding to Info (the zero value).
func TestParamSetSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		fallback Severity
		want     Severity
	}{
		{
			name:     "error",
			params:   ParamSet{"severity": "error"},
			fallback: SeverityWarning,
			want:     SeverityError,
		},
		{
			name:     "warning",
			params:   ParamSet{"severity": "warning"},
			fallback: SeverityError,
			want:     SeverityWarning,
		},
		{
			name:     "info",
			params:   ParamSet{"severity": "info"},
			fallback: SeverityError,
			want:     SeverityInfo,
		},
		{
			name:     "absent key keeps the fallback",
			params:   ParamSet{},
			fallback: SeverityError,
			want:     SeverityError,
		},
		{
			name:     "nil set keeps the fallback",
			params:   nil,
			fallback: SeverityError,
			want:     SeverityError,
		},
		{
			name:     "unrecognized value keeps the fallback",
			params:   ParamSet{"severity": "fatal"},
			fallback: SeverityError,
			want:     SeverityError,
		},
		{
			name:     "non-string value keeps the fallback",
			params:   ParamSet{"severity": 2},
			fallback: SeverityError,
			want:     SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.params.Severity(testCase.fallback); got != testCase.want {
				t.Errorf("Severity(%v) = %v, want %v", testCase.fallback, got, testCase.want)
			}
		})
	}
}
