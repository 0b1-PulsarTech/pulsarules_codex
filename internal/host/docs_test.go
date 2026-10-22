package host

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
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
			wantBackup := ownedfile.BackupMessage(foreign, foreign+ownedfile.BackupSuffix)
			assertNote(t, report.Warnings, wantBackup)
		})
	}
}

func TestRemoveSkillsAndWorkflows_ReportsRestoredAndOrphaned(t *testing.T) {
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
			body := "# body\n" + ownedfile.Marker + "\n"
			if _, err := docs.Write(dir, testCase.docName, body); err != nil {
				t.Fatalf("Write: %v", err)
			}
			seedFile(t, docPath+ownedfile.BackupSuffix+".2", "older\n")

			var report Report
			if err := testCase.remove(dest, &report); err != nil {
				t.Fatalf("remove: %v", err)
			}
			assertNote(t, report.Notes, ownedfile.RestoreMessage(docPath))
			assertNote(t, report.Warnings, docPath+ownedfile.BackupSuffix+".2")
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

func TestReportInstalled_KindWording(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		kind         docKind
		wantNotes    []string
		wantWarnings []string
	}{
		{
			name:         "skills",
			kind:         skillKind(),
			wantNotes:    []string{"installed: " + filepath.Join("dest", "go-style")},
			wantWarnings: []string{"skipped (unknown skill): nope"},
		},
		{
			name:         "workflows",
			kind:         workflowKind(),
			wantNotes:    []string{"installed workflow: " + filepath.Join("dest", "go-style")},
			wantWarnings: []string{"skipped workflow (unknown): nope"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var report Report
			result := docs.InstallResult{
				Installed: []string{"go-style"},
				Skipped:   []string{"nope"},
			}
			if err := reportInstalled(testCase.kind, "dest", result, nil, &report); err != nil {
				t.Fatalf("reportInstalled: %v", err)
			}
			if !slices.Equal(report.Notes, testCase.wantNotes) ||
				!slices.Equal(report.Warnings, testCase.wantWarnings) {
				t.Errorf("report = %+v, want notes %q warnings %q",
					report, testCase.wantNotes, testCase.wantWarnings)
			}
		})
	}
}

func TestRemoveKind_KindWording(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		kind     docKind
		wantNote string
	}{
		{name: "skills", kind: skillKind(), wantNote: "removed: "},
		{name: "workflows", kind: workflowKind(), wantNote: "removed workflow: "},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dest := t.TempDir()
			dir := filepath.Join(dest, "security")
			body := "# body\n" + ownedfile.Marker + "\n"
			if _, err := docs.Write(dir, testCase.kind.docName, body); err != nil {
				t.Fatalf("Write: %v", err)
			}

			var report Report
			if err := removeKind(testCase.kind, dest, &report); err != nil {
				t.Fatalf("removeKind: %v", err)
			}
			if !slices.Equal(report.Notes, []string{testCase.wantNote + dir}) {
				t.Errorf("Notes = %q, want %q", report.Notes, testCase.wantNote+dir)
			}
		})
	}
}
