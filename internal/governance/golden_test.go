package governance

import (
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func goldenScope(changedPaths []string) (*vcs.Status, FileSet) {
	if len(changedPaths) == 0 {
		return nil, FileSetAll
	}
	var changes []vcs.Change
	for _, path := range changedPaths {
		changes = append(changes, vcs.Change{Path: path, Extension: filepath.Ext(path)})
	}
	return &vcs.Status{Changes: changes}, FileSetChanged
}

func TestGolden(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		dir          string
		changedPaths []string
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
		{
			name:         "files under a skipped dir are not analyzed",
			dir:          "skipped-dir",
			changedPaths: []string{"pkg/testdata/clock.go", "pkg/clock.go"},
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
			status, files := goldenScope(testCase.changedPaths)
			result := NewSession(
				repo,
				nil,
				testCatalog(t),
				cfg,
			).Analyze(Request{Scope: ScopeChanged, Status: status, Files: files})

			assertFindings(t, testCase.dir, result.Findings, want.Findings)
			if result.Suppressed != want.SuppressedGenerated {
				t.Errorf(
					"case %s: SuppressedGenerated = %d, want %d",
					testCase.dir, result.Suppressed, want.SuppressedGenerated,
				)
			}

			if len(want.IncludeGeneratedFindings) == 0 {
				return
			}
			includeCfg := config.Defaults()
			includeCfg.ApplyPreset()
			includeCfg.IncludeGenerated = true
			withGenerated := NewSession(repo, nil, testCatalog(t), includeCfg).
				Analyze(Request{Scope: ScopeChanged, Status: status, Files: files})
			assertFindings(t, testCase.dir+"/include-generated",
				withGenerated.Findings, want.IncludeGeneratedFindings)
		})
	}
}
