package governance

import (
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestGolden(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
	}{
		{name: "clean package produces no findings", dir: "clean"},
		{name: "single violation pinned to its file and line", dir: "violation"},
		{name: "generated file suppressed but counted", dir: "generated"},
		{
			name: "AST analyzers reach _test.go files, not just production code",
			dir:  "test-file-blindness",
		},
		{
			name: "markdown reaches the static analyzers, not just Go",
			dir:  "markers",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			const goldenCasesDir = "testdata/golden"
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
