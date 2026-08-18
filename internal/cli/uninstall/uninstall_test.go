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
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
)

// TestRun_RoundTrip installs one skill into a throwaway project directory
// then uninstalls it, asserting both the rendered skill doc and the hook
// wiring Install wrote are gone, and that a second Run is not an error.
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

	// Idempotent: uninstalling a project Run already reversed is not an error.
	if err := Run(inj, uninstallOpts); err != nil {
		t.Fatalf("second Run: %v", err)
	}
}

// TestRun_KeepSkills asserts --keep-skills leaves the rendered skill doc in
// place while still removing the hook wiring.
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

// TestRun_UnknownTarget asserts an invalid --target fails before touching
// disk.
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

// TestRun_Idempotent asserts uninstalling a project install never touched is
// not an error.
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

// TestRun_DetectsBothTargetsWhenNoneGiven asserts that, with no --target,
// Run does not fall back to install's "claude only" default: it probes the
// project and reverses every layout it finds, so installing both claude and
// opencode then uninstalling with no --target leaves neither behind.
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

// TestRun_UnwiresLocalHooksScopeByDefault asserts that, with no
// --hooks-scope, Run finds and removes hook wiring install left in
// settings.local.json - the footgun where a project installed with
// --hooks-scope local kept live wiring after an uninstall that reported
// success.
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

// TestRun_ReturnsErrorOnGitHookFailure asserts a hard failure while
// reversing the git hooks makes Run return a non-nil error instead of the
// old behavior of warning to stderr and reporting success (exit 0) while
// leaving executable hook wiring in place.
func TestRun_ReturnsErrorOnGitHookFailure(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}

	// githook.Uninstall reads each hook path with os.ReadFile; making
	// commit-msg a directory forces a non-IsNotExist read error instead of
	// the "file absent" no-op case.
	hookAsDir := filepath.Join(projectDir, ".git", "hooks", "commit-msg")
	if err := os.MkdirAll(hookAsDir, 0o750); err != nil {
		t.Fatalf("mkdir %q: %v", hookAsDir, err)
	}

	opts := &cliopts.Options{Command: "uninstall", Project: projectDir}
	if err := Run(inj, opts); err == nil {
		t.Fatal("expected a non-nil error when git hook removal fails")
	}
}

// TestRun_TargetErrorNotDoubled reproduces a hard failure inside the
// "claude" target's Uninstall (settings.json replaced by a directory, so
// UnwireSettings's ReadFile hits a real I/O error, not the "absent file"
// no-op): target.Registry.Uninstall already wraps it as `uninstall target
// %q: %w`, so Run must not wrap it again with the same format.
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
