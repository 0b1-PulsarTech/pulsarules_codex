package output

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestPackage_ZipsEverySkillIncludingRouter(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	out := filepath.Join(t.TempDir(), "skills.zip")

	if err := Package(idx, rnd, out); err != nil {
		t.Fatalf("Package: %v", err)
	}
	reader, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = reader.Close() }()

	if len(reader.File) != len(idx.Skills) {
		t.Errorf("zip has %d entries, want %d", len(reader.File), len(idx.Skills))
	}
	var foundRouter bool
	for _, file := range reader.File {
		if file.Name == "project-router/SKILL.md" {
			foundRouter = true
		}
	}
	if !foundRouter {
		t.Error("zip missing project-router/SKILL.md")
	}
}

func TestPackage_MidLoopFailureLeavesNoFile(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	idx.Skills = append(idx.Skills, knowledge.Skill{
		ID:               "zzz-broken-fixture",
		Order:            999999,
		ComposeWorkflows: []string{"does-not-exist"},
	})
	out := filepath.Join(t.TempDir(), "skills.zip")

	err := Package(idx, rnd, out)
	if err == nil {
		t.Fatal("Package: expected an error from the broken fixture skill")
	}

	if _, statErr := os.Stat(out); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected no file at %q after a failed Package, stat err = %v", out, statErr)
	}
}

func TestPackage_FailsWhenOutputCannotBeCreated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		outPath func(tb testing.TB, root string) string
		wantErr string
	}{
		{
			name: "parent is a regular file",
			outPath: func(tb testing.TB, root string) string {
				tb.Helper()
				seedFile(tb, filepath.Join(root, "blocker"), "x")
				return filepath.Join(root, "blocker", "skills.zip")
			},
			wantErr: "mkdir: ",
		},
		{
			name: "output path is a directory",
			outPath: func(tb testing.TB, root string) string {
				tb.Helper()
				if err := os.MkdirAll(filepath.Join(root, "skills.zip"), 0o750); err != nil {
					tb.Fatalf("mkdir: %v", err)
				}
				return filepath.Join(root, "skills.zip")
			},
			wantErr: "create ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			idx, rnd := loadFixture(t)
			outPath := testCase.outPath(t, t.TempDir())

			err := Package(idx, rnd, outPath)
			if err == nil || !strings.HasPrefix(err.Error(), testCase.wantErr) {
				t.Fatalf("err = %v, want prefix %q", err, testCase.wantErr)
			}
		})
	}
}
