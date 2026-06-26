package target

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeTargetName(t *testing.T) {
	t.Parallel()
	if got := (claudeTarget{}).Name(); got != "claude" {
		t.Fatalf("Name() = %q, want claude", got)
	}
}

func TestClaudeTargetInstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		ids          []string
		wantFiles    []string
		absentFiles  []string
		wantNotes    int
		wantWarnings int
	}{
		{
			name:      "single skill",
			ids:       []string{"go-style"},
			wantFiles: []string{filepath.Join("go-style", "SKILL.md")},
			wantNotes: 1,
		},
		{
			name: "router and skill",
			ids:  []string{"project-router", "go-style"},
			wantFiles: []string{
				filepath.Join("project-router", "SKILL.md"),
				filepath.Join("go-style", "SKILL.md"),
			},
			// project-router composes the feature-development workflow (+1 note).
			wantNotes: 3,
		},
		{
			name:         "unknown skill is skipped, not fatal",
			ids:          []string{"nonexistent-skill"},
			absentFiles:  []string{filepath.Join("nonexistent-skill", "SKILL.md")},
			wantWarnings: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			skills := filepath.Join(base, ".claude", "skills")
			ctx := newTestContext(t, base, testCase.ids)

			report, err := claudeTarget{}.Install(ctx)
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			for _, rel := range testCase.wantFiles {
				if _, statErr := os.Stat(filepath.Join(skills, rel)); statErr != nil {
					t.Errorf("missing rendered skill %q: %v", rel, statErr)
				}
			}
			for _, rel := range testCase.absentFiles {
				if _, statErr := os.Stat(filepath.Join(skills, rel)); statErr == nil {
					t.Errorf("expected %q to be absent", rel)
				}
			}
			if len(report.Notes) != testCase.wantNotes {
				t.Errorf(
					"Notes = %d, want %d (%v)",
					len(report.Notes),
					testCase.wantNotes,
					report.Notes,
				)
			}
			if len(report.Warnings) != testCase.wantWarnings {
				t.Errorf(
					"Warnings = %d, want %d (%v)",
					len(report.Warnings),
					testCase.wantWarnings,
					report.Warnings,
				)
			}
		})
	}
}
