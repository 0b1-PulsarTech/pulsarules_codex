package output

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestPackage zips every skill as "<id>/SKILL.md"; the archive must be readable
// and contain the router entry.
func TestPackage(t *testing.T) {
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

// TestPackage_MidLoopFailureLeavesNoFile proves a render failure partway
// through the skill list closes the zip writer and archive, then removes the
// partial file, so a failed package run never leaves a corrupt zip at outPath.
func TestPackage_MidLoopFailureLeavesNoFile(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	// Sorts after every real skill (SkillsOrdered orders by Order then ID), so
	// several entries are written to the archive before this one fails.
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
