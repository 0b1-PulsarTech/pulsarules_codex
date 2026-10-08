package opencodewire

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertRemoved(tb testing.TB, path string) {
	tb.Helper()

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("expected %q removed, stat err = %v", path, err)
	}
}

func assertSurvives(tb testing.TB, path string) {
	tb.Helper()

	if _, err := os.Stat(path); err != nil {
		tb.Errorf("expected %q to survive, stat err = %v", path, err)
	}
}

func TestRetireLegacyAgents_ForeignWarns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedContent string
		wantRemoved bool
		wantWarn    bool
	}{
		{
			name: "pre-migration render is removed",
			seedContent: "# AGENTS.md - demo\n\n## Mandatory routing contract\n\n" +
				legacyAgentsFingerprint + ".\n",
			wantRemoved: true,
		},
		{
			name:        "foreign content at the legacy path survives with a warning",
			seedContent: "# My own notes\nNothing to do with pulsarules_cli.\n",
			wantWarn:    true,
		},
		{
			name:        "missing legacy file is a no-op",
			seedContent: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			legacyPath := filepath.Join(projectDir, legacyAgentsPath)
			if testCase.seedContent != "" {
				seedFile(t, legacyPath, testCase.seedContent)
			}

			removed, warning, err := RetireLegacyAgents(projectDir)
			if err != nil {
				t.Fatalf("RetireLegacyAgents: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			if (warning != "") != testCase.wantWarn {
				t.Errorf("warning = %q, want non-empty=%v", warning, testCase.wantWarn)
			}
			if testCase.wantWarn && !strings.Contains(warning, legacyPath) {
				t.Errorf("warning %q does not name the offending file %q", warning, legacyPath)
			}
			switch {
			case testCase.seedContent == "":
			case testCase.wantRemoved:
				assertRemoved(t, legacyPath)
			default:
				assertSurvives(t, legacyPath)
			}
		})
	}
}

func TestRetireLegacyAgents_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	legacyPath := filepath.Join(projectDir, legacyAgentsPath)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o750); err != nil {
		t.Fatalf("mkdir .opencode: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte(legacyAgentsFingerprint), 0o600); err != nil {
		t.Fatalf("seed legacy AGENTS.md: %v", err)
	}

	if removed, _, err := RetireLegacyAgents(projectDir); err != nil || !removed {
		t.Fatalf("RetireLegacyAgents #1: removed=%v err=%v", removed, err)
	}
	if removed, _, err := RetireLegacyAgents(projectDir); err != nil || removed {
		t.Fatalf("RetireLegacyAgents #2: removed=%v err=%v", removed, err)
	}
}

func TestRetireLegacyAgents_PathIsDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	legacyPath := filepath.Join(projectDir, legacyAgentsPath)
	if err := os.MkdirAll(legacyPath, 0o750); err != nil {
		t.Fatalf("seed directory in place of the legacy AGENTS.md: %v", err)
	}

	if _, _, err := RetireLegacyAgents(projectDir); err == nil {
		t.Error("expected an error reading a directory as the legacy AGENTS.md, got nil")
	}
}
