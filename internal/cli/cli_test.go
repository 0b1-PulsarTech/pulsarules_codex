package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := Run(inj, &version.Options{Command: "frobnicate"})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if _, ok := errors.AsType[*ExitError](err); ok {
		t.Fatalf("unknown command should not be an *ExitError, got %v", err)
	}
}

func TestRunCommitLint_ReturnsExitError(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const messageTrippingEmojiTypeAndInitialRules = "bad commit message"
	err := runCommitLint(inj, &version.Options{CommitMsg: messageTrippingEmojiTypeAndInitialRules})

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %v (%T)", err, err)
	}
	if exitErr.Code != 1 {
		t.Fatalf("Code = %d, want 1", exitErr.Code)
	}
}

func TestRunCommitLint_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	wantErr := errors.New("boom: injector wiring failed")
	remy.RegisterConstructorErr(inj, remy.Factory[vcs.Repository], func() (vcs.Repository, error) {
		return nil, wantErr
	})

	err := runCommitLint(inj, &version.Options{CommitMsg: "valid enough message"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("runCommitLint() error = %v, want wrapping %v", err, wantErr)
	}
}

func TestRunCommitLint_NoRepository(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: t.TempDir()}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const invalidCommitMessage = "bad commit message"
	err := runCommitLint(
		inj,
		&version.Options{CommitMsg: invalidCommitMessage},
	)
	if _, ok := errors.AsType[*ExitError](err); !ok {
		t.Fatalf("expected *ExitError from analysis findings, got %v (%T)", err, err)
	}
}

func TestExitError_Error(t *testing.T) {
	t.Parallel()

	if got := (&ExitError{Code: 2}).Error(); got != "exit code 2" {
		t.Errorf("Error() = %q, want %q", got, "exit code 2")
	}
}

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

func TestResolveProjectDir(t *testing.T) {
	testCases := []struct {
		name   string
		opts   *version.Options
		envDir string
		want   string
	}{
		{
			name: "explicit flag wins everywhere",
			opts: &version.Options{Command: "governance", ProjectDir: "/explicit"},
			want: "/explicit",
		},
		{
			name:   "governance falls back to the env var",
			opts:   &version.Options{Command: "governance"},
			envDir: "/from-env",
			want:   "/from-env",
		},
		{
			name: "governance with no env reports nothing rather than guessing",
			opts: &version.Options{Command: "governance"},
			want: "",
		},
		{
			name:   "every other command defaults to the working directory",
			opts:   &version.Options{Command: "list"},
			envDir: "/from-env",
			want:   ".",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("CLAUDE_PROJECT_DIR", testCase.envDir)
			if got := resolveProjectDir(testCase.opts); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestResolveLogPath(t *testing.T) {
	t.Run("unset env leaves the path to obs", func(t *testing.T) {
		t.Setenv("CLAUDE_PROJECT_DIR", "")
		if got := resolveLogPath(); got != "" {
			t.Errorf("resolveLogPath() = %q, want empty", got)
		}
	})
	t.Run("set env lands under .claude", func(t *testing.T) {
		t.Setenv("CLAUDE_PROJECT_DIR", "/proj")
		want := filepath.Join("/proj", ".claude", "hook-execution.log")
		if got := resolveLogPath(); got != want {
			t.Errorf("resolveLogPath() = %q, want %q", got, want)
		}
	})
}
