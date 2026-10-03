package arch

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func newBoundaryAndCycleFixture(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "go.mod", "module example.com/target\n\ngo 1.24\n")
	writeFile(t, projectDir, "domain/user.go", `package domain

import _ "example.com/target/infra"

type User struct{}
`)
	writeFile(t, projectDir, "infra/db.go", `package infra

import _ "example.com/target/domain"

type DB struct{}
`)
	return projectDir
}

func TestPackageBoundary_Violation(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	if len(findings) == 0 {
		t.Fatal(
			"expected at least one arch-boundary finding for domain importing infra; " +
				"got none - the analyzer likely resolved the wrong module path and " +
				"filtered every import out of the graph",
		)
	}
	for _, finding := range findings {
		if finding.AnalyzerID != "arch-boundary" {
			t.Errorf("finding has AnalyzerID %q, want %q", finding.AnalyzerID, "arch-boundary")
		}
	}
}

func TestImportCycle_DetectsCycle(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) == 0 {
		t.Fatal(
			"expected at least one import-cycle finding for domain<->infra; " +
				"got none - the analyzer likely resolved the wrong module path and " +
				"filtered every import out of the graph",
		)
	}
	for _, finding := range findings {
		if finding.AnalyzerID != "import-cycle" {
			t.Errorf("finding has AnalyzerID %q, want %q", finding.AnalyzerID, "import-cycle")
		}
	}
}

func TestPackageBoundaryAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	if findings != nil {
		t.Errorf("expected nil findings with no go.mod, got %v", findings)
	}
}

func TestImportCycleAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if findings != nil {
		t.Errorf("expected nil findings with no go.mod, got %v", findings)
	}
}

func TestPackageBoundary_MalformedGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "go.mod", "go 1.24\n")
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

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

func TestImportCycle_MalformedGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "go.mod", "go 1.24\n")
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

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

func TestPackageBoundary_NoProjectDir(t *testing.T) {
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

func TestArchAnalyzers_ReportTheirRegisteredID(t *testing.T) {
	t.Parallel()

	if id := NewPackageBoundaryAnalyzer().ID(); id != "arch-boundary" {
		t.Errorf("ID() = %q, want arch-boundary", id)
	}
	if id := NewImportCycleAnalyzer().ID(); id != "import-cycle" {
		t.Errorf("ID() = %q, want import-cycle", id)
	}
}

func TestAnalyzers_Analyze_NilSources(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := &core.AnalysisContext{ProjectDir: projectDir}

	if findings := NewPackageBoundaryAnalyzer().Analyze(ctx); findings != nil {
		t.Errorf("arch-boundary: expected nil findings without Sources, got %v", findings)
	}
	if findings := NewImportCycleAnalyzer().Analyze(ctx); findings != nil {
		t.Errorf("import-cycle: expected nil findings without Sources, got %v", findings)
	}
}

func TestImportCycle_LaterFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "go.mod", "module example.com/target\n\ngo 1.24\n")
	writeFile(t, projectDir, "a/a1.go", "package a\n")
	writeFile(t, projectDir, "a/a2.go", "package a\n\nimport _ \"example.com/target/b\"\n")
	writeFile(t, projectDir, "b/b.go", "package b\n\nimport _ \"example.com/target/a\"\n")
	ctx := &core.AnalysisContext{ProjectDir: projectDir, Sources: newSources(projectDir)}

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) != 1 {
		t.Fatalf(
			"expected 1 import-cycle finding from the second file of package a, got %v",
			findings,
		)
	}
}
