package install

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

//nolint:paralleltest // stubGoplsSoClaudeMCPIsWired needs t.Setenv, which bars t.Parallel.
func TestRun_MalformedMCPJSON_ErrorNotDoubled(t *testing.T) {
	stubGoplsSoClaudeMCPIsWired(t)

	projectDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectDir, ".mcp.json"), []byte("{not valid json"), 0o600,
	); err != nil {
		t.Fatalf("seed malformed .mcp.json: %v", err)
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	const forcedLanguage = "go"
	opts := &cliopts.Options{
		Command:   "install",
		Project:   projectDir,
		Skills:    "go-style",
		Languages: forcedLanguage,
	}

	err := Run(inj, opts)
	if err == nil {
		t.Fatal("expected an error from a malformed .mcp.json")
	}
	if strings.Count(err.Error(), `install target "claude"`) > 1 {
		t.Errorf("error = %q, wraps `install target %q` more than once", err.Error(), "claude")
	}
}

func stubGoplsSoClaudeMCPIsWired(tb testing.TB) {
	tb.Helper()

	dir := tb.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "gopls"), []byte("#!/bin/sh\nexit 0\n"), 0o700,
	); err != nil {
		tb.Fatalf("write gopls stub: %v", err)
	}
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func captureStreams(tb testing.TB, fn func()) (stdout, stderr string) {
	tb.Helper()
	originalOut, originalErr := os.Stdout, os.Stderr
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout, os.Stderr = outWriter, errWriter
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()

	fn()

	for _, writer := range []*os.File{outWriter, errWriter} {
		if closeErr := writer.Close(); closeErr != nil {
			tb.Fatalf("close pipe writer: %v", closeErr)
		}
	}
	var outBuf, errBuf bytes.Buffer
	if _, copyErr := io.Copy(&outBuf, outReader); copyErr != nil {
		tb.Fatalf("read stdout pipe: %v", copyErr)
	}
	if _, copyErr := io.Copy(&errBuf, errReader); copyErr != nil {
		tb.Fatalf("read stderr pipe: %v", copyErr)
	}
	return outBuf.String(), errBuf.String()
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestInstallPostTargets_EmptyGitHooks(t *testing.T) {
	dir := t.TempDir()
	opts := &cliopts.Options{NoGitHooks: false}
	hookReg := install.NewRegistry()

	output, _ := captureStreams(t, func() {
		if err := installPostTargets(
			opts,
			installConfig{absBase: dir, gitHooks: []string{}},
			installCollaborators{hooks: hookReg},
		); err != nil {
			t.Fatalf("installPostTargets: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("installed git hooks:")) {
		t.Errorf("output = %q, must not print a git-hooks success line for an empty list", output)
	}
}

func injectRun(tb testing.TB, inj remy.Injector) {
	tb.Helper()
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		tb.Fatalf("DoInjections: %v", err)
	}
}

//nolint:paralleltest // captureStreams swaps os.Stdout and os.Stderr.
func TestRun_PrintHooks(t *testing.T) {
	projectDir := t.TempDir()
	opts := &cliopts.Options{Command: "install", Project: projectDir, PrintHooks: true}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	injectRun(t, inj)
	var runErr error
	stdout, _ := captureStreams(t, func() { runErr = Run(inj, opts) })
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	if !strings.Contains(stdout, `"hooks"`) {
		t.Errorf("stdout = %q, want the hooks block", stdout)
	}
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		t.Fatalf("read project dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("--print-hooks wrote %d entries, want none", len(entries))
	}
}

func TestRun_RejectsAnUnknownTargetBeforeWriting(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	opts := &cliopts.Options{
		Command:   "install",
		Project:   projectDir,
		Skills:    "go-style",
		Languages: "go",
		Target:    []string{"bogus"},
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	injectRun(t, inj)
	err := Run(inj, opts)
	if err == nil || !strings.Contains(err.Error(), `invalid --target "bogus" (want `) {
		t.Fatalf("err = %v, want an invalid --target error", err)
	}
	_, statErr := os.Stat(filepath.Join(projectDir, ".claude"))
	if !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("rejected install created .claude, stat err = %v", statErr)
	}
}
