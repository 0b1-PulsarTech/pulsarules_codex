package governance

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestElseSeverityIsRaisedWithoutNesting(t *testing.T) {
	t.Parallel()

	deep := "package x\n\nfunc f(ok bool) int {\n" +
		strings.Repeat("if ok {\n", 5) + "if ok {\nreturn 1\n} else {\nreturn 2\n}\n" +
		strings.Repeat("}\n", 5) + "return 0\n}\n"
	dir := t.TempDir()
	writeFileAtDir(t, dir, "deep.go", deep)
	testCases := []struct {
		name        string
		overrides   []pass.Override
		wantElse    string
		wantNesting string
	}{
		{name: "both at warning by default", wantElse: "warning", wantNesting: "warning"},
		{
			name: "control-flow/else raised to error leaves nesting at warning",
			overrides: []pass.Override{
				pass.NewOverride([]string{"**"}, map[string]pass.OverrideConfig{
					"control-flow/else": {Severity: "error"},
				}),
			},
			wantElse:    "error",
			wantNesting: "warning",
		},
		{
			name: "control-flow raised to error reaches both",
			overrides: []pass.Override{
				pass.NewOverride([]string{"**"}, map[string]pass.OverrideConfig{
					"control-flow": {Severity: "error"},
				}),
			},
			wantElse:    "error",
			wantNesting: "error",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Defaults()
			cfg.ApplyPreset()
			cfg.Overrides = testCase.overrides
			got := severitiesByFile(t, dir, cfg)
			if got["control-flow/else deep.go"] != testCase.wantElse {
				t.Errorf("else severity = %q, want %q (all: %v)",
					got["control-flow/else deep.go"], testCase.wantElse, got)
			}
			if got["control-flow deep.go"] != testCase.wantNesting {
				t.Errorf("nesting severity = %q, want %q (all: %v)",
					got["control-flow deep.go"], testCase.wantNesting, got)
			}
		})
	}
}
