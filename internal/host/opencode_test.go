package host

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func TestUnwireOpencodeConfig_NoNote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		seedConfig string
	}{
		{name: "absent opencode.json"},
		{name: "present opencode.json with no wiring", seedConfig: `{"theme": "dark"}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if testCase.seedConfig != "" {
				path := filepath.Join(base, "opencode.json")
				if err := os.WriteFile(path, []byte(testCase.seedConfig), 0o600); err != nil {
					t.Fatalf("seed opencode.json: %v", err)
				}
			}

			var report Report
			if err := unwireOpencodeConfig(base, &report); err != nil {
				t.Fatalf("unwireOpencodeConfig: %v", err)
			}
			if len(report.Notes) != 0 {
				t.Errorf("Notes = %v, want none (nothing was unwired)", report.Notes)
			}
		})
	}
}

func TestUnwireOpencodeConfig_Notes(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	seed := `{"instructions": ["AGENTS.md"]}`
	path := filepath.Join(base, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed opencode.json: %v", err)
	}

	var report Report
	if err := unwireOpencodeConfig(base, &report); err != nil {
		t.Fatalf("unwireOpencodeConfig: %v", err)
	}
	if !slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "unwired")
	}) {
		t.Errorf("Notes = %v, want an 'unwired' entry", report.Notes)
	}
}

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

type opencodeConfig struct {
	Instructions []string `json:"instructions"`
	Skills       struct {
		Paths []string `json:"paths"`
	} `json:"skills"`
}

func readOpencodeConfig(tb testing.TB, configPath string) opencodeConfig {
	tb.Helper()
	raw, err := os.ReadFile(configPath) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read opencode.json: %v", err)
	}
	var config opencodeConfig
	if err = json.Unmarshal(raw, &config); err != nil {
		tb.Fatalf("parse opencode.json: %v", err)
	}
	return config
}

func TestOpencodeInstall_LegacyAgents(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	legacyContent := "# AGENTS.md - demo\n\n## Mandatory routing contract\n\nopencode has no " +
		"SessionStart hook, so this contract is stated here instead: ...\n"
	legacyPath := filepath.Join(base, ".opencode", "AGENTS.md")
	seedFile(t, legacyPath, legacyContent)
	configPath := filepath.Join(base, "opencode.json")
	seedConfig := `{"instructions": [".opencode/AGENTS.md", ".opencode/skills/*/SKILL.md"]}`
	seedFile(t, configPath, seedConfig)

	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (opencodeTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	assertMissing(t, legacyPath)
	assertPresent(t, filepath.Join(base, "AGENTS.md"))
	config := readOpencodeConfig(t, configPath)
	if slices.Contains(config.Instructions, ".opencode/AGENTS.md") {
		t.Errorf("legacy instructions entry survived: %v", config.Instructions)
	}
	if slices.Contains(config.Instructions, ".opencode/skills/*/SKILL.md") {
		t.Errorf("eager SKILL.md glob survived in instructions: %v", config.Instructions)
	}
	if !slices.Contains(config.Skills.Paths, opencode.SkillsSubdir) {
		t.Errorf("skills.paths missing %q, got %v", opencode.SkillsSubdir, config.Skills.Paths)
	}
	if count := countEntry(config.Instructions, "AGENTS.md"); count != 1 {
		t.Errorf("AGENTS.md entry appears %d times, want 1: %v", count, config.Instructions)
	}
}

func countEntry(entries []string, want string) int {
	return len(slices.DeleteFunc(slices.Clone(entries), func(entry string) bool {
		return entry != want
	}))
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
	requirePresent(t, skillDoc)
	agentsDoc := filepath.Join(base, "AGENTS.md")
	requirePresent(t, agentsDoc)

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	report, err := (opencodeTarget{}).Uninstall(uctx)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	assertMissing(t, skillDoc)
	assertMissing(t, agentsDoc)
	assertMissing(t, filepath.Join(base, ".opencode", "plugins"))
	assertNote(t, report.Notes, "unwired")
	assertNote(t, report.Notes, "removed opencode governance plugin")
	assertMissing(t, filepath.Join(base, ".opencode"))

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

func TestOpencodeInstall_GoplsLanguages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		languages detect.Set
		wantGopls bool
	}{
		{name: "dart project gets no gopls", languages: detect.NewSet(language.Dart)},
		{name: "unknown languages get no gopls", languages: detect.Set{}},
		{
			name:      "go project gets gopls when it is on PATH",
			languages: detect.NewSet(language.Go),
			wantGopls: mcp.GoplsOnPath(),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.Languages = testCase.languages
			ctx.NoMCP = false
			if _, err := (opencodeTarget{}).Install(ctx); err != nil {
				t.Fatalf("Install: %v", err)
			}
			config, err := os.ReadFile(filepath.Join(base, "opencode.json"))
			if err != nil {
				t.Fatalf("read opencode.json: %v", err)
			}
			if got := strings.Contains(string(config), `"gopls"`); got != testCase.wantGopls {
				t.Fatalf("gopls wired = %v, want %v; config = %s", got, testCase.wantGopls, config)
			}
		})
	}
}
