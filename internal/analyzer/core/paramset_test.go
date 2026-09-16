package core

import (
	"slices"
	"testing"
)

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

type paramSetStringTestCase struct {
	name     string
	params   ParamSet
	key      string
	fallback []string
	want     []string
}

func paramSetStringCases() []paramSetStringTestCase {
	return []paramSetStringTestCase{
		{"nil params keeps fallback", nil, "extensions", []string{".go"}, []string{".go"}},
		{
			"missing key keeps fallback",
			ParamSet{},
			"extensions",
			[]string{".go"},
			[]string{".go"},
		},
		{
			"[]string value wins",
			ParamSet{"extensions": []string{".dart", ".ts"}},
			"extensions",
			[]string{".go"},
			[]string{".dart", ".ts"},
		},
		{
			"[]any value wins (YAML/JSON decoder shape)",
			ParamSet{"extensions": []any{".dart", ".ts"}},
			"extensions",
			[]string{".go"},
			[]string{".dart", ".ts"},
		},
		{
			"[]any with a non-string entry drops it",
			ParamSet{"extensions": []any{".dart", 7, ".ts"}},
			"extensions",
			[]string{".go"},
			[]string{".dart", ".ts"},
		},
		{
			"comma-separated string wins",
			ParamSet{"extensions": ".dart, .ts ,.js"},
			"extensions",
			[]string{".go"},
			[]string{".dart", ".ts", ".js"},
		},
		{
			"empty []string keeps fallback",
			ParamSet{"extensions": []string{}},
			"extensions",
			[]string{".go"},
			[]string{".go"},
		},
		{
			"empty string keeps fallback",
			ParamSet{"extensions": ""},
			"extensions",
			[]string{".go"},
			[]string{".go"},
		},
		{
			"wrong type keeps fallback",
			ParamSet{"extensions": 7},
			"extensions",
			[]string{".go"},
			[]string{".go"},
		},
	}
}

func TestParamSetStrings(t *testing.T) {
	t.Parallel()

	for _, testCase := range paramSetStringCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.params.Strings(testCase.key, testCase.fallback)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Strings(%q) = %v, want %v", testCase.key, got, testCase.want)
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
			ctx: &AnalysisContext{
				Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{}},
			},
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

func TestValidateSeverityName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
	}{
		{name: "empty keeps the analyzer default"},
		{name: "error", severity: "error"},
		{name: "warning", severity: "warning"},
		{name: "info", severity: "info"},
		{name: "a typo is rejected", severity: "fatal", wantErr: true},
		{name: "capitalised is rejected", severity: "Warning", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSeverityName(testCase.severity)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ValidateSeverityName(%q) err = %v, wantErr %v",
					testCase.severity, err, testCase.wantErr)
			}
		})
	}
}
