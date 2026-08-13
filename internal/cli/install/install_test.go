package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
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
	opts := &cliopts.Options{Command: "install", Project: projectDir, Skills: "go-style"}

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
