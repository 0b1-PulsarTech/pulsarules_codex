package opencodehook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir); err != nil {
		t.Fatalf("Install: %v", err)
	}

	pluginPath := filepath.Join(dir, ".opencode", "plugins", pluginName)
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if !strings.Contains(string(data), "PulsarulesGovernance") {
		t.Error("plugin missing PulsarulesGovernance export")
	}
	if !strings.Contains(string(data), "session.created") {
		t.Error("plugin missing session.created hook")
	}
	if !strings.Contains(string(data), "tool.execute.before") {
		t.Error("plugin missing tool.execute.before hook")
	}
	if !strings.Contains(string(data), "tool.execute.after") {
		t.Error("plugin missing tool.execute.after hook")
	}
	if !strings.Contains(string(data), "session.idle") {
		t.Error("plugin missing session.idle hook")
	}
}

func TestInstall_BinaryPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := Install(dir); err != nil {
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
	if err := Install(dir); err != nil {
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

func TestPluginScript_ContainsBinaryPath(t *testing.T) {
	t.Parallel()

	if !strings.Contains(pluginScript, binaryRel) {
		t.Error("plugin script does not reference the binary path")
	}
}
