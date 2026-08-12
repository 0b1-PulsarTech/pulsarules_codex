package hookwire

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestUninstallHook(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	if _, err := UninstallHook(claudeDir); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	for _, dir := range []string{"hooks", "bin"} {
		if _, err := os.Stat(filepath.Join(claudeDir, dir)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed once empty, stat err = %v", dir, err)
		}
	}
}

func TestUninstallHook_LeavesUnrelatedContent(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	extra := filepath.Join(claudeDir, "hooks", "user-hook.sh")
	if err := os.WriteFile(
		extra,
		[]byte("#!/bin/sh\n"),
		0o755,
	); err != nil { //nolint:gosec // test fixture.
		t.Fatalf("seed extra file: %v", err)
	}

	if _, err := UninstallHook(claudeDir); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	if _, err := os.Stat(extra); err != nil {
		t.Errorf("expected unrelated hook file to survive, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(claudeDir, "hooks")); err != nil {
		t.Errorf("expected hooks dir to survive (not empty), stat err = %v", err)
	}
}

func TestUninstallHook_ForeignReadme(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	readme := filepath.Join(claudeDir, "hooks", "README.md")
	foreign := "# My own notes\nDo not touch.\n"
	if err := os.WriteFile(readme, []byte(foreign), 0o600); err != nil {
		t.Fatalf("overwrite foreign readme: %v", err)
	}

	if _, err := UninstallHook(claudeDir); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	got, readErr := os.ReadFile(readme) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("expected foreign README.md to survive, read err = %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("README.md content changed, got %q want %q", got, foreign)
	}
	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	if _, statErr := os.Stat(script); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected marker-carrying script to be removed, stat err = %v", statErr)
	}
}

func TestUninstallHook_RestoresBackup(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readme := filepath.Join(hooksDir, "README.md")
	foreign := "# My own notes about this hooks dir\n"
	if err := os.WriteFile(readme, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign readme: %v", err)
	}
	if _, err := InstallHook(fakeTemplates(), claudeDir); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	restored, err := UninstallHook(claudeDir)
	if err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}
	wantMsg := marker.RestoreMessage(readme)
	if len(restored) != 1 || restored[0] != wantMsg {
		t.Fatalf("restored = %v, want [%q]", restored, wantMsg)
	}
	got, readErr := os.ReadFile(readme) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read restored readme: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("restored content = %q, want %q", got, foreign)
	}
}

func TestUninstallHook_Idempotent(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := UninstallHook(claudeDir); err != nil {
		t.Fatalf("UninstallHook on untouched dir: %v", err)
	}
}

func TestUninstallHook_ForeignAssetSurvives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		asset      string
		content    string
		wantExists bool
	}{
		{
			name:       "foreign README survives",
			asset:      "README.md",
			content:    "# My own notes about this hooks dir\n",
			wantExists: true,
		},
		{
			name:  "installed README is removed",
			asset: "README.md",
			content: "<!-- Installed by pulsarules_cli; " +
				"remove or edit this file to disable. -->\n",
			wantExists: false,
		},
		{
			name:       "foreign script survives",
			asset:      "skill-router-reminder.sh",
			content:    "#!/usr/bin/env bash\necho mine\n",
			wantExists: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			claudeDir := t.TempDir()
			hooksDir := filepath.Join(claudeDir, "hooks")
			if err := os.MkdirAll(hooksDir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			assetPath := filepath.Join(hooksDir, testCase.asset)
			if err := os.WriteFile(assetPath, []byte(testCase.content), 0o600); err != nil {
				t.Fatalf("write asset: %v", err)
			}

			if _, err := UninstallHook(claudeDir); err != nil {
				t.Fatalf("UninstallHook: %v", err)
			}

			_, err := os.Stat(assetPath)
			if testCase.wantExists && err != nil {
				t.Errorf(
					"%s was deleted; a file this tool did not write must survive",
					testCase.asset,
				)
			}
			if !testCase.wantExists && err == nil {
				t.Errorf("%s survived; an installed asset must be removed", testCase.asset)
			}
		})
	}
}
