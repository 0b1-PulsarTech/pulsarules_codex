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
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"frobnicate"}, flags.Environment{})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if errors.Is(err, command.ErrFindings) {
		t.Fatalf("unknown command should not be command.ErrFindings, got %v", err)
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
			err := commands()[opts.Command](inj, &opts)
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
			call := Invocation{
				Name:   testCase.command,
				Runner: legacyRunner{name: testCase.command, opts: &version.Options{}},
			}
			got := call.HandleBootstrapErr(errors.New("boom"))
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
	t.Parallel()

	testCases := []struct {
		name string
		opts *version.Options
		want string
	}{
		{
			name: "explicit flag wins everywhere",
			opts: &version.Options{
				Command:    "governance",
				ProjectDir: "/explicit",
				Env:        flags.Environment{ProjectDir: "/from-env"},
			},
			want: "/explicit",
		},
		{
			name: "governance falls back to the env var",
			opts: &version.Options{
				Command: "governance",
				Env:     flags.Environment{ProjectDir: "/from-env"},
			},
			want: "/from-env",
		},
		{
			name: "governance with no env reports nothing rather than guessing",
			opts: &version.Options{Command: "governance"},
			want: "",
		},
		{
			name: "every other command defaults to the working directory",
			opts: &version.Options{
				Command: "list",
				Env:     flags.Environment{ProjectDir: "/from-env"},
			},
			want: ".",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveProjectDir(testCase.opts); got != testCase.want {
				t.Errorf("resolveProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestResolveProjectDir_InstallFlag(t *testing.T) {
	t.Parallel()

	opts, err := version.ParseArgs(
		[]string{"install", "--project", "/tmp/repo", "--all"},
		flags.Environment{},
	)
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if got := resolveProjectDir(opts); got != "/tmp/repo" {
		t.Errorf("resolveProjectDir() = %q, want %q", got, "/tmp/repo")
	}
}

func TestBootstrapOptions_ReadsTheEnvironmentOnce(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join("/proj", ".claude", "hook-execution.log")
	opts := &version.Options{
		Command: "hook",
		Root:    "/root",
		Env:     flags.Environment{LogPath: logPath},
	}
	got := Invocation{
		Root:   opts.Root,
		Env:    opts.Env,
		Runner: legacyRunner{name: opts.Command, opts: opts},
	}.BootstrapOptions()
	if got.LogPath != logPath || got.Root != "/root" {
		t.Errorf("BootstrapOptions() = %+v, want LogPath %q and Root /root", got, logPath)
	}
	unset := Invocation{Runner: legacyRunner{opts: &version.Options{Command: "hook"}}}
	if got = unset.BootstrapOptions(); got.LogPath != "" {
		t.Errorf("unset env LogPath = %q, want empty so logfile decides", got.LogPath)
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
	if errors.Is(err, command.ErrFindings) {
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
			if got := outcome(
				commands()[opts.Command](inj, opts),
			); !strings.HasPrefix(
				got,
				testCase.want,
			) {
				t.Errorf("Run outcome = %q, want prefix %q", got, testCase.want)
			}
		})
	}
}

func bindWithoutCatalog(t *testing.T, inj remy.Injector, dir string) {
	t.Helper()

	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
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
				call, err := Parse([]string{"commitlint", "--msg", "feat: x"}, flags.Environment{})
				if err != nil {
					return err
				}
				return call.Run(inj)
			},
		},
		{
			name: "governance",
			run: func(inj remy.Injector) error {
				return governance.RunGovernance(inj, &version.Options{ProjectDir: "."})
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
