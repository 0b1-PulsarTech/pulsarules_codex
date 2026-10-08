package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
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

func TestRemoveDocs_ReportsRestoredAndOrphaned(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		docName string
		remove  func(dest string, report *Report) error
	}{
		{name: "skills", docName: "SKILL.md", remove: removeSkills},
		{name: "workflows", docName: "WORKFLOW.md", remove: removeWorkflows},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dest := t.TempDir()
			dir := filepath.Join(dest, "security")
			docPath := filepath.Join(dir, testCase.docName)
			seedFile(t, docPath, "# my own notes, no marker\n")
			body := "# body\n" + marker.Installed + "\n"
			if _, err := output.WriteDoc(dir, testCase.docName, body); err != nil {
				t.Fatalf("WriteDoc: %v", err)
			}
			seedFile(t, docPath+marker.BackupSuffix+".2", "older\n")

			var report Report
			if err := testCase.remove(dest, &report); err != nil {
				t.Fatalf("remove: %v", err)
			}
			assertNote(t, report.Notes, marker.RestoreMessage(docPath))
			assertNote(t, report.Warnings, docPath+marker.BackupSuffix+".2")
		})
	}
}

func TestInstallSkills_PassesRouterFilterToRouter(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	ctx := newTestContext(t, t.TempDir(), []string{"project-router"})
	ctx.RouterFilter = []string{"go-style"}

	var report Report
	if err := installSkills(ctx, dest, &report); err != nil {
		t.Fatalf("installSkills: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dest, "project-router", "SKILL.md"))
	if err != nil {
		t.Fatalf("read router: %v", err)
	}
	if strings.Contains(string(body), "database-persistence") {
		t.Error("router lists database-persistence, want it dropped by the filter")
	}
	if !strings.Contains(string(body), "go-style") {
		t.Error("router lacks go-style, want the filtered-in skill listed")
	}
}
