package golangcilint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type forcedBaselineCase struct {
	name        string
	config      string
	wantCount   int
	wantMessage string
}

func forcedBaselineCases() []forcedBaselineCase {
	return []forcedBaselineCase{
		{
			name:      "no config at all",
			config:    "",
			wantCount: 0,
		},
		{
			name: "config disables nothing",
			config: `version: "2"
linters:
  default: none
  enable:
    - govet
`,
			wantCount: 0,
		},
		{
			name: "config omits a forced linter without disabling it",
			config: `version: "2"
linters:
  default: none
  enable:
    - govet
  disable:
    - dupl
`,
			wantCount: 0,
		},
		{
			name: "config disables one forced linter",
			config: `version: "2"
linters:
  default: standard
  disable:
    - forcetypeassert
`,
			wantCount:   1,
			wantMessage: "forcetypeassert",
		},
		{
			name: "config disables two forced linters",
			config: `version: "2"
linters:
  default: all
  disable:
    - nilerr
    - thelper
`,
			wantCount: 2,
		},
		{
			name:        "config is not valid YAML",
			config:      "linters: [this: is: broken",
			wantCount:   1,
			wantMessage: "cannot read the golangci-lint config",
		},
		{
			name:        "config makes yaml.v3 panic on a merged map key",
			config:      "version: \"2\"\nb: &b {x: 1}\nlinters: {<<: ~, *b : 1}\n",
			wantCount:   1,
			wantMessage: "cannot read the golangci-lint config",
		},
	}
}

func TestForcedBaselineFindings_ConfigDisable(t *testing.T) {
	t.Parallel()

	testCases := forcedBaselineCases()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var configPath string
			if testCase.config != "" {
				configPath = filepath.Join(t.TempDir(), ".golangci.yml")
				if err := os.WriteFile(configPath, []byte(testCase.config), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			findings := NewAnalyzer("").forcedBaselineFindings(configPath)
			if len(findings) != testCase.wantCount {
				t.Fatalf(
					"findings = %d, want %d: %+v", len(findings), testCase.wantCount, findings,
				)
			}
			if testCase.wantMessage == "" {
				return
			}
			if !strings.Contains(findings[0].Message, testCase.wantMessage) {
				t.Errorf(
					"message = %q, want it to name %q",
					findings[0].Message,
					testCase.wantMessage,
				)
			}
		})
	}
}

func TestForcedBaselineFindings_UnreadableConfig(t *testing.T) {
	t.Parallel()

	findings := NewAnalyzer("").forcedBaselineFindings(filepath.Join(t.TempDir(), "absent.yml"))
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none", findings)
	}
}
