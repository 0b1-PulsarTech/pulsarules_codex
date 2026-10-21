package opencode

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func testTemplates(t *testing.T) fs.FS {
	t.Helper()
	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return templates
}

func pluginScriptSource(t *testing.T) string {
	t.Helper()
	raw, err := fs.ReadFile(testTemplates(t), pluginTemplate)
	if err != nil {
		t.Fatalf("read plugin template: %v", err)
	}
	return string(raw)
}

func TestInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}

	pluginPath := filepath.Join(dir, ".opencode", "plugins", pluginName)
	installed, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if string(installed) != pluginScriptSource(t) {
		t.Error("installed plugin does not match the embedded template")
	}
}

func TestInstall_BinaryPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	binPath := filepath.Join(dir, ".opencode", "bin", "pulsarules_cli")
	stat, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if stat.Mode()&0o111 == 0 {
		t.Error("binary not executable")
	}
}

func TestInstall_GitignoreIgnoresBin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	giPath := filepath.Join(dir, ".opencode", ".gitignore")
	giContent, err := os.ReadFile(giPath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(giContent), "/bin/") {
		t.Error(".gitignore missing /bin/ entry")
	}
}

func TestUninstall_RemovesPluginAndGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}

	res, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !res.Removed {
		t.Error("expected Uninstall to report the plugin as removed")
	}

	for _, rel := range []string{
		filepath.Join(".opencode", "plugins"),
		filepath.Join(".opencode", "bin"),
	} {
		if _, statErr := os.Stat(filepath.Join(dir, rel)); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed, stat err = %v", rel, statErr)
		}
	}
	if _, statErr := os.Stat(
		filepath.Join(dir, ".opencode", ".gitignore"),
	); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected .gitignore to be removed once empty, stat err = %v", statErr)
	}
}

func TestUninstall_UntouchedDir(t *testing.T) {
	t.Parallel()

	res, err := Uninstall(t.TempDir())
	if err != nil {
		t.Fatalf("Uninstall on untouched dir: %v", err)
	}
	if res.Removed {
		t.Error("expected Uninstall on an untouched dir to report nothing removed")
	}
}

func TestUninstall_PluginFailureWording(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, ".opencode", "plugins", pluginName)
	if err := os.MkdirAll(pluginPath, disk.DirPrivate); err != nil {
		t.Fatalf("seed a directory in place of the plugin: %v", err)
	}

	_, err := Uninstall(dir)
	want := `uninstall plugin: check "` + pluginPath + `": read "` + pluginPath +
		`": read ` + pluginPath + `: is a directory`
	if err == nil || err.Error() != want {
		t.Fatalf("Uninstall err = %v, want %q", err, want)
	}
}

func TestUninstall_OrphanScanFailureIsBare(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err := os.Chmod(pluginsDir, 0o300); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(pluginsDir, disk.DirPrivate) })

	_, err := Uninstall(dir)
	want := `list backups of "` + filepath.Join(pluginsDir, pluginName) + `": scan "` +
		pluginsDir + `": open ` + pluginsDir + `: permission denied`
	if err == nil || err.Error() != want {
		t.Fatalf("Uninstall err = %v, want %q", err, want)
	}
}

func TestInstall_MissingTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	empty := os.DirFS(t.TempDir())
	if _, err := Install(dir, empty); err == nil {
		t.Fatal("expected an error when the plugin template is missing")
	}
}

func TestInstall_ForeignFileBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, disk.DirPrivate); err != nil {
		t.Fatalf("mkdir plugins: %v", err)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	const foreignContent = "// hand-authored plugin, not ours\n"
	if err := os.WriteFile(pluginPath, []byte(foreignContent), disk.File); err != nil {
		t.Fatalf("seed foreign plugin: %v", err)
	}

	ledger, err := Install(dir, testTemplates(t))
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(ledger.BackedUp) != 1 {
		t.Fatalf("backedUp = %v, want exactly one backup message", ledger.BackedUp)
	}

	backupData, err := os.ReadFile(pluginPath + ".pulsarules-backup")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupData) != foreignContent {
		t.Errorf("backup content = %q, want %q", backupData, foreignContent)
	}

	installedData, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read installed plugin: %v", err)
	}
	if string(installedData) != pluginScriptSource(t) {
		t.Error("installed plugin does not match the embedded template")
	}
}

