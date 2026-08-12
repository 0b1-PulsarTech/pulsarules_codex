package opencode

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestInstall_MissingTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	empty := os.DirFS(t.TempDir())
	if err := Install(dir, empty); err == nil {
		t.Fatal("expected an error when the plugin template is missing")
	}
}

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
