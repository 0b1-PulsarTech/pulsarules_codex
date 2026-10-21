package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
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
	return runCommitLint(inj, &version.Options{CommitMsg: message})
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
	testCases := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{"no params use the defaults", "", true},
		{"zero windows fall back", wrapper + "      emoji_hard_window: 0\n" +
			"      emoji_soft_window: 0\n", true},
		{"null windows fall back", wrapper + "      emoji_hard_window: null\n" +
			"      emoji_soft_window: null\n", true},
		{"negative windows fall back", wrapper + "      emoji_hard_window: -1\n" +
			"      emoji_soft_window: -1\n", true},
		{"a valid window is honoured", wrapper + "      emoji_hard_window: 3\n", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := lintAfterHistory(t, testCase.config, []string{recent, newest}, repeat)
			if got := errors.Is(err, ErrFindings); got != testCase.wantErr {
				t.Fatalf("runCommitLint() error = %v, want findings %v", err, testCase.wantErr)
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
					t.Fatalf("runCommitLint() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrFindings) {
				t.Fatalf("runCommitLint() error = %v, want ErrFindings (exit code 1)", err)
			}
		})
	}
}

func TestOptionalRepository(t *testing.T) {
	t.Parallel()

	wiringErr := errors.New("boom: injector wiring failed")
	dir := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	present, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	testCases := []struct {
		name     string
		repo     *vcs.Repository
		repoErr  error
		wantNil  bool
		wantErr  error
		wantRoot string
	}{
		{name: "absent repository is a true nil", repoErr: vcs.ErrNoRepository, wantNil: true},
		{name: "wiring failure is returned", repoErr: wiringErr, wantNil: true, wantErr: wiringErr},
		{name: "present repository is returned", repo: present, wantRoot: present.Root()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			remy.RegisterConstructorErr(
				inj, remy.Factory[*vcs.Repository],
				func() (*vcs.Repository, error) { return testCase.repo, testCase.repoErr },
			)

			repo, err := optionalRepository(inj)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("optionalRepository() error = %v, want %v", err, testCase.wantErr)
			}
			if (repo == nil) != testCase.wantNil {
				t.Fatalf("repo == nil is %v, want %v (typed nil?)", repo == nil, testCase.wantNil)
			}
			if repo != nil && repo.Root() != testCase.wantRoot {
				t.Fatalf("Root() = %q, want %q", repo.Root(), testCase.wantRoot)
			}
		})
	}
}