func TestPluginScript_DispatchedHooksOnly(
	t *testing.T,
) {
	t.Parallel()
	script := pluginScriptSource(t)

	testCases := []struct {
		name    string
		substr  string
		wantHas bool
	}{
		{
			"registers experimental.chat.system.transform",
			`"experimental.chat.system.transform"`,
			true,
		},
		{"does not register tool.execute.before", `"tool.execute.before"`, false},
		{"registers tool.execute.after", `"tool.execute.after"`, true},
		{"does not register session.created", `"session.created"`, false},
		{"does not register session.idle", `"session.idle"`, false},
		{"does not register session.deleted", `"session.deleted"`, false},
		{"references the binary path", binaryRel, true},
		{"uses the tagged-template $ form", "$`${bin} hook ${mode}", true},
		{"does not use the broken array-call form", `$([bin, "hook", mode])`, false},
		{"reads opencode's session id", "input.sessionID", true},
		{"does not mint its own session id", "crypto.randomUUID()", false},
		{"pipes the payload to stdin", "Buffer.from(payload)", true},
		{"payload carries session_id", "session_id:", true},
		{"payload carries tool_input", "tool_input", true},
		{"payload carries file_path", "file_path", true},
		{"sets env on the shell command", ".env(", true},
		{"exports PULSARULES_PROJECT_DIR", "PULSARULES_PROJECT_DIR", true},
		{"exports PULSARULES_SKILLS_DIR", "PULSARULES_SKILLS_DIR", true},
		{"exports PULSARULES_LOG_PATH", "PULSARULES_LOG_PATH", true},
		{"guards a missing binary without a subprocess", "accessSync(bin", true},
		{"does not shell out to test -x", "test -x", false},
		{"logs caught errors", "console.error", true},
		{"has no empty catch block", "catch (e) {}", false},
		{"documents the governance-check gap", "// simplification:", true},
		{"transform bails out with no sessionID", "if (!input.sessionID) return;", true},
		{"transform pushes onto output.system", "output.system.push", true},
		{
			"transform calls the per-turn nudge mode every invocation",
			`runHook("user-prompt", input.sessionID)`,
			true,
		},
		{
			"post-edit reads the file path from input.args, not output.args",
			"input.args?.filePath",
			true,
		},
		{"post-edit never reads output.args", "output.args", false},
		{"post-edit appends context to output.output", "output.output", true},
		{"post-edit does not console.log the context", "console.log(ctx)", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			has := strings.Contains(script, testCase.substr)
			if has != testCase.wantHas {
				t.Errorf("Contains(%q) = %v, want %v", testCase.substr, has, testCase.wantHas)
			}
		})
	}
}

func TestUninstall_RestoresForeignPlugin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, ".opencode", "plugins", pluginName)
	if err := os.MkdirAll(filepath.Dir(pluginPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const foreign = "// a user's own plugin\n"
	if err := os.WriteFile(pluginPath, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign plugin: %v", err)
	}

	if _, err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	res, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !res.Removed || len(res.Restored) != 1 {
		t.Fatalf(
			"removed = %v, restored = %v, want a removal with one restore",
			res.Removed, res.Restored,
		)
	}
	got, err := os.ReadFile(pluginPath)
	if err != nil || string(got) != foreign {
		t.Errorf("plugin content = %q (%v), want the foreign original back", got, err)
	}
}

func TestPluginScript_CallsOnlyModesTheBinaryHandles(t *testing.T) {
	t.Parallel()

	handled := map[string]bool{}
	for _, event := range hook.Events() {
		handled[event.Mode] = true
	}
	source := pluginScriptSource(t)
	calls := regexp.MustCompile("runHook\\(\\s*[\"'`]([^\"'`]+)[\"'`]").
		FindAllStringSubmatch(source, -1)
	if len(calls) < 3 {
		t.Fatalf("found %d runHook calls in the plugin, want at least 3", len(calls))
	}
	total := len(regexp.MustCompile(`\brunHook\(`).FindAllString(source, -1)) -
		strings.Count(source, "function runHook(")
	if total != len(calls) {
		t.Errorf("plugin has %d runHook calls but %d use a literal mode", total, len(calls))
	}
	for _, call := range calls {
		if !handled[call[1]] {
			t.Errorf("plugin runs mode %q, which hook.Events() does not handle", call[1])
		}
	}
}
