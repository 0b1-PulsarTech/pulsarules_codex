package golangcilint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

func TestRun_NoBinary(t *testing.T) {
	t.Parallel()

	r := NewRunner("")
	findings := r.Run(".", "")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings with empty path, got %d", len(findings))
	}
}

// TestRun_MissingBinary proves a binary that does not resolve on PATH (or
// as a direct path) is a clean skip - no findings at all - instead of the
// synthetic finding it used to surface before the LookPath probe.
func TestRun_MissingBinary(t *testing.T) {
	t.Parallel()

	r := NewRunner("/nonexistent/golangci-lint")
	findings := r.Run(".", "")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for a missing binary, got %d: %+v", len(findings), findings)
	}
}

// TestRun_Timeout proves a binary that overruns Runner.timeout is reported
// as a real failure (not silently skipped like a missing binary), and that
// Run returns well before the script's own sleep would finish.
func TestRun_Timeout(t *testing.T) {
	t.Parallel()

	script := writeSlowScript(t)
	r := &Runner{path: script, timeout: 50 * time.Millisecond}

	start := time.Now()
	findings := r.Run(t.TempDir(), "")
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("Run took %v, want it killed near the 50ms timeout", elapsed)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for a timed-out run, got %d: %+v", len(findings), findings)
	}
	if findings[0].AnalyzerID != "golangci-lint" {
		t.Errorf("AnalyzerID = %q, want %q", findings[0].AnalyzerID, "golangci-lint")
	}
}

// writeSlowScript writes an executable shell script that outlives any
// timeout this package tests with, so Run's context cancellation is what
// actually stops it.
func writeSlowScript(t *testing.T) string {
	t.Helper()

	// exec replaces the shell instead of forking sleep as its child, so
	// killing this one process is enough - no grandchild can outlive it
	// and hold the captured stdout/stderr pipe open past ioWaitGrace.
	path := filepath.Join(t.TempDir(), "slow-golangci-lint")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 5\n"), 0o700); err != nil {
		t.Fatalf("write slow script: %v", err)
	}
	return path
}

func TestLintArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		target     string
		configFlag string
		wantConfig bool
	}{
		{
			name:   "no config discovered",
			target: "./...",
		},
		{
			name:       "config discovered",
			target:     "./...",
			configFlag: "/some/project/.golangci.yml",
			wantConfig: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			args := lintArgs(testCase.target, testCase.configFlag)

			if args[0] != "run" {
				t.Fatalf("args[0] = %q, want %q", args[0], "run")
			}
			if args[len(args)-1] != testCase.target {
				t.Errorf("last arg = %q, want target %q", args[len(args)-1], testCase.target)
			}
			for _, linter := range forcedLinters {
				if !containsPair(args, "-E", linter) {
					t.Errorf("args %v missing forced -E %s", args, linter)
				}
			}
			hasConfig := containsPair(args, "--config", testCase.configFlag)
			if hasConfig != testCase.wantConfig {
				t.Errorf(
					"--config %s present = %v, want %v",
					testCase.configFlag,
					hasConfig,
					testCase.wantConfig,
				)
			}
		})
	}
}

// containsPair reports whether args holds flag immediately followed by value.
func containsPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func TestParseGoWorkModules(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := `go 1.22.0

use ./internal/foo

use ./cmd/bar

replace example.com/old => ../local/new
`
	if err := os.WriteFile(filepath.Join(dir, "go.work"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	modules := parseGoWorkModules(dir)
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d: %v", len(modules), modules)
	}
	if modules[0] != "./internal/foo" {
		t.Errorf("module[0] = %q, want ./internal/foo", modules[0])
	}
	if modules[1] != "./cmd/bar" {
		t.Errorf("module[1] = %q, want ./cmd/bar", modules[1])
	}
}

func TestParseGoWorkModules_MissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modules := parseGoWorkModules(dir)
	if modules != nil {
		t.Errorf("expected nil for missing go.work, got %v", modules)
	}
}

func TestResolvedConfigFlag_Explicit(t *testing.T) {
	t.Parallel()

	result := resolvedConfigFlag("/some/project", "/explicit/path/.golangci.yml")
	if result != "/explicit/path/.golangci.yml" {
		t.Errorf("expected explicit path, got %q", result)
	}
}

func TestResolvedConfigFlag_AutoDiscovery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, ".golangci.yml"),
		[]byte("linters:\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	result := resolvedConfigFlag(dir, "")
	if result != filepath.Join(dir, ".golangci.yml") {
		t.Errorf("expected %q, got %q", filepath.Join(dir, ".golangci.yml"), result)
	}
}

func TestResolvedConfigFlag_ToolsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(toolsDir, ".golangci.yml"),
		[]byte("linters:\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	result := resolvedConfigFlag(dir, "")
	if result != filepath.Join(toolsDir, ".golangci.yml") {
		t.Errorf("expected %q, got %q", filepath.Join(toolsDir, ".golangci.yml"), result)
	}
}

func TestResolvedConfigFlag_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result := resolvedConfigFlag(dir, "")
	if result != "" {
		t.Errorf("expected empty when no config found, got %q", result)
	}
}

func TestParseOutput_Raw(t *testing.T) {
	t.Parallel()

	out := `{"Issues":[{"FromLinter":"errcheck","Text":"unchecked error","Severity":"warning","Pos":{"Filename":"foo.go","Line":42}}]}`
	findings := parseOutput(execx.Result{Stdout: out}, nil)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].AnalyzerID != "golangci-lint/errcheck" {
		t.Errorf("AnalyzerID = %q", findings[0].AnalyzerID)
	}
	if findings[0].File != "foo.go" {
		t.Errorf("File = %q", findings[0].File)
	}
	if findings[0].Line != 42 {
		t.Errorf("Line = %d", findings[0].Line)
	}
}

func TestIsLintExit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		code int
		want bool
	}{
		{0, false},
		{1, true},
		{7, true},
		{3, false},
		{4, false},
		{5, false},
	}
	for _, testCase := range testCases {
		if got := isLintExit(testCase.code); got != testCase.want {
			t.Errorf("isLintExit(%d) = %v, want %v", testCase.code, got, testCase.want)
		}
	}
}

// TestEscapesProject pins the filter that stopped a stale golangci-lint cache from
// reporting files in a deleted worktree as real findings - it poisoned four separate
// measurements in this repo before anything caught it.
func TestEscapesProject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		want bool
	}{
		{"inside the project", "internal/analyzer/arch/imports.go", false},
		{"dot-slash prefixed", "./internal/foo.go", false},
		{"climbs out once", "../other/foo.go", true},
		{"climbs out many times", "../../../../tmp/wt-final/internal/foo.go", true},
		{"climbs out then back in", "../project/internal/foo.go", true},
		{"normalises to inside", "internal/../internal/foo.go", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := escapesProject(testCase.path); got != testCase.want {
				t.Errorf("escapesProject(%q) = %v, want %v", testCase.path, got, testCase.want)
			}
		})
	}
}
