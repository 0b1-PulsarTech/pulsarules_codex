package install

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

// TestRun_MalformedMCPJSON_ErrorNotDoubled reproduces a malformed .mcp.json:
// target.Registry.Install already wraps the target Strategy's error as
// `install target %q: %w` (internal/skill/target/registry.go), so Run must
// not wrap it again with the identical format, which would read as
// `install target "claude": install target "claude": write mcp: ...`.
func TestRun_MalformedMCPJSON_ErrorNotDoubled(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(
		projectDir+"/.mcp.json", []byte("{not valid json"), 0o600,
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

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. No other test in this package writes to
// os.Stdout, so the swap is safe alongside t.Parallel().
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

// TestInstallPostTargets_EmptyGitHooksPrintsNoSuccessLine reproduces an
// explicit `--git-hooks ""` (without --no-git-hooks): resolveGitHooks parses
// it into a non-nil, empty hook list, so githook.Install writes nothing, yet
// the caller must not print "installed git hooks: " with an empty joined
// string claiming a success that never happened.
func TestInstallPostTargets_EmptyGitHooksPrintsNoSuccessLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	opts := &cliopts.Options{NoGitHooks: false}
	hookReg := install.NewRegistry()

	output := captureStdout(t, func() {
		if err := installPostTargets(opts, dir, nil, hookReg, []string{}); err != nil {
			t.Fatalf("installPostTargets: %v", err)
		}
	})

	if bytes.Contains([]byte(output), []byte("installed git hooks:")) {
		t.Errorf("output = %q, must not print a git-hooks success line for an empty list", output)
	}
}
