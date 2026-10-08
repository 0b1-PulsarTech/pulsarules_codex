package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func lintInRepo(t *testing.T, configYAML, message string) error {
	t.Helper()

	dir := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if configYAML != "" {
		path := filepath.Join(dir, projectfile.Name)
		if err := os.WriteFile(path, []byte(configYAML), disk.File); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: dir}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	return runCommitLint(inj, &version.Options{CommitMsg: message})
}

func TestRunCommitLint_ProjectConfig(t *testing.T) {
	t.Parallel()

	const (
		goodMessage = ":memo: docs(cli): Document the config flag"
		badMessage  = "bad commit message"
		downgrade   = "version: 1\nanalyzers: {commit-lint: {severity: info}}\n"
	)
	testCases := []struct {
		name    string
		config  string
		message string
		wantErr bool
	}{
		{"no file, valid message", "", goodMessage, false},
		{"no file, bad message", "", badMessage, true},
		{"valid file, valid message", "version: 1\n", goodMessage, false},
		{"valid file downgrades the message errors", downgrade, badMessage, false},
		{"invalid file blocks a valid message", "version: 1\nbogus: 1\n", goodMessage, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := lintInRepo(t, testCase.config, testCase.message)
			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("runCommitLint() error = %v, want nil", err)
				}
				return
			}
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 1 {
				t.Fatalf("runCommitLint() error = %v, want exit code 1", err)
			}
		})
	}
}
