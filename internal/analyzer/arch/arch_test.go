package arch

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func newContext(projectDir string) *pass.Context {
	return &pass.Context{ProjectDir: projectDir, Sources: newSources(projectDir)}
}

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
	ctx := newContext(projectDir)

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
		if finding.File != "domain/user.go" || finding.Line != 3 {
			t.Errorf("finding anchored at %s:%d, want the import spec at domain/user.go:3",
				finding.File, finding.Line)
		}
	}
}

func TestImportCycle_DetectsCycle(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := newContext(projectDir)

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

func TestImportCycle_MessageOrder(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := newContext(projectDir)
	want := "import cycle: domain \u2192 infra"

	for range 50 {
		findings := NewImportCycleAnalyzer().Analyze(ctx)
		if len(findings) != 1 || findings[0].Message != want {
			t.Fatalf("findings = %v, want one with message %q", findings, want)
		}
	}
}

func TestPackageBoundaryAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := newContext(projectDir)

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	if findings != nil {
		t.Errorf("expected nil findings with no go.mod, got %v", findings)
	}
}

func TestImportCycleAnalyzer_Analyze_NoGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := newContext(projectDir)

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
	ctx := newContext(projectDir)

	findings := NewPackageBoundaryAnalyzer().Analyze(ctx)

	if len(findings) != 1 {
		t.Fatalf(
			"expected exactly one finding reporting the unresolved module path, got %d",
			len(findings),
		)
	}
	if findings[0].Severity != finding.SeverityError {
		t.Errorf("expected an error-severity finding, got %v", findings[0].Severity)
	}
}

func TestImportCycle_MalformedGoMod(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeFile(t, projectDir, "go.mod", "go 1.24\n")
	writeFile(t, projectDir, "domain/user.go", "package domain\n")
	ctx := newContext(projectDir)

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) != 1 {
		t.Fatalf(
			"expected exactly one finding reporting the unresolved module path, got %d",
			len(findings),
		)
	}
	if findings[0].Severity != finding.SeverityError {
		t.Errorf("expected an error-severity finding, got %v", findings[0].Severity)
	}
}

func TestPackageBoundary_NoProjectDir(t *testing.T) {
	t.Parallel()

	findings := NewPackageBoundaryAnalyzer().Analyze(&pass.Context{})
	if findings != nil {
		t.Errorf("expected nil findings with no ProjectDir, got %v", findings)
	}
}

func TestImportCycleAnalyzer_Analyze_NoProjectDir(t *testing.T) {
	t.Parallel()

	findings := NewImportCycleAnalyzer().Analyze(&pass.Context{})
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
	ctx := newContext(projectDir)
	ctx.Sources = nil

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
	ctx := newContext(projectDir)

	findings := NewImportCycleAnalyzer().Analyze(ctx)

	if len(findings) != 1 {
		t.Fatalf(
			"expected 1 import-cycle finding from the second file of package a, got %v",
			findings,
		)
	}
}

type readCounter struct {
	source.Provider
	reads map[string]int
}

func (counter *readCounter) Read(path string) ([]byte, bool) {
	counter.reads[path]++
	return counter.Provider.Read(path)
}

func TestAnalyzers_ShareOneParse(t *testing.T) {
	t.Parallel()

	projectDir := newBoundaryAndCycleFixture(t)
	ctx := newContext(projectDir)
	counter := &readCounter{Provider: ctx.Sources, reads: make(map[string]int)}
	ctx.Sources = counter

	NewPackageBoundaryAnalyzer().Analyze(ctx)
	NewImportCycleAnalyzer().Analyze(ctx)

	if len(counter.reads) == 0 {
		t.Fatal("no source was read")
	}
	for file, reads := range counter.reads {
		if reads != 1 {
			t.Errorf("%s read %d times across both arch analyzers, want 1", file, reads)
		}
	}
}
