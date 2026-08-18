package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
)

// TestRun_UnknownCommand could not be written before this package existed:
// dispatching a command used to require building cmd/pulsarules_cli into a
// binary and running it as a subprocess to observe the error.
func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := Run(inj, &cliopts.Options{Command: "frobnicate"})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		t.Fatalf("unknown command should not be an *ExitError, got %v", err)
	}
}

// TestRunCommitLint_ReturnsExitError proves runGovernance/runCommitLint no
// longer call os.Exit directly: an error-severity finding now comes back as
// a plain *ExitError value the caller can assert on, instead of killing the
// test process. That is the justification for moving os.Exit's two call
// sites out of a library package and behind ExitError.
func TestRunCommitLint_ReturnsExitError(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	// "bad commit message" trips the commit-emoji/commit-type/commit-initial
	// rules, all SeverityError, with no git repo or fixture files needed.
	err := runCommitLint(inj, &cliopts.Options{CommitMsg: "bad commit message"})

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %v (%T)", err, err)
	}
	if exitErr.Code != 1 {
		t.Fatalf("Code = %d, want 1", exitErr.Code)
	}
}

// TestRun_UninstallDispatch proves Run wires "uninstall" to uninstall.Run
// (not just "install"): a missing --global/--project fails through the same
// BaseDir validation install uses, wrapped with the "uninstall:" prefix.
func TestRun_UninstallDispatch(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	err := Run(inj, &cliopts.Options{Command: "uninstall"})
	if err == nil {
		t.Fatal("expected an error: uninstall requires --global or --project")
	}
	if !strings.HasPrefix(err.Error(), "uninstall:") {
		t.Errorf("error should carry the uninstall: prefix, got %v", err)
	}
}

func TestExitError_Error(t *testing.T) {
	t.Parallel()

	if got := (&ExitError{Code: 2}).Error(); got != "exit code 2" {
		t.Errorf("Error() = %q, want %q", got, "exit code 2")
	}
}

// TestHandleBootstrapErr pins the one asymmetry the hook contract depends on:
// a hook must never block the agent's turn, so a bootstrap failure there exits
// 0 while every other command exits 1.
func TestHandleBootstrapErr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		command string
		want    int
	}{
		{name: "hook degrades to success", command: "hook", want: 0},
		{name: "governance fails loudly", command: "governance", want: 1},
		{name: "unknown command fails loudly", command: "", want: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := HandleBootstrapErr(testCase.command, errors.New("boom"))
			if got != testCase.want {
				t.Errorf(
					"HandleBootstrapErr(%q) = %d, want %d",
					testCase.command,
					got,
					testCase.want,
				)
			}
		})
	}
}

// TestResolveProjectDir covers the fallback that differs per command. It sets
// an environment variable, so it cannot run in parallel.
func TestResolveProjectDir(t *testing.T) {
	testCases := []struct {
		name   string
		opts   *cliopts.Options
		envDir string
		want   string
	}{
		{
			name: "explicit flag wins everywhere",
			opts: &cliopts.Options{Command: "governance", ProjectDir: "/explicit"},
			want: "/explicit",
		},
		{
			name:   "governance falls back to the env var",
			opts:   &cliopts.Options{Command: "governance"},
			envDir: "/from-env",
			want:   "/from-env",
		},
		{
			name: "governance with no env reports nothing rather than guessing",
			opts: &cliopts.Options{Command: "governance"},
			want: "",
		},
		{
			name:   "every other command defaults to the working directory",
			opts:   &cliopts.Options{Command: "list"},
			envDir: "/from-env",
			want:   ".",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_PROJECT_DIR", testCase.envDir)
			if got := resolveProjectDir(testCase.opts); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestResolveLogPath(t *testing.T) {
	t.Run("unset env leaves the path to obs", func(t *testing.T) {
		t.Setenv("PULSARULES_PROJECT_DIR", "")
		if got := resolveLogPath(); got != "" {
			t.Errorf("resolveLogPath() = %q, want empty", got)
		}
	})
	t.Run("set env lands under .claude", func(t *testing.T) {
		t.Setenv("PULSARULES_PROJECT_DIR", "/proj")
		want := filepath.Join("/proj", ".claude", "hook-execution.log")
		if got := resolveLogPath(); got != want {
			t.Errorf("resolveLogPath() = %q, want %q", got, want)
		}
	})
}
