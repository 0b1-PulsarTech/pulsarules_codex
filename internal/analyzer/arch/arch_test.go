package arch

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// newBoundaryAndCycleFixture writes a throwaway Go project under a fresh
// t.TempDir(): its own go.mod declaring a module path OTHER than this
// tool's, a domain package that imports an infra package (a boundary
// violation: an inner layer depending on an outer one) and an infra package
// that imports the domain package back (closing a genuine two-node import
// cycle). It returns the project root.
func newBoundaryAndCycleFixture(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	writeFile(t, tmp, "go.mod", "module example.com/target\n\ngo 1.24\n")
	writeFile(t, tmp, "domain/user.go", `package domain

import _ "example.com/target/infra"

type User struct{}
`)
	writeFile(t, tmp, "infra/db.go", `package infra

import _ "example.com/target/domain"

type DB struct{}
`)
	return tmp
}

func TestPackageBoundaryAnalyzer_Analyze_DetectsRealBoundaryViolation(t *testing.T) {
	t.Parallel()

	tmp := newBoundaryAndCycleFixture(t)
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	if len(findings) == 0 {
		t.Fatal(
			"expected at least one arch-boundary finding for domain importing infra; " +
				"got none - the analyzer likely resolved the wrong module path and " +
				"filtered every import out of the graph",
		)
	}
	for _, f := range findings {
		if f.AnalyzerID != "arch-boundary" {
			t.Errorf("finding has AnalyzerID %q, want %q", f.AnalyzerID, "arch-boundary")
		}
	}
}

func TestImportCycleAnalyzer_Analyze_DetectsRealCycle(t *testing.T) {
	t.Parallel()

	tmp := newBoundaryAndCycleFixture(t)
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) == 0 {
		t.Fatal(
			"expected at least one import-cycle finding for domain<->infra; " +
				"got none - the analyzer likely resolved the wrong module path and " +
				"filtered every import out of the graph",
		)
	}
	for _, f := range findings {
		if f.AnalyzerID != "import-cycle" {
			t.Errorf("finding has AnalyzerID %q, want %q", f.AnalyzerID, "import-cycle")
		}
	}
}

func TestPackageBoundaryAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	// why: a target with no go.mod at all is "not a Go project here", the
	// same convention the pre-search hook uses (internal/hook/emit_turn.go)
	// and the one this repo's golden/empty-repo fixtures already rely on -
	// it is not the broken-environment case this analyzer must report.
	if findings != nil {
		t.Errorf("expected nil findings with no go.mod, got %v", findings)
	}
}

func TestImportCycleAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if findings != nil {
		t.Errorf("expected nil findings with no go.mod, got %v", findings)
	}
}

func TestPackageBoundaryAnalyzer_Analyze_MalformedGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "go.mod", "go 1.24\n") // present, but no "module" line
	writeFile(t, tmp, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	// why: a go.mod that EXISTS but is unusable is a genuinely broken
	// environment - the analyzer cannot tell whether the project has
	// violations, so silently reporting zero is exactly the defect this fix
	// closes. It must say so instead.
	if len(findings) != 1 {
		t.Fatalf(
			"expected exactly one finding reporting the unresolved module path, got %d",
			len(findings),
		)
	}
	if findings[0].Severity != core.SeverityError {
		t.Errorf("expected an error-severity finding, got %v", findings[0].Severity)
	}
}

func TestImportCycleAnalyzer_Analyze_MalformedGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "go.mod", "go 1.24\n") // present, but no "module" line
	writeFile(t, tmp, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: tmp}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) != 1 {
		t.Fatalf(
			"expected exactly one finding reporting the unresolved module path, got %d",
			len(findings),
		)
	}
	if findings[0].Severity != core.SeverityError {
		t.Errorf("expected an error-severity finding, got %v", findings[0].Severity)
	}
}

func TestPackageBoundaryAnalyzer_Analyze_NoProjectDir(t *testing.T) {
	t.Parallel()

	findings := NewPackageBoundaryAnalyzer().Analyze(&core.AnalysisContext{})
	if findings != nil {
		t.Errorf("expected nil findings with no ProjectDir, got %v", findings)
	}
}

func TestImportCycleAnalyzer_Analyze_NoProjectDir(t *testing.T) {
	t.Parallel()

	findings := NewImportCycleAnalyzer().Analyze(&core.AnalysisContext{})
	if findings != nil {
		t.Errorf("expected nil findings with no ProjectDir, got %v", findings)
	}
}

// TestArchAnalyzers_MetadataAndSelfAnalysis is a light sanity check that both
// analyzers still find themselves architecturally clean and identify
// correctly; the fixture-based tests above cover the real regression.
func TestArchAnalyzers_MetadataAndSelfAnalysis(t *testing.T) {
	t.Parallel()

	if id := NewPackageBoundaryAnalyzer().ID(); id != "arch-boundary" {
		t.Errorf("ID() = %q, want arch-boundary", id)
	}
	if id := NewImportCycleAnalyzer().ID(); id != "import-cycle" {
		t.Errorf("ID() = %q, want import-cycle", id)
	}
}
