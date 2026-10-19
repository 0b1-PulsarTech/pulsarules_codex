package pass

import (
	"maps"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
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

func TestParamSetSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		fallback finding.Severity
		want     finding.Severity
	}{
		{
			name:     "error",
			params:   ParamSet{"severity": "error"},
			fallback: finding.SeverityWarning,
			want:     finding.SeverityError,
		},
		{
			name:     "warning",
			params:   ParamSet{"severity": "warning"},
			fallback: finding.SeverityError,
			want:     finding.SeverityWarning,
		},
		{
			name:     "info",
			params:   ParamSet{"severity": "info"},
			fallback: finding.SeverityError,
			want:     finding.SeverityInfo,
		},
		{
			name:     "absent key keeps the fallback",
			params:   ParamSet{},
			fallback: finding.SeverityError,
			want:     finding.SeverityError,
		},
		{
			name:     "nil set keeps the fallback",
			params:   nil,
			fallback: finding.SeverityError,
			want:     finding.SeverityError,
		},
		{
			name:     "unrecognized value keeps the fallback",
			params:   ParamSet{"severity": "fatal"},
			fallback: finding.SeverityError,
			want:     finding.SeverityError,
		},
		{
			name:     "non-string value keeps the fallback",
			params:   ParamSet{"severity": 2},
			fallback: finding.SeverityError,
			want:     finding.SeverityError,
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

func TestParamSetNonNegativeInt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   ParamSet
		fallback int
		want     int
	}{
		{"nil params keeps fallback", nil, 290, 290},
		{"missing key keeps fallback", ParamSet{}, 290, 290},
		{"explicit zero is kept", ParamSet{"error_lines": 0}, 290, 0},
		{"positive value wins", ParamSet{"error_lines": 400}, 290, 400},
		{"numeric string wins", ParamSet{"error_lines": "400"}, 290, 400},
		{"negative value keeps fallback", ParamSet{"error_lines": -1}, 290, 290},
		{"wrong type keeps fallback", ParamSet{"error_lines": true}, 290, 290},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.params.NonNegativeInt("error_lines", testCase.fallback)
			if got != testCase.want {
				t.Errorf("NonNegativeInt = %d, want %d", got, testCase.want)
			}
		})
	}
}

func scopedOverride(files []string, params map[string]any) Override {
	return Override{
		Files:     glob.CompileSet(files),
		Analyzers: map[string]OverrideConfig{"file-size": {Params: params}},
	}
}

type paramsAtCase struct {
	name      string
	base      map[string]any
	overrides []Override
	path      string
	want      ParamSet
}

func runParamsAtCases(t *testing.T, testCases []paramsAtCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := &Context{Config: &AnalysisConfig{
				Analyzers: map[string]AnalyzerConfig{"file-size": {Params: testCase.base}},
				Overrides: testCase.overrides,
			}}
			got := ctx.ParamsAt("file-size", testCase.path)
			if len(got) != len(testCase.want) || !maps.Equal(got, testCase.want) {
				t.Errorf("ParamsAt(%q) = %v, want %v", testCase.path, got, testCase.want)
			}
		})
	}
}

func TestAnalysisContextParamsAt(t *testing.T) {
	t.Parallel()

	svelte := []string{"**/*.svelte"}
	runParamsAtCases(t, []paramsAtCase{
		{
			"no override keeps the base",
			map[string]any{"max_lines": 180},
			nil, "a/b.svelte",
			ParamSet{"max_lines": 180},
		},
		{
			"matching glob adds params", nil,
			[]Override{scopedOverride(svelte, map[string]any{"max_lines": 150})},
			"a/b.svelte",
			ParamSet{"max_lines": 150},
		},
		{
			"non matching glob is ignored",
			map[string]any{"max_lines": 180},
			[]Override{scopedOverride(svelte, map[string]any{"max_lines": 150})},
			"a/b.go",
			ParamSet{"max_lines": 180},
		},
		{
			"override without params is skipped",
			map[string]any{"max_lines": 180},
			[]Override{scopedOverride(svelte, nil)},
			"a/b.svelte",
			ParamSet{"max_lines": 180},
		},
	})
}

func TestAnalysisContextParamsAt_Precedence(t *testing.T) {
	t.Parallel()

	runParamsAtCases(t, []paramsAtCase{
		{
			"later override wins per key",
			map[string]any{"max_lines": 180},
			[]Override{
				scopedOverride([]string{"**/*.svelte"}, map[string]any{
					"max_lines": 150, "error_lines": 300,
				}),
				scopedOverride([]string{"a/*.svelte"}, map[string]any{"max_lines": 100}),
			},
			"a/b.svelte",
			ParamSet{"max_lines": 100, "error_lines": 300},
		},
		{
			"no files matches everything", nil,
			[]Override{scopedOverride(nil, map[string]any{"error_lines": 0})},
			"x.go",
			ParamSet{"error_lines": 0},
		},
		{
			"invalid glob matches nothing", nil,
			[]Override{scopedOverride([]string{""}, map[string]any{"max_lines": 1})},
			"x.go",
			nil,
		},
	})
}

func TestParamsAt_NilConfigAndBase(t *testing.T) {
	t.Parallel()

	if got := (&Context{}).ParamsAt("file-size", "a.go"); len(got) != 0 {
		t.Fatalf("nil config resolved %v, want empty", got)
	}
	base := map[string]any{"max_lines": 180}
	ctx := &Context{Config: &AnalysisConfig{
		Analyzers: map[string]AnalyzerConfig{"file-size": {Params: base}},
		Overrides: []Override{scopedOverride(nil, map[string]any{"max_lines": 50})},
	}}
	ctx.ParamsAt("file-size", "a.go")
	if base["max_lines"] != 180 {
		t.Fatalf("base params mutated: %v", base)
	}
}

//nolint:paralleltest // testing.AllocsPerRun panics inside a parallel test
func TestOverrideCoversReusesTheCompiledSet(t *testing.T) {
	files := []string{"web/**/*.svelte", "lib/**"}
	override := NewOverride(files, nil)
	paths := []string{"web/a/b.svelte", "lib/x.go", "cmd/main.go"}
	covers := testing.AllocsPerRun(50, func() {
		for _, path := range paths {
			override.Covers(path)
		}
	})
	recompiled := testing.AllocsPerRun(50, func() {
		for _, path := range paths {
			glob.CompileSet(files).Match(path)
		}
	})
	if covers >= recompiled {
		t.Fatalf("Covers allocs %v >= recompiling per path %v: globs not compiled once",
			covers, recompiled)
	}
}
