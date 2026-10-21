package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := Run(inj, &version.Options{Command: "frobnicate"})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if errors.Is(err, ErrFindings) {
		t.Fatalf("unknown command should not be ErrFindings, got %v", err)
	}
}

func TestRunCommitLint_ReturnsErrFindings(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const messageTrippingEmojiTypeAndInitialRules = "bad commit message"
	err := runCommitLint(inj, &version.Options{CommitMsg: messageTrippingEmojiTypeAndInitialRules})

	if !errors.Is(err, ErrFindings) {
		t.Fatalf("error = %v, want ErrFindings", err)
	}
}

func TestRunCommitLint_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	wantErr := errors.New("boom: injector wiring failed")
	remy.RegisterConstructorErr(
		inj, remy.Factory[*vcs.Repository],
		func() (*vcs.Repository, error) { return nil, wantErr },
	)

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
	if !errors.Is(err, ErrFindings) {
		t.Fatalf("error = %v, want ErrFindings from analysis findings", err)
	}
}

func TestRun_PrefixesInstallAndUninstallErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       version.Options
		wantPrefix string
	}{
		{
			name:       "uninstall without a scope",
			opts:       version.Options{Command: "uninstall"},
			wantPrefix: "uninstall: ",
		},
		{
			name: "install with an unknown target",
			opts: version.Options{
				Command:   "install",
				Skills:    []string{"go-style"},
				Languages: []string{"go"},
				Target:    []string{"bogus"},
			},
			wantPrefix: "install: invalid --target",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
				t.Fatalf("DoInjections: %v", err)
			}
			opts := testCase.opts
			if opts.Command == "install" {
				opts.Project = t.TempDir()
			}
			err := Run(inj, &opts)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.HasPrefix(err.Error(), testCase.wantPrefix) {
				t.Errorf("error = %q, want prefix %q", err.Error(), testCase.wantPrefix)
			}
		})
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

func TestErrorPrinters_StaySingleLine(t *testing.T) {
	t.Parallel()

	forged := fmt.Errorf("open %s: %w", "a\n[ERROR] forged", os.ErrNotExist)
	testCases := []struct {
		name  string
		label string
		print func(out io.Writer)
	}{
		{"report error", "pulsarules_cli:", func(out io.Writer) { ReportError(out, forged) }},
		{
			"bootstrap hook",
			"pulsarules_cli: hook:",
			func(out io.Writer) { handleBootstrapErr(out, "hook", forged) },
		},
		{
			"bootstrap other",
			"pulsarules_cli:",
			func(out io.Writer) { handleBootstrapErr(out, "clean", forged) },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			testCase.print(&out)
			text := strings.TrimSuffix(out.String(), "\n")
			if strings.Contains(text, "\n") || !strings.Contains(text, `a\n[ERROR] forged`) ||
				!strings.HasPrefix(text, testCase.label+" ") {
				t.Errorf("output = %q, want one escaped line", out.String())
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
			t.Setenv("PULSARULES_PROJECT_DIR", testCase.envDir)
			if got := resolveProjectDir(testCase.opts); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestResolveProjectDir_InstallFlag(t *testing.T) {
	t.Parallel()

	opts, err := version.ParseArgs([]string{"install", "--project", "/tmp/repo", "--all"})
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

func TestDispatchCommands_MatchBoundCommands(t *testing.T) {
	t.Parallel()

	dispatched := slices.Sorted(maps.Keys(commands()))
	if bound := version.Commands(); !slices.Equal(dispatched, bound) {
		t.Errorf("Run dispatches %v, ParseArgs binds %v", dispatched, bound)
	}
}

func seedGovernanceRepo(t *testing.T, dir, source string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, "alpha.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "alpha.go"}} {
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func outcome(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, ErrFindings) {
		return "exit 1"
	}
	return err.Error()
}

func TestRun_GovernanceReportsFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		scope  string
		source string
		want   string
	}{
		{name: "clean tree", source: "package alpha\n", want: "ok"},
		{name: "warnings do not fail the run", source: "package alpha\n\nfunc {\n", want: "ok"},
		{name: "errors exit 1", source: "package alpha\n\nvar _ = \"\u2014\"\n", want: "exit 1"},
		{name: "invalid scope", scope: "fulll", want: "invalid --scope"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			seedGovernanceRepo(t, dir, testCase.source)
			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: dir}); err != nil {
				t.Fatalf("DoInjections: %v", err)
			}
			opts := &version.Options{
				Command: "governance", ProjectDir: dir, Scope: testCase.scope, AllFiles: true,
			}
			if got := outcome(Run(inj, opts)); !strings.HasPrefix(got, testCase.want) {
				t.Errorf("Run outcome = %q, want prefix %q", got, testCase.want)
			}
		})
	}
}

func bindWithoutCatalog(t *testing.T, inj remy.Injector, dir string) {
	t.Helper()

	initGitRepo(t, dir)
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	index, assets, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	remy.RegisterInstance(inj, repo)
	remy.RegisterInstance(inj, index)
	remy.RegisterInstance(inj, templates.New(assets))
	remy.RegisterInstance(inj, slog.New(slog.DiscardHandler))
}

func TestCatalogFailureFailsTheCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		run  func(inj remy.Injector) error
	}{
		{
			name: "commit-lint",
			run: func(inj remy.Injector) error {
				return runCommitLint(inj, &version.Options{CommitMsg: "feat: x"})
			},
		},
		{
			name: "governance",
			run: func(inj remy.Injector) error {
				return runGovernance(inj, &version.Options{ProjectDir: "."})
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			bindWithoutCatalog(t, inj, t.TempDir())
			err := testCase.run(inj)
			if err == nil || !strings.Contains(err.Error(), "get emoji catalog") {
				t.Fatalf("err = %v, want one naming %q", err, "get emoji catalog")
			}
		})
	}
}
