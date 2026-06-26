package hookwire

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallHook copies the script (executable) and README into <claudeDir>/hooks.
func TestInstallHook(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	info, err := os.Stat(script)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("script not executable: mode %v", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(claudeDir, "hooks", "README.md")); err != nil {
		t.Errorf("README not installed: %v", err)
	}
}
