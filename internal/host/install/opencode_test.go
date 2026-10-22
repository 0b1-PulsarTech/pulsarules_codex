package install

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func TestUninstallOpencode_UntouchedDir(t *testing.T) {
	t.Parallel()

	rpt, err := UninstallOpencode(t.TempDir())
	if err != nil {
		t.Fatalf("Uninstall(opencode) on untouched dir: %v", err)
	}
	if len(rpt.Notes) != 0 {
		t.Errorf("Notes = %v, want none for untouched dir", rpt.Notes)
	}
}

func TestOpencode_InstallThenUninstallNotes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	installed, err := Opencode(dir, testTemplates(t))
	if err != nil {
		t.Fatalf("Opencode: %v", err)
	}
	wantInstall := "installed opencode governance plugin: " + pluginsDir
	if len(installed.Notes) != 1 || installed.Notes[0] != wantInstall {
		t.Errorf("install Notes = %v, want [%q]", installed.Notes, wantInstall)
	}

	removed, err := UninstallOpencode(dir)
	if err != nil {
		t.Fatalf("UninstallOpencode: %v", err)
	}
	wantRemove := "removed opencode governance plugin: " + pluginsDir
	if len(removed.Notes) != 1 || removed.Notes[0] != wantRemove {
		t.Errorf("uninstall Notes = %v, want [%q]", removed.Notes, wantRemove)
	}
}

func TestOpencode_FailureWording(t *testing.T) {
	t.Parallel()

	_, err := Opencode(t.TempDir(), fstest.MapFS{})
	want := "install opencode: install opencode hook: read plugin template: " +
		"open hooks/opencode-plugin.js: file does not exist"
	if err == nil || err.Error() != want {
		t.Fatalf("Opencode err = %v, want %q", err, want)
	}

	dir := t.TempDir()
	pluginPath := filepath.Join(dir, ".opencode", "plugins", "pulsarules-governance.js")
	if err = os.MkdirAll(pluginPath, disk.ModeDirPrivate); err != nil {
		t.Fatalf("seed a directory in place of the plugin: %v", err)
	}
	_, err = UninstallOpencode(dir)
	want = `uninstall opencode: uninstall opencode hook: uninstall plugin: check "` +
		pluginPath + `": read "` + pluginPath + `": read ` + pluginPath + `: is a directory`
	if err == nil || err.Error() != want {
		t.Fatalf("UninstallOpencode err = %v, want %q", err, want)
	}
}
