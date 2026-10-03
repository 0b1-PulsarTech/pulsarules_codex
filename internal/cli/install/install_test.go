package install

import (
	"bytes"
	"io"
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("close pipe writer: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, reader); copyErr != nil {
		t.Fatalf("read pipe: %v", copyErr)
	}
	return buf.String()
}

//nolint:paralleltest // captureStdout swaps os.Stdout.
func TestInstallPostTargets_EmptyGitHooks(t *testing.T) {
	dir := t.TempDir()
	opts := &cliopts.Options{NoGitHooks: false}
	hookReg := install.NewRegistry()

	output := captureStdout(t, func() {
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
