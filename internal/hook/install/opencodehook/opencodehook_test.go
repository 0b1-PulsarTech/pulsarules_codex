package opencodehook

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// testTemplates loads the real embedded templates FS so plugin-content
// assertions exercise the production script, not a test stand-in.
func testTemplates(t *testing.T) fs.FS {
	t.Helper()
	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return templates
}

// pluginScriptSource reads the production plugin script straight out of the
// embedded templates FS. Content assertions use this instead of running the
// full Install (mkdir + write + selfbin.Copy + gitignore.Ensure): they only
// need the bytes, and integration-tests requires unit tests to stay I/O-free.
func pluginScriptSource(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(testTemplates(t), pluginTemplate)
	if err != nil {
		t.Fatalf("read plugin template: %v", err)
	}
	return string(data)
}

func TestInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}

	pluginPath := filepath.Join(dir, ".opencode", "plugins", pluginName)
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if string(data) != pluginScriptSource(t) {
		t.Error("installed plugin does not match the embedded template")
	}
}

func TestInstall_BinaryPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	binPath := filepath.Join(dir, ".opencode", "bin", "pulsarules_cli")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("binary not executable")
	}
}

func TestInstall_GitignoreIgnoresBin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}
	giPath := filepath.Join(dir, ".opencode", ".gitignore")
	data, err := os.ReadFile(giPath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), "/bin/") {
		t.Error(".gitignore missing /bin/ entry")
	}
}

// TestUninstall asserts Install then Uninstall removes the plugin, the
// binary, and the gitignore entry, and cleans up the now-empty directories.
func TestUninstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir, testTemplates(t)); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !removed {
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
	// /bin/ was the only entry Install ever added, so Uninstall's Remove
	// deletes the .gitignore file entirely rather than leave it empty.
	if _, statErr := os.Stat(
		filepath.Join(dir, ".opencode", ".gitignore"),
	); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected .gitignore to be removed once empty, stat err = %v", statErr)
	}
}

// TestUninstall_Idempotent asserts running Uninstall against a directory
// Install never touched is not an error, and reports nothing was removed.
func TestUninstall_Idempotent(t *testing.T) {
	t.Parallel()

	removed, err := Uninstall(t.TempDir())
	if err != nil {
		t.Fatalf("Uninstall on untouched dir: %v", err)
	}
	if removed {
		t.Error("expected Uninstall on an untouched dir to report nothing removed")
	}
}

func TestInstall_MissingTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	empty := os.DirFS(t.TempDir())
	if err := Install(dir, empty); err == nil {
		t.Fatal("expected an error when the plugin template is missing")
	}
}

// TestPluginScript pins the production script's content against every
// verified opencode plugin-API fact this redesign relies on: it registers
// only hook names opencode's trigger() actually dispatches, and it does NOT
// register session.created/session.idle/session.deleted - bus-event names
// that look like hooks but never fire, which is the regression this test
// suite exists to catch.
func TestPluginScript(t *testing.T) {
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
		{"guards a missing binary without a subprocess", "accessSync(bin", true},
		{"does not shell out to test -x", "test -x", false},
		{"logs caught errors", "console.error", true},
		{"has no empty catch block", "catch (e) {}", false},
		{"documents the governance-check gap", "// simplification:", true},
		{"transform bails out with no sessionID", "if (!input.sessionID) return;", true},
		{"transform pushes onto output.system", "output.system.push", true},
		{
			"transform calls the per-turn digest mode every invocation",
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
