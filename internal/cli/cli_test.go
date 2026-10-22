package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
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

			svc := servicesIn(t, ".")
			args := slices.Clone(testCase.args)
			if idx := slices.Index(args, "--project"); idx >= 0 {
				args[idx+1] = t.TempDir()
			}
			call, err := Parse(args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			err = call.Run(svc)
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
		{
			name: "governance falls back on the environment",
			args: []string{"governance"},
			want: "/from-env",
		},
		{
			name: "governance prefers its flag",
			args: []string{"governance", "--project", "/repo"},
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

func servicesIn(t *testing.T, projectDir string) bootstrap.Services {
	t.Helper()

	svc, err := bootstrap.New(bootstrap.Options{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("bootstrap.New: %v", err)
	}
	return svc
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), disk.DirPrivate); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func capture(t *testing.T, stdin string, body func()) (stdout, stderr string) {
	t.Helper()

	scratch := t.TempDir()
	stdinFile := filepath.Join(scratch, "in")
	writeFile(t, stdinFile, stdin)
	input, err := os.Open(stdinFile)
	if err != nil {
		t.Fatalf("open stdin: %v", err)
	}
	defer func() { _ = input.Close() }()
	outFile, err := os.Create(filepath.Join(scratch, "out"))
	if err != nil {
		t.Fatalf("create stdout: %v", err)
	}
	defer func() { _ = outFile.Close() }()
	errFile, err := os.Create(filepath.Join(scratch, "err"))
	if err != nil {
		t.Fatalf("create stderr: %v", err)
	}
	defer func() { _ = errFile.Close() }()
	savedIn, savedOut, savedErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = input, outFile, errFile
	func() {
		defer func() { os.Stdin, os.Stdout, os.Stderr = savedIn, savedOut, savedErr }()
		body()
	}()
	return readText(t, outFile.Name()), readText(t, errFile.Name())
}

func readText(t *testing.T, path string) string {
	t.Helper()

	captured, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(captured)
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()

	full := append([]string{
		"-C", dir, "-c", "user.name=n", "-c", "user.email=n@example.com",
		"-c", "commit.gpgsign=false",
	}, args...)
	if out, err := exec.CommandContext(t.Context(), "git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
