package analysis

import (
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

// goldenCasesDir holds one subdirectory per golden case: a plain file tree
// (no real .git, no go.mod needed) plus an expect.json.
const goldenCasesDir = "testdata/golden"

// TestGolden runs the analyzer pipeline over each fixture in
// testdata/golden and compares the findings against that case's
// expect.json. It is the false-positive guard this repo's analyzer suite
// requires: "clean" proves a realistic multi-file package produces zero
// findings, "violation" pins one finding to its exact file and line, and
// "generated" proves the suppression pass hides a finding in a
// generated-marked file while still counting it.
func TestGolden(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
	}{
		{name: "clean package produces no findings", dir: "clean"},
		{name: "single violation pinned to its file and line", dir: "violation"},
		{name: "generated file suppressed but counted", dir: "generated"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			caseDir := filepath.Join(goldenCasesDir, testCase.dir)
			want := loadExpectation(t, filepath.Join(caseDir, "expect.json"))

			repoDir := t.TempDir()
			copyTree(t, caseDir, repoDir)
			repo := &goldenRepo{root: repoDir}

			cfg := config.Defaults()
			cfg.ApplyPreset()
			result := NewSession(repo, "", nil, cfg).Analyze(ScopeChanged, nil, FileSetAll)

			assertFindings(t, testCase.dir, result.Findings, want.Findings)
			if result.SuppressedGenerated != want.SuppressedGenerated {
				t.Errorf(
					"case %s: SuppressedGenerated = %d, want %d",
					testCase.dir, result.SuppressedGenerated, want.SuppressedGenerated,
				)
			}

			if len(want.IncludeGeneratedFindings) == 0 {
				return
			}
			includeCfg := config.Defaults()
			includeCfg.ApplyPreset()
			includeCfg.IncludeGenerated = true
			withGenerated := NewSession(repo, "", nil, includeCfg).
				Analyze(ScopeChanged, nil, FileSetAll)
			assertFindings(t, testCase.dir+"/include-generated",
				withGenerated.Findings, want.IncludeGeneratedFindings)
		})
	}
}
