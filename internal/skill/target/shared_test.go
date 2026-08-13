package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestInstallSkillsAndWorkflows_Backups(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		ids     []string
		docName string
		first   string
		blocked string
		install func(ctx Context, dest string, report *Report) error
		wantErr string
	}{
		{
			name:    "skills",
			ids:     []string{"go-style", "code-minimalism"},
			docName: "SKILL.md",
			first:   "go-style",
			blocked: "code-minimalism",
			install: installSkills,
			wantErr: "render skills to ",
		},
		{
			name:    "workflows",
			ids:     []string{"project-router", "refactoring"},
			docName: "WORKFLOW.md",
			first:   "feature-development",
			blocked: "refactoring",
			install: installWorkflows,
			wantErr: "render workflows to ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dest := t.TempDir()
			foreign := filepath.Join(dest, testCase.first, testCase.docName)
			seedFile(t, foreign, "# my own notes, no marker\n")
			seedFile(t, filepath.Join(dest, testCase.blocked), "not a directory")
			ctx := newTestContext(t, t.TempDir(), testCase.ids)

			var report Report
			err := testCase.install(ctx, dest, &report)
			if err == nil {
				t.Fatal("expected an error from the blocked id")
			}
			assertNote(t, []string{err.Error()}, testCase.wantErr)
			wantBackup := marker.BackupMessage(foreign, foreign+marker.BackupSuffix)
			assertNote(t, report.Warnings, wantBackup)
		})
	}
}

func seedFile(tb testing.TB, filePath, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", filePath, err)
	}
}

func assertNote(tb testing.TB, entries []string, substr string) {
	tb.Helper()
	if !strings.Contains(strings.Join(entries, "\n"), substr) {
		tb.Errorf("entries = %v, want one containing %q", entries, substr)
	}
}
