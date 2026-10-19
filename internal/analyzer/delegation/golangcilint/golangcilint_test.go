package golangcilint

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func TestRun_NoBinary(t *testing.T) {
	t.Parallel()

	findings := NewAnalyzer("").Analyze(&pass.Context{ProjectDir: "."})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings with empty path, got %d", len(findings))
	}
}

func TestRun_MissingBinary(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer("/nonexistent/golangci-lint")
	findings := analyzer.Analyze(&pass.Context{ProjectDir: "."})
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for a missing binary, got %d: %+v", len(findings), findings)
	}
}

func TestRun_TimeoutIsARealFailureNotASilentSkip(t *testing.T) {
	t.Parallel()

	script := writeExecSleepScriptNoGrandchild(t)
	analyzer := NewAnalyzer(script)
	analyzer.timeout = 50 * time.Millisecond

	start := time.Now()
	findings := analyzer.Analyze(&pass.Context{ProjectDir: t.TempDir()})
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
	if want := "golangci-lint failed"; findings[0].Message != want {
		t.Errorf("Message = %q, want %q", findings[0].Message, want)
	}
}

func TestRun_BoundsOutput(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "noisy-golangci-lint")
	script := fmt.Sprintf("#!/bin/sh\nhead -c %d /dev/zero\n", command.MaxOutputBytes+1)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write noisy script: %v", err)
	}

	findings := NewAnalyzer(path).Analyze(&pass.Context{ProjectDir: t.TempDir()})
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	want := fmt.Sprintf("golangci-lint failed: output exceeded %d bytes", command.MaxOutputBytes)
	if findings[0].Message != want {
		t.Errorf("Message = %q, want %q", findings[0].Message, want)
	}
}

func writeExecSleepScriptNoGrandchild(t *testing.T) string {
	t.Helper()

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
			for _, linter := range forcedLinters() {
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

func containsPair(args []string, flag, value string) bool {
	for pos := 0; pos < len(args)-1; pos++ {
		if args[pos] == flag && args[pos+1] == value {
			return true
		}
	}
	return false
}

func TestParseGoWorkModules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		write   bool
		want    []string
	}{
		{
			name:    "single lines",
			content: "go 1.22.0\n\nuse ./internal/foo\n\nuse ./cmd/bar\n\nreplace example.com/old => ../local/new\n",
			write:   true,
			want:    []string{"./internal/foo", "./cmd/bar"},
		},
		{
			name:    "block",
			content: "go 1.22.0\n\nuse (\n\t./a // first\n\t\"./b\"\n)\n",
			write:   true,
			want:    []string{"./a", "./b"},
		},
		{name: "missing file"},
		{name: "malformed falls back to nil", content: "bogus directive\n", write: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if testCase.write {
				path := filepath.Join(dir, "go.work")
				if err := os.WriteFile(path, []byte(testCase.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := parseGoWorkModules(dir); !slices.Equal(got, testCase.want) {
				t.Fatalf("parseGoWorkModules() = %q, want %q", got, testCase.want)
			}
		})
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

	out := `{"Issues": [{
  "FromLinter": "errcheck", "Text": "unchecked error", "Severity": "warning",
  "Pos": {"Filename": "foo.go", "Line": 42}
}]}`
	findings := NewAnalyzer("").parseOutput(process.Result{Stdout: out}, nil)
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

func TestEscapesProject_StaleCacheEntries(t *testing.T) {
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
