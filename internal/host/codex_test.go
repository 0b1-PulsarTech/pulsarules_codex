package host

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

const trustNote = "trust the project in Codex and run /hooks to approve the hooks"

func TestCodexTargetName(t *testing.T) {
	t.Parallel()
	if got := (codexTarget{}).Name(); got != "codex" {
		t.Fatalf("Name() = %q, want codex", got)
	}
}

func TestCodexTargetPresent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "codex dir present", dir: ".codex", want: true},
		{name: "agents skills alone are not enough", dir: ".agents"},
		{name: "untouched project"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			if testCase.dir != "" {
				if err := os.MkdirAll(filepath.Join(base, testCase.dir), 0o750); err != nil {
					t.Fatal(err)
				}
			}
			if got := (codexTarget{}).Present(base); got != testCase.want {
				t.Errorf("Present(%q) = %v, want %v", base, got, testCase.want)
			}
		})
	}
}

func exists(tb testing.TB, path string) bool {
	tb.Helper()
	_, err := os.Stat(path)
	return err == nil
}

func TestCodexInstall_ProjectLayout(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	report, err := (codexTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, want := range []string{
		".agents/skills/go-style/SKILL.md", "AGENTS.md", ".codex/hooks.json",
		".codex/hooks/skill-router-reminder.sh", ".codex/bin/pulsarules_cli", ".codex/.gitignore",
	} {
		if !exists(t, filepath.Join(base, want)) {
			t.Errorf("Install did not write %s", want)
		}
	}
	for _, unwanted := range []string{".claude", ".codex/skills", ".codex/config.toml", ".mcp.json"} {
		if exists(t, filepath.Join(base, unwanted)) {
			t.Errorf("Install wrote %s, which Codex does not need here", unwanted)
		}
	}
	if !slices.Contains(report.Notes, trustNote) {
		t.Errorf("Notes = %q, want the trust note %q", report.Notes, trustNote)
	}
}

func TestCodexInstall_NoHooksWritesNoCodexDir(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	report, err := (codexTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if exists(t, filepath.Join(base, ".codex")) {
		t.Error("--no-hooks and --no-mcp must leave no .codex directory")
	}
	if slices.Contains(report.Notes, trustNote) {
		t.Errorf("Notes = %q, want no trust note without hooks", report.Notes)
	}
}

func TestCodexInstall_AgentsMDStaysUnderCodexLimit(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, nil)
	for _, skill := range ctx.Index.SkillsOrdered() {
		ctx.IDs = append(ctx.IDs, skill.ID)
	}
	if _, err := (codexTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	stat, err := os.Stat(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	const codexDefaultLimit = 32 * 1024
	if stat.Size() >= codexDefaultLimit {
		t.Errorf("AGENTS.md is %d bytes, Codex reads %d by default", stat.Size(), codexDefaultLimit)
	}
}

func TestCodexInstall_KeepsAUserAGENTSMD(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	mine := filepath.Join(base, "AGENTS.md")
	if err := os.WriteFile(mine, []byte("my own rules\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := newTestContext(t, base, []string{"go-style"})
	report, err := (codexTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got, _ := os.ReadFile(
		mine,
	); string(
		got,
	) != "my own rules\n" { //nolint:gosec // test fixture.
		t.Errorf("AGENTS.md = %q, want the user's own file kept", got)
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), mine) {
		t.Errorf("Warnings = %q, want the kept file reported", report.Warnings)
	}
	if _, err = (codexTarget{}).Uninstall(UninstallContext{Base: base}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !exists(t, mine) {
		t.Error("Uninstall removed the user's own AGENTS.md")
	}
}

func TestCodexUninstall_ReversesAProjectInstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	if _, err := (codexTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := (codexTarget{}).Uninstall(UninstallContext{Base: base}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil || len(entries) != 0 {
		t.Errorf("project after uninstall = %v, %v, want it empty", entries, err)
	}
}

func TestCodexUninstall_KeepSkillsKeepsSkillsAndAGENTSMD(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoHooks = false
	if _, err := (codexTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := (codexTarget{}).Uninstall(
		UninstallContext{Base: base, KeepSkills: true},
	); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	for _, kept := range []string{".agents/skills/go-style/SKILL.md", "AGENTS.md"} {
		if !exists(t, filepath.Join(base, kept)) {
			t.Errorf("--keep-skills removed %s", kept)
		}
	}
	if exists(t, filepath.Join(base, ".codex", "hooks.json")) {
		t.Error("the hook wiring must go even with --keep-skills")
	}
}

func TestCodexInstall_RetiresOnlyOurOwnLegacySkills(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ours := filepath.Join(base, ".codex", "skills", "old-skill")
	theirs := filepath.Join(base, ".codex", "skills", "their-skill")
	for dir, body := range map[string]string{
		ours:   "<!-- " + ownedfile.Marker + " -->\n# old\n",
		theirs: "# written by hand\n",
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	settings := filepath.Join(base, ".codex", "settings.json")
	if err := os.WriteFile(
		settings,
		[]byte(`{"command":"x/skill-router-reminder.sh"}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	report, err := (codexTarget{}).Install(newTestContext(t, base, []string{"go-style"}))
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if exists(t, ours) || !exists(t, theirs) {
		t.Errorf(
			"legacy cleanup: ours present = %v, theirs present = %v",
			exists(t, ours),
			exists(t, theirs),
		)
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), settings) {
		t.Errorf("Warnings = %q, want the settings.json that Codex ignores", report.Warnings)
	}
}

func TestCodexInstall_GlobalUsesHomeSkillsAndCodexHome(t *testing.T) {
	t.Parallel()

	home, codexHome := t.TempDir(), filepath.Join(t.TempDir(), "codex-home")
	ctx := newTestContext(t, home, []string{"go-style"})
	ctx.NoHooks = false
	ctx.Global, ctx.CodexDir = true, codexHome
	if _, err := (codexTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, want := range []string{
		filepath.Join(home, ".agents/skills/go-style/SKILL.md"),
		filepath.Join(codexHome, "AGENTS.md"), filepath.Join(codexHome, "hooks.json"),
		filepath.Join(codexHome, "hooks/skill-router-reminder.sh"),
	} {
		if !exists(t, want) {
			t.Errorf("global Install did not write %s", want)
		}
	}
	if exists(t, filepath.Join(home, ".codex")) || exists(t, filepath.Join(home, "AGENTS.md")) {
		t.Error("a global install must write only under CODEX_HOME and ~/.agents")
	}
	if exists(t, filepath.Join(codexHome, ".gitignore")) {
		t.Error("a global install is no repository and gets no .gitignore")
	}
	if _, err := (codexTarget{}).Uninstall(UninstallContext{
		Base: home, Global: true, CodexDir: codexHome,
	}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("home after uninstall = %v, want it empty", entries)
	}
	if entries, _ := os.ReadDir(codexHome); len(entries) != 0 {
		t.Errorf("codex home after uninstall = %v, want it empty", entries)
	}
}

//nolint:paralleltest // fakeGoplsOnPath needs t.Setenv, which bars t.Parallel.
func TestCodexInstall_WiresGoplsIntoConfigToml(t *testing.T) {
	fakeGoplsOnPath(t)
	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	ctx.NoMCP = false
	ctx.Languages = detect.NewSet(language.Go)
	report, err := (codexTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	config, readErr := os.ReadFile(
		filepath.Join(base, ".codex", "config.toml"),
	) //nolint:gosec // test.
	if readErr != nil {
		t.Fatalf("read config.toml: %v", readErr)
	}
	if !strings.Contains(
		string(config),
		"[mcp_servers.gopls]\ncommand = \"gopls\"\nargs = [\"mcp\"]\n",
	) ||
		strings.Contains(string(config), "cwd") {
		t.Errorf("config.toml = %q, want the gopls server and no cwd", config)
	}
	if !exists(t, filepath.Join(base, ".agents/skills/gopls-navigation/SKILL.md")) {
		t.Error("the gopls-navigation skill is generated next to the others")
	}
	if len(report.Warnings) != 0 {
		t.Errorf("Warnings = %q, want none", report.Warnings)
	}
	if _, err = (codexTarget{}).Uninstall(UninstallContext{Base: base}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if exists(t, filepath.Join(base, ".codex")) {
		t.Error("Uninstall left the .codex directory behind")
	}
}

type configState struct {
	name, seed, wantWarning string
	symlink                 bool
}

func configStates() []configState {
	const bare = "[mcp_servers]\ndart = { command = \"mine\" }\n"
	const unterminated = "# >>> pulsarules_cli managed block: removed by uninstall\n"
	return []configState{
		{"own table clashes", "[mcp_servers.gopls]\n", "defines [mcp_servers.gopls];", false},
		{"a bare parent without gopls is no clash", bare, "", false},
		{"gopls under a bare parent", "[mcp_servers]\ngopls = {}\n", "defines gopls under", false},
		{"an inline parent says why", "mcp_servers = {}\n", "inline mcp_servers table", false},
		{"an unterminated block", unterminated, "alone", false},
		{"a symlinked config", bare, `    command = "gopls"`, true},
	}
}

func seedConfig(tb testing.TB, base string, testCase configState) (config, target string) {
	tb.Helper()
	config = filepath.Join(base, ".codex", "config.toml")
	target = config
	if testCase.symlink {
		target = filepath.Join(base, "shared.toml")
	}
	if err := os.MkdirAll(filepath.Dir(config), 0o750); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(testCase.seed), 0o600); err != nil {
		tb.Fatal(err)
	}
	if testCase.symlink && os.Symlink(target, config) != nil {
		tb.Fatal("symlink")
	}
	return config, target
}

func assertInstalled(tb testing.TB, state configState, warnings []string, got []byte) {
	tb.Helper()
	ours := strings.Contains(string(got), "[mcp_servers.gopls]\ncommand = \"gopls\"")
	warned := state.wantWarning == "" || hasWarning(warnings, state.wantWarning)
	if ours != (state.wantWarning == "") || !warned {
		tb.Fatalf("Install: ours %v, %q:\n%s", ours, warnings, got)
	}
}

func hasWarning(warnings []string, want string) bool {
	return strings.Contains(strings.Join(warnings, "\n"), want)
}

//nolint:paralleltest // fakeGoplsOnPath needs t.Setenv, which bars t.Parallel.
func TestCodexInstall_ConfigStates(t *testing.T) {
	for _, testCase := range configStates() {
		t.Run(testCase.name, func(t *testing.T) {
			fakeGoplsOnPath(t)
			base := t.TempDir()
			config, target := seedConfig(t, base, testCase)
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.NoMCP, ctx.Languages = false, detect.NewSet(language.Go)
			installed, err := (codexTarget{}).Install(ctx)
			got, _ := os.ReadFile(target) //nolint:gosec // test fixture.
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			assertInstalled(t, testCase, installed.Warnings, got)
			uninstalled, err := (codexTarget{}).Uninstall(UninstallContext{Base: base})
			stat, _ := os.Lstat(config)
			linked := stat.Mode()&os.ModeSymlink != 0
			if got, _ = os.ReadFile(target); err != nil || string(got) != testCase.seed ||
				linked != testCase.symlink ||
				testCase.symlink && !hasWarning(uninstalled.Warnings, "left it alone") {
				t.Errorf("Uninstall = %v, %q, %q, want the seed", err, uninstalled.Warnings, got)
			}
		})
	}
}

func TestCodexInstall_FailuresAreWrapped(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, ".agents"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (codexTarget{}).Install(newTestContext(t, base, []string{"go-style"}))
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Install = %v, want a failure to write under a file named .agents", err)
	}
}

func TestCodexUninstall_UnparseableHooksFileWarns(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	hooksFile := filepath.Join(base, ".codex", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksFile), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooksFile, []byte(`[1]`), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := (codexTarget{}).Uninstall(UninstallContext{Base: base})
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], hooksFile) {
		t.Errorf("Warnings = %q, want one warning for the unparseable hooks file", report.Warnings)
	}
	if got, _ := os.ReadFile(hooksFile); string(got) != `[1]` { //nolint:gosec // test fixture.
		t.Errorf("hooks.json = %q, want it untouched", got)
	}
}

func TestCodexInstall_ConfigAndSettingsFailuresStopTheInstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		blocker string
		noMCP   bool
		want    string
	}{
		{name: "config.toml is a directory", blocker: "config.toml", want: "write mcp: "},
		{
			name:    "settings.json is a directory",
			blocker: "settings.json",
			noMCP:   true,
			want:    "check codex settings: ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			if err := os.MkdirAll(
				filepath.Join(base, ".codex", testCase.blocker),
				0o750,
			); err != nil {
				t.Fatal(err)
			}
			ctx := newTestContext(t, base, []string{"go-style"})
			ctx.NoMCP = testCase.noMCP
			_, err := (codexTarget{}).Install(ctx)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("Install = %v, want an error containing %q", err, testCase.want)
			}
		})
	}
}
