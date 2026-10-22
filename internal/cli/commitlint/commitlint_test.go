package commitlint

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func lintInRepo(t *testing.T, configYAML, message string) error {
	t.Helper()

	return lintAfterHistory(t, configYAML, nil, message)
}

func lintAfterHistory(t *testing.T, configYAML string, history []string, message string) error {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	for _, subject := range history {
		runGit(t, dir, "commit", "-q", "--allow-empty", "-m", subject)
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
	return (&Command{msg: message}).Run(inj, nil)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestCommand_Run_EmojiWindowFallsBack(t *testing.T) {
	t.Parallel()

	const (
		repeat  = ":tea: docs(cli): Document the config flag"
		recent  = ":tea: docs(cli): Document an earlier flag"
		newest  = ":memo: docs(cli): Document the newest flag"
		wrapper = "version: 1\nanalyzers:\n  commit-lint:\n    params:\n"
	)
	fillers := []string{
		":gear: chore(cli): Tune a setting",
		":recycle: refactor(cli): Fold a helper",
	}
	testCases := []struct {
		name    string
		config  string
		between []string
		wantErr bool
	}{
		{"no params use the defaults", "", nil, true},
		{"zero windows fall back", wrapper + "      emoji_hard_window: 0\n" +
			"      emoji_soft_window: 0\n", nil, true},
		{"null windows fall back", wrapper + "      emoji_hard_window: null\n" +
			"      emoji_soft_window: null\n", nil, true},
		{"negative windows fall back", wrapper + "      emoji_hard_window: -1\n" +
			"      emoji_soft_window: -1\n", nil, true},
		{"a MaxInt64 window is clamped", wrapper +
			"      emoji_hard_window: 9223372036854775807\n" +
			"      emoji_soft_window: 9223372036854775807\n", nil, true},
		{"a valid window is honoured", wrapper + "      emoji_hard_window: 3\n", nil, true},
		{"a valid window excludes an older repeat", wrapper +
			"      emoji_hard_window: 3\n      emoji_soft_window: 3\n", fillers, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			history := append([]string{recent}, testCase.between...)
			err := lintAfterHistory(t, testCase.config, append(history, newest), repeat)
			if got := errors.Is(err, command.ErrFindings); got != testCase.wantErr {
				t.Fatalf("Run() error = %v, want findings %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCommand_Run_ProjectConfig(t *testing.T) {
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
					t.Fatalf("Run() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, command.ErrFindings) {
				t.Fatalf("Run() error = %v, want command.ErrFindings (exit code 1)", err)
			}
		})
	}
}

func TestCommand_Run_ReturnsErrFindings(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const messageTrippingEmojiTypeAndInitialRules = "bad commit message"
	err := (&Command{msg: messageTrippingEmojiTypeAndInitialRules}).Run(inj, nil)

	if !errors.Is(err, command.ErrFindings) {
		t.Fatalf("error = %v, want command.ErrFindings", err)
	}
}

func TestCommand_Run_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	wantErr := errors.New("boom: injector wiring failed")
	remy.RegisterConstructorErr(
		inj, remy.Factory[*vcs.Repository],
		func() (*vcs.Repository, error) { return nil, wantErr },
	)

	err := (&Command{msg: "valid enough message"}).Run(inj, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want wrapping %v", err, wantErr)
	}
}

func TestCommand_Run_NoRepository(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: t.TempDir()}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const invalidCommitMessage = "bad commit message"
	err := (&Command{msg: invalidCommitMessage}).Run(inj, nil)
	if !errors.Is(err, command.ErrFindings) {
		t.Fatalf("error = %v, want command.ErrFindings from analysis findings", err)
	}
}

func TestCommand_Bind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want Command
		dir  string
	}{
		{name: "defaults", dir: "."},
		{
			name: "every flag",
			args: []string{"--msg", "m", "--file", "f", "--project", "p"},
			want: Command{msg: "m", file: "f"},
			dir:  "p",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var cmd Command
			fs := flag.NewFlagSet("commitlint", flag.ContinueOnError)
			cmd.Bind(fs)
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cmd.msg != testCase.want.msg || cmd.file != testCase.want.file ||
				cmd.ProjectDir() != testCase.dir {
				t.Errorf("Command = %+v dir %q, want %+v and %q",
					cmd, cmd.ProjectDir(), testCase.want, testCase.dir)
			}
		})
	}
}

func TestCommand_Run_CatalogFailure(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	remy.RegisterConstructorErr(
		inj, remy.Factory[*vcs.Repository],
		func() (*vcs.Repository, error) { return nil, vcs.ErrNoRepository },
	)
	remy.RegisterInstance(inj, &knowledge.Index{})

	err := (&Command{msg: "feat: x"}).Run(inj, nil)
	if err == nil || !strings.Contains(err.Error(), "get emoji catalog") {
		t.Fatalf("Run() error = %v, want one naming %q", err, "get emoji catalog")
	}
}
