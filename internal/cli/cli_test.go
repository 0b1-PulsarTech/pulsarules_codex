package cli

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
				Skills:    "go-style",
				Languages: "go",
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

func switchCaseCommands(t *testing.T, path, funcName string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var commands []string
	ast.Inspect(file, func(node ast.Node) bool {
		fn, ok := node.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName {
			return true
		}
		commands = caseLabelsInFirstSwitch(fn.Body)
		return false
	})
	if commands == nil {
		t.Fatalf("no switch statement found in %s.%s", path, funcName)
	}
	return commands
}

func caseLabelsInFirstSwitch(body *ast.BlockStmt) []string {
	var commands []string
	ast.Inspect(body, func(node ast.Node) bool {
		sw, ok := node.(*ast.SwitchStmt)
		if !ok {
			return commands == nil
		}
		for _, stmt := range sw.Body.List {
			clause, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, expr := range clause.List {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if unquoted, unquoteErr := strconv.Unquote(lit.Value); unquoteErr == nil {
					commands = append(commands, unquoted)
				}
			}
		}
		return false
	})
	return commands
}

func TestDispatchCommands_InUsageText(t *testing.T) {
	t.Parallel()

	for _, command := range switchCaseCommands(t, "cli.go", "Run") {
		if !strings.Contains(version.UsageText(), "pulsarules_cli "+command) {
			t.Errorf("usage.txt has no %q line for a command Run dispatches", command)
		}
	}
}

func TestDispatchCommands_ParityWithParseArgs(t *testing.T) {
	t.Parallel()

	runCommands := switchCaseCommands(t, "cli.go", "Run")
	parseCommands := switchCaseCommands(
		t, filepath.Join("version", "parse.go"), "bindCommandFlags",
	)
	slices.Sort(runCommands)
	slices.Sort(parseCommands)
	if !slices.Equal(runCommands, parseCommands) {
		t.Errorf(
			"Run dispatches %v, bindCommandFlags binds %v - the two switches diverged",
			runCommands, parseCommands,
		)
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
	if exitErr, isExit := errors.AsType[*ExitError](err); isExit {
		return fmt.Sprintf("exit %d", exitErr.Code)
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
		{
			name: "warnings do not fail the run",
			source: "package alpha\n\nfunc f() {\n\t// one\n\t// two\n\t// three\n\t// four\n" +
				"\t_ = 1\n}\n",
			want: "ok",
		},
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
