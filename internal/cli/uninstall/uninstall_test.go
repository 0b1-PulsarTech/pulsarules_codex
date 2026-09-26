package uninstall

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
)

func TestRun_RoundTrip(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &cliopts.Options{
		Command: "install",
		Project: projectDir,
		Skills:  "go-style",
		NoMCP:   true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	skillDoc := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skillDoc); statErr != nil {
		t.Fatalf("install did not write %q: %v", skillDoc, statErr)
	}
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); statErr != nil {
		t.Fatalf("install did not wire %q: %v", settingsPath, statErr)
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(skillDoc); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", skillDoc, statErr)
	}
	if _, statErr := os.Stat(settingsPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", settingsPath, statErr)
	}

	t.Run("idempotent: second uninstall is not an error", func(t *testing.T) {
		t.Parallel()
		if err := Run(inj, uninstallOpts); err != nil {
			t.Fatalf("second Run: %v", err)
		}
	})
}

func TestRun_KeepSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &cliopts.Options{
		Command: "install",
		Project: projectDir,
		Skills:  "go-style",
		NoMCP:   true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir, KeepSkills: true}
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	skillDoc := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skillDoc); statErr != nil {
		t.Errorf("--keep-skills should have kept %q, stat err = %v", skillDoc, statErr)
	}
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected hook wiring to be removed regardless, stat err = %v", statErr)
	}
}

func TestRun_RemovesLanguageConfig(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(projectDir, "go.mod"),
		[]byte("module x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &cliopts.Options{
		Command: "install",
		Project: projectDir,
		Skills:  "go-style",
		NoMCP:   true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	configPath := filepath.Join(projectDir, langdetect.ConfigFileName)
	if _, statErr := os.Stat(configPath); statErr != nil {
		t.Fatalf("install did not write %q: %v", configPath, statErr)
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(configPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", configPath, statErr)
	}
}

func TestRun_UnknownTarget(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	opts := &cliopts.Options{Command: "uninstall", Project: t.TempDir(), Target: []string{"bogus"}}
	if err := Run(inj, opts); err == nil {
		t.Fatal("expected an error for an unknown --target")
	}
}

func TestRun_Idempotent(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	opts := &cliopts.Options{Command: "uninstall", Project: t.TempDir()}
	if err := Run(inj, opts); err != nil {
		t.Fatalf("Run on untouched project: %v", err)
	}
}

func TestRun_DetectsBothTargetsWhenNoneGiven(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &cliopts.Options{
		Command: "install",
		Project: projectDir,
		Skills:  "go-style",
		Target:  []string{"claude", "opencode"},
		NoMCP:   true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	claudeSkill := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")
	opencodeConfig := filepath.Join(projectDir, "opencode.json")
	for _, path := range []string{claudeSkill, opencodeConfig} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("install did not write %q: %v", path, statErr)
		}
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, path := range []string{claudeSkill, opencodeConfig} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed, stat err = %v", path, statErr)
		}
	}
}

func TestRun_UnwiresLocalHooksScopeByDefault(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &cliopts.Options{
		Command:    "install",
		Project:    projectDir,
		Skills:     "go-style",
		NoMCP:      true,
		HooksScope: "local",
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	localSettings := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, statErr := os.Stat(localSettings); statErr != nil {
		t.Fatalf("install did not wire %q: %v", localSettings, statErr)
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, statErr := os.Stat(localSettings); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", localSettings, statErr)
	}
}

func TestRun_ReturnsErrorOnGitHookFailure(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	hookAsDir := filepath.Join(
		projectDir,
		".git",
		"hooks",
		"commit-msg",
	)
	if err := os.MkdirAll(hookAsDir, 0o750); err != nil {
		t.Fatalf("mkdir %q: %v", hookAsDir, err)
	}

	opts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, opts); err == nil {
		t.Fatal("expected a non-nil error when git hook removal fails")
	}
}

func TestRun_TargetErrorNotDoubled(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	installOpts := &cliopts.Options{
		Command: "install", Project: projectDir, Skills: "go-style", NoMCP: true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("remove settings.json: %v", err)
	}
	if err := os.MkdirAll(settingsPath, 0o750); err != nil {
		t.Fatalf("replace settings.json with a directory: %v", err)
	}

	uninstallOpts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	err := Run(inj, uninstallOpts)
	if err == nil {
		t.Fatal("expected an error when settings.json cannot be read")
	}
	if strings.Count(err.Error(), `uninstall target "claude"`) > 1 {
		t.Errorf("error = %q, wraps `uninstall target %q` more than once", err.Error(), "claude")
	}
}

func captureStdout(tb testing.TB, fn func()) string {
	tb.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		tb.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()

	if closeErr := writer.Close(); closeErr != nil {
		tb.Fatalf("close pipe writer: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, reader); copyErr != nil {
		tb.Fatalf("read pipe: %v", copyErr)
	}
	return buf.String()
}

//nolint:paralleltest // captureStdout swaps os.Stdout.
func TestRemoveLanguageConfig(t *testing.T) {
	testCases := []struct {
		name       string
		seed       func(tb testing.TB, configPath string)
		wantStdout string
		wantErr    bool
	}{
		{
			name: "present file is removed and reported",
			seed: func(tb testing.TB, configPath string) {
				tb.Helper()
				if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
					tb.Fatalf("seed config: %v", err)
				}
			},
			wantStdout: "removed " + langdetect.ConfigFileName + "\n",
		},
		{
			name: "absent file prints nothing",
			seed: func(testing.TB, string) {},
		},
		{
			name: "unremovable config is an error",
			seed: func(tb testing.TB, configPath string) {
				tb.Helper()
				if err := os.MkdirAll(filepath.Join(configPath, "child"), 0o750); err != nil {
					tb.Fatalf("seed directory: %v", err)
				}
			},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			testCase.seed(t, filepath.Join(dir, langdetect.ConfigFileName))

			var err error
			stdout := captureStdout(t, func() { err = removeLanguageConfig(dir) })
			if (err != nil) != testCase.wantErr {
				t.Errorf("removeLanguageConfig err = %v, wantErr %v", err, testCase.wantErr)
			}
			if stdout != testCase.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, testCase.wantStdout)
			}
		})
	}
}
