package config

import "testing"

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	if cfg.Emoji.WindowSize != 5 {
		t.Errorf("WindowSize = %d, want 5", cfg.Emoji.WindowSize)
	}
	if cfg.Emoji.SoftWindowSize != 20 {
		t.Errorf("SoftWindowSize = %d, want 20", cfg.Emoji.SoftWindowSize)
	}
	if cfg.Emoji.SuggestionCount != 7 {
		t.Errorf("SuggestionCount = %d, want 7", cfg.Emoji.SuggestionCount)
	}
	if cfg.Emoji.SoftWindowSize <= cfg.Emoji.WindowSize {
		t.Error("the advisory window must reach past the blocking one")
	}
}

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Analyzers["disabled"] = AnalyzerConfig{Enabled: false}

	testCases := []struct {
		name string
		id   string
		want bool
	}{
		{"absent defaults to enabled", "nonexistent", true},
		{"explicitly disabled", "disabled", false},
		{"explicitly enabled", "other", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := cfg.IsEnabled(testCase.id); got != testCase.want {
				t.Fatalf("IsEnabled(%q) = %v, want %v", testCase.id, got, testCase.want)
			}
		})
	}
}

func TestParam(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Analyzers["file-size"] = AnalyzerConfig{
		Enabled: true,
		Params:  map[string]any{"max_lines": 200},
	}

	if got := cfg.Param("file-size", "max_lines", 180); got != 200 {
		t.Errorf("Param = %v, want 200", got)
	}
	if got := cfg.Param("file-size", "unknown", 99); got != 99 {
		t.Errorf("Param unknown = %v, want 99", got)
	}
	if got := cfg.Param("nonexistent", "key", 42); got != 42 {
		t.Errorf("Param on absent analyzer = %v, want 42", got)
	}
}

func TestSetParam(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		seed       func(*GovernanceConfig)
		analyzerID string
		key        string
		value      any
		wantParam  any
		wantOthers map[string]any
	}{
		{
			name:       "creates the entry when the analyzer has none",
			seed:       func(*GovernanceConfig) {},
			analyzerID: "golangci-lint",
			key:        "config_path",
			value:      "./golangci.yml",
			wantParam:  "./golangci.yml",
		},
		{
			name: "overwrites an existing entry without dropping its other params",
			seed: func(cfg *GovernanceConfig) {
				cfg.Analyzers["golangci-lint"] = AnalyzerConfig{
					Enabled: false,
					Params:  map[string]any{"timeout": 30},
				}
			},
			analyzerID: "golangci-lint",
			key:        "config_path",
			value:      "./golangci.yml",
			wantParam:  "./golangci.yml",
			wantOthers: map[string]any{"timeout": 30},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := Defaults()
			testCase.seed(cfg)
			cfg.SetParam(testCase.analyzerID, testCase.key, testCase.value)

			if got := cfg.Param(testCase.analyzerID, testCase.key, nil); got != testCase.wantParam {
				t.Errorf("Param(%q) = %v, want %v", testCase.key, got, testCase.wantParam)
			}
			for otherKey, want := range testCase.wantOthers {
				if got := cfg.Param(testCase.analyzerID, otherKey, nil); got != want {
					t.Errorf("Param(%q) = %v, want %v", otherKey, got, want)
				}
			}
		})
	}
}
