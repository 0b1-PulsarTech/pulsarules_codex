package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_WritesEverySkillIncludingRouter(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	out := t.TempDir()

	written, err := Generate(idx, rnd, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(written) != len(idx.Skills) {
		t.Errorf("written %d skills, want %d", len(written), len(idx.Skills))
	}
	if _, err = os.Stat(filepath.Join(out, "skills", "project-router", "SKILL.md")); err != nil {
		t.Errorf("router not generated: %v", err)
	}
}
