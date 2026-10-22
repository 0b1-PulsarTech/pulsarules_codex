package commitlint

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
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
	return RunCommitLint(inj, &version.Options{CommitMsg: message})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRunCommitLint_EmojiWindowFallsBack(t *testing.T) {
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
				t.Fatalf("RunCommitLint() error = %v, want findings %v", err, testCase.wantErr)
			}
		})
	}
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
					t.Fatalf("RunCommitLint() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, command.ErrFindings) {
				t.Fatalf("RunCommitLint() error = %v, want command.ErrFindings (exit code 1)", err)
			}
		})
	}
}

func TestRunCommitLint_ReturnsErrFindings(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const messageTrippingEmojiTypeAndInitialRules = "bad commit message"
	err := RunCommitLint(inj, &version.Options{CommitMsg: messageTrippingEmojiTypeAndInitialRules})

	if !errors.Is(err, command.ErrFindings) {
		t.Fatalf("error = %v, want command.ErrFindings", err)
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

	err := RunCommitLint(inj, &version.Options{CommitMsg: "valid enough message"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunCommitLint() error = %v, want wrapping %v", err, wantErr)
	}
}

func TestRunCommitLint_NoRepository(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: t.TempDir()}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	const invalidCommitMessage = "bad commit message"
	err := RunCommitLint(
		inj,
		&version.Options{CommitMsg: invalidCommitMessage},
	)
	if !errors.Is(err, command.ErrFindings) {
		t.Fatalf("error = %v, want command.ErrFindings from analysis findings", err)
	}
}
