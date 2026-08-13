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

var forcedBaselineCases = []forcedBaselineCase{
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
}

// TestForcedBaselineFindings covers the one case -E loses to the target's own config. The
// behaviour was verified against golangci-lint v2.12.2 before this check existed: a config with
// `linters: {default: standard, disable: [forcetypeassert]}` runs with forcetypeassert OFF even
// when the command line passes `-E forcetypeassert`, and reports zero findings for it - which
// reads exactly like a clean run.
func TestForcedBaselineFindings(t *testing.T) {
	t.Parallel()

	for _, testCase := range forcedBaselineCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var configPath string
			if testCase.config != "" {
				configPath = filepath.Join(t.TempDir(), ".golangci.yml")
				if err := os.WriteFile(configPath, []byte(testCase.config), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			findings := forcedBaselineFindings(configPath)
			if len(findings) != testCase.wantCount {
				t.Fatalf("findings = %d, want %d: %+v", len(findings), testCase.wantCount, findings)
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

// TestForcedBaselineFindings_UnreadableConfigIsSilent asserts a config path that does not exist
// produces nothing: "no config" means nothing is disabled, which is the case -E already wins.
func TestForcedBaselineFindings_UnreadableConfigIsSilent(t *testing.T) {
	t.Parallel()

	findings := forcedBaselineFindings(filepath.Join(t.TempDir(), "absent.yml"))
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none", findings)
	}
}
