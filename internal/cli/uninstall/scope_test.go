package uninstall

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func TestRun_ProjectNeverTouchesTheUserDirs(t *testing.T) {
	t.Parallel()

	home, configDir, projectDir := t.TempDir(), t.TempDir(), t.TempDir()
	wired := testWiring(t)
	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}

	cmd := New(flags.Environment{Home: home, ConfigDir: configDir})
	cmd.project = projectDir
	if err := cmd.Run(wired.targets); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, statErr := os.Stat(
		filepath.Join(projectDir, ".claude"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("the project's own .claude survived the uninstall, stat err = %v", statErr)
	}
	for _, dir := range []string{home, configDir} {
		if entries, readErr := os.ReadDir(dir); readErr != nil || len(entries) != 0 {
			t.Errorf(
				"%q = %v, %v, want a project uninstall to leave it empty",
				dir,
				entries,
				readErr,
			)
		}
	}
}
