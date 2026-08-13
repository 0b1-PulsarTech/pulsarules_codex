package uninstall

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func TestRun_RoundTrip(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	installOpts := &version.Options{
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

	uninstallOpts := &version.Options{Command: "uninstall", Project: projectDir}
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

	installOpts := &version.Options{
		Command: "install",
		Project: projectDir,
		Skills:  "go-style",
		NoMCP:   true,
	}
	if err := install.Run(inj, installOpts); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	uninstallOpts := &version.Options{Command: "uninstall", Project: projectDir, KeepSkills: true}
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

func TestRun_UnknownTarget(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	opts := &version.Options{Command: "uninstall", Project: t.TempDir(), Target: []string{"bogus"}}
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
	opts := &version.Options{Command: "uninstall", Project: t.TempDir()}
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

	installOpts := &version.Options{
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

	uninstallOpts := &version.Options{Command: "uninstall", Project: projectDir}
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

	installOpts := &version.Options{
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

	uninstallOpts := &version.Options{Command: "uninstall", Project: projectDir}
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

	opts := &version.Options{Command: "uninstall", Project: projectDir}
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
	installOpts := &version.Options{
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

	uninstallOpts := &version.Options{Command: "uninstall", Project: projectDir}
	err := Run(inj, uninstallOpts)
	if err == nil {
		t.Fatal("expected an error when settings.json cannot be read")
	}
	if strings.Count(err.Error(), `uninstall target "claude"`) > 1 {
		t.Errorf("error = %q, wraps `uninstall target %q` more than once", err.Error(), "claude")
	}
}
