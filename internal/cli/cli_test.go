package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := Run(inj, &cliopts.Options{Command: "frobnicate"})
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
	err := runCommitLint(inj, &cliopts.Options{CommitMsg: messageTrippingEmojiTypeAndInitialRules})

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

	err := runCommitLint(inj, &cliopts.Options{CommitMsg: "valid enough message"})
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
		&cliopts.Options{CommitMsg: invalidCommitMessage},
	)
	if _, ok := errors.AsType[*ExitError](err); !ok {
		t.Fatalf("expected *ExitError from analysis findings, got %v (%T)", err, err)
	}
}

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

func TestResolveProjectDir_InstallFlag(t *testing.T) {
	t.Parallel()

	opts, err := cliopts.ParseArgs([]string{"install", "--project", "/tmp/repo", "--all"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if got := resolveProjectDir(opts); got != "/tmp/repo" {
		t.Errorf("resolveProjectDir() = %q, want %q", got, "/tmp/repo")
	}
}

func TestResolveLogPath(t *testing.T) {
	testCases := []struct {
		name    string
		envPath string
		want    string
	}{
		{name: "unset env leaves the decision to obs", envPath: "", want: ""},
		{
			name:    "set env is returned verbatim",
			envPath: filepath.Join("/proj", ".claude", "hook-execution.log"),
			want:    filepath.Join("/proj", ".claude", "hook-execution.log"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PULSARULES_LOG_PATH", testCase.envPath)
			if got := resolveLogPath(); got != testCase.want {
				t.Errorf("resolveLogPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}
