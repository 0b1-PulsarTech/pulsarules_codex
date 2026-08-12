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

func TestPluginScript_ContainsBinaryPath(t *testing.T) {
	t.Parallel()

	if !strings.Contains(pluginScriptSource(t), binaryRel) {
		t.Error("plugin script does not reference the binary path")
	}
}
