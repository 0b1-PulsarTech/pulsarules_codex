package target

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

func TestOpencodeTargetName(t *testing.T) {
	t.Parallel()
	if got := (opencodeTarget{}).Name(); got != "opencode" {
		t.Fatalf("Name() = %q, want opencode", got)
	}
}

func TestOpencodeTargetPresent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		makeOpencode bool
		makeConfig   bool
		want         bool
	}{
		{name: "opencode dir present", makeOpencode: true, want: true},
		{name: "bare opencode.json", makeConfig: true, want: true},
		{name: "untouched project", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if testCase.makeOpencode {
				if err := os.MkdirAll(filepath.Join(base, ".opencode"), 0o750); err != nil {
					t.Fatalf("mkdir .opencode: %v", err)
				}
			}
			if testCase.makeConfig {
				path := filepath.Join(base, "opencode.json")
				if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
					t.Fatalf("seed opencode.json: %v", err)
				}
			}
			if got := (opencodeTarget{}).Present(base); got != testCase.want {
				t.Errorf("Present(%q) = %v, want %v", base, got, testCase.want)
			}
		})
	}
}

func TestOpencodeTargetInstall(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})

	report, err := opencodeTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	wantPaths := []string{
		filepath.Join(base, ".opencode", "skills", "go-style", "SKILL.md"),
		filepath.Join(base, "AGENTS.md"),
		filepath.Join(base, "opencode.json"),
	}
	for _, path := range wantPaths {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("missing %q: %v", path, statErr)
		}
	}
	if !slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "wired opencode")
	}) {
		t.Errorf("Notes missing 'wired opencode': %v", report.Notes)
	}
}

func TestOpencodeInstall_PluginBackup(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	pluginsDir := filepath.Join(base, ".opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o750); err != nil {
		t.Fatalf("mkdir plugins: %v", err)
	}
	pluginPath := filepath.Join(pluginsDir, "pulsarules-governance.js")
	if err := os.WriteFile(
		pluginPath, []byte("// hand-authored, not ours\n"), 0o600,
	); err != nil {
		t.Fatalf("seed foreign plugin: %v", err)
	}

	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false

	report, err := opencodeTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !slices.ContainsFunc(report.Warnings, func(warning string) bool {
		return strings.Contains(warning, "backed up")
	}) {
		t.Errorf("Warnings missing a backup notice: %v", report.Warnings)
	}
}

func TestOpencodeInstall_LegacyAgents(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	legacyDir := filepath.Join(base, ".opencode")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatalf("mkdir .opencode: %v", err)
	}
	legacyContent := "# AGENTS.md - demo\n\n## Mandatory routing contract\n\nopencode has no " +
		"SessionStart hook, so this contract is stated here instead: ...\n"
	legacyPath := filepath.Join(legacyDir, "AGENTS.md")
	if err := os.WriteFile(legacyPath, []byte(legacyContent), 0o600); err != nil {
		t.Fatalf("seed legacy AGENTS.md: %v", err)
	}
	seedConfig := `{"instructions": [".opencode/AGENTS.md", ".opencode/skills/*/SKILL.md"]}`
	configPath := filepath.Join(base, "opencode.json")
	if err := os.WriteFile(configPath, []byte(seedConfig), 0o600); err != nil {
		t.Fatalf("seed opencode.json: %v", err)
	}

	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (opencodeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if _, statErr := os.Stat(legacyPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected legacy %q to be retired, stat err = %v", legacyPath, statErr)
	}
	if _, statErr := os.Stat(filepath.Join(base, "AGENTS.md")); statErr != nil {
		t.Errorf("expected root AGENTS.md to exist: %v", statErr)
	}

	raw, err := os.ReadFile(configPath) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read opencode.json: %v", err)
	}
	var config struct {
		Instructions []string `json:"instructions"`
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse opencode.json: %v", err)
	}
	if slices.Contains(config.Instructions, ".opencode/AGENTS.md") {
		t.Errorf("legacy instructions entry survived: %v", config.Instructions)
	}
	count := 0
	for _, entry := range config.Instructions {
		if entry == "AGENTS.md" {
			count++
		}
	}
	if count != 1 {
		t.Errorf(
			"AGENTS.md instructions entry appears %d times, want 1: %v",
			count,
			config.Instructions,
		)
	}
}

func TestOpencodeTargetUninstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	if _, err := (opencodeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	skillDoc := filepath.Join(base, ".opencode", "skills", "go-style", "SKILL.md")
	if _, statErr := os.Stat(skillDoc); statErr != nil {
		t.Fatalf("Install did not write %q: %v", skillDoc, statErr)
	}
	agentsDoc := filepath.Join(base, "AGENTS.md")
	if _, statErr := os.Stat(agentsDoc); statErr != nil {
		t.Fatalf("Install did not write %q: %v", agentsDoc, statErr)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	report, err := (opencodeTarget{}).Uninstall(uctx)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(skillDoc); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", skillDoc, statErr)
	}
	if _, statErr := os.Stat(agentsDoc); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", agentsDoc, statErr)
	}
	if _, statErr := os.Stat(
		filepath.Join(base, ".opencode", "plugins"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("expected the plugin dir to be removed, stat err = %v", statErr)
	}
	if !slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "unwired")
	}) {
		t.Errorf("Notes missing an 'unwired' entry: %v", report.Notes)
	}
	if !slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "removed opencode governance plugin")
	}) {
		t.Errorf("Notes missing the plugin removal note: %v", report.Notes)
	}
	opencodeDir := filepath.Join(base, ".opencode")
	if _, statErr := os.Stat(opencodeDir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be reaped once empty, stat err = %v", opencodeDir, statErr)
	}

	if _, err = (opencodeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("second Uninstall: %v", err)
	}
}

func TestOpencodeUninstall_UserFile(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	if _, err := (opencodeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	userFile := filepath.Join(base, ".opencode", "notes.txt")
	if err := os.WriteFile(userFile, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (opencodeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, statErr := os.Stat(userFile); statErr != nil {
		t.Errorf("expected %q to survive, stat err = %v", userFile, statErr)
	}
	opencodeDir := filepath.Join(base, ".opencode")
	if _, statErr := os.Stat(opencodeDir); statErr != nil {
		t.Errorf("expected %q to survive (holds a user file), stat err = %v", opencodeDir, statErr)
	}
}

func TestOpencodeUninstall_NoPluginNote(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	report, err := (opencodeTarget{}).Uninstall(uctx)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "removed opencode governance plugin")
	}) {
		t.Errorf("Notes falsely claims a plugin removal: %v", report.Notes)
	}
}

func TestOpencodeUninstall_KeepSkills(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (opencodeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	agentsDoc := filepath.Join(base, "AGENTS.md")

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry(), KeepSkills: true}
	if _, err := (opencodeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(agentsDoc); statErr != nil {
		t.Errorf("--keep-skills should have kept %q, stat err = %v", agentsDoc, statErr)
	}
}

func TestOpencodeUninstall_UserAuthored(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	agentsDoc := filepath.Join(base, "AGENTS.md")
	foreign := "# My own AGENTS.md\nDo not touch.\n"
	if err := os.WriteFile(agentsDoc, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed user-authored AGENTS.md: %v", err)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (opencodeTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	got, err := os.ReadFile(agentsDoc) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("expected user-authored AGENTS.md to survive, read err = %v", err)
	}
	if string(got) != foreign {
		t.Errorf("AGENTS.md content changed, got %q want %q", got, foreign)
	}
}
