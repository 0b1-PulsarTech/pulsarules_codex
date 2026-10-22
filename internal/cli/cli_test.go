package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"frobnicate"}, flags.Environment{}, io.Discard)
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
		args       []string
		wantPrefix string
	}{
		{
			name:       "uninstall without a scope",
			args:       []string{"uninstall"},
			wantPrefix: "uninstall: ",
		},
		{
			name: "install with an unknown target",
			args: []string{
				"install", "--project", "", "--skills", "go-style", "--languages", "go",
				"--target", "bogus",
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
			args := slices.Clone(testCase.args)
			if idx := slices.Index(args, "--project"); idx >= 0 {
				args[idx+1] = t.TempDir()
			}
			call, err := Parse(args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			err = call.Run(inj)
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
		name string
		args []string
		want int
	}{
		{name: "hook degrades to success", args: []string{"hook"}, want: 0},
		{name: "governance fails loudly", args: []string{"governance"}, want: 1},
		{name: "a command without a runner fails loudly", want: 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			call, err := Parse(testCase.args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := call.HandleBootstrapErr(errors.New("boom")); got != testCase.want {
				t.Errorf("HandleBootstrapErr(%q) = %d, want %d", testCase.args, got, testCase.want)
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

func TestBootstrapOptions_ProjectDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "install anchors on its project", args: []string{"install", "--project", "/repo"},
			want: "/repo",
		},
		{
			name: "uninstall anchors on its project",
			args: []string{"uninstall", "--project", "/repo"},
			want: "/repo",
		},
		{name: "every other command uses the working directory", args: []string{"list"}, want: "."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			call, err := Parse(
				testCase.args,
				flags.Environment{ProjectDir: "/from-env"},
				io.Discard,
			)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := call.BootstrapOptions().ProjectDir; got != testCase.want {
				t.Errorf("ProjectDir = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBootstrapOptions_ReadsTheEnvironmentOnce(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join("/proj", ".claude", "hook-execution.log")
	call, err := Parse(
		[]string{"hook", "--root", "/root", "stop"},
		flags.Environment{LogPath: logPath},
		io.Discard,
	)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := call.BootstrapOptions()
	if got.LogPath != logPath || got.Root != "/root" {
		t.Errorf("BootstrapOptions() = %+v, want LogPath %q and Root /root", got, logPath)
	}
	unset, err := Parse([]string{"hook"}, flags.Environment{}, io.Discard)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got = unset.BootstrapOptions(); got.LogPath != "" {
		t.Errorf("unset env LogPath = %q, want empty so logfile decides", got.LogPath)
	}
}

func TestTable_NamesAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, candidate := range table() {
		if seen[candidate.name] {
			t.Errorf("command %q is registered twice", candidate.name)
		}
		seen[candidate.name] = true
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
			args := []string{"governance", "--project", dir, "--all-files"}
			if testCase.scope != "" {
				args = append(args, "--scope", testCase.scope)
			}
			call, err := Parse(args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := outcome(call.Run(inj)); !strings.HasPrefix(got, testCase.want) {
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
				call, err := Parse(
					[]string{"commitlint", "--msg", "feat: x"},
					flags.Environment{},
					io.Discard,
				)
				if err != nil {
					return err
				}
				return call.Run(inj)
			},
		},
		{
			name: "governance",
			run: func(inj remy.Injector) error {
				call, err := Parse(
					[]string{"governance", "--project", "."},
					flags.Environment{},
					io.Discard,
				)
				if err != nil {
					return err
				}
				return call.Run(inj)
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
