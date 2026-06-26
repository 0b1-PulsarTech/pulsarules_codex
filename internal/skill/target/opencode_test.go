package target

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestOpencodeTargetName(t *testing.T) {
	t.Parallel()
	if got := (opencodeTarget{}).Name(); got != "opencode" {
		t.Fatalf("Name() = %q, want opencode", got)
	}
}

func TestOpencodeTargetInstall(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})

	report, err := opencodeTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	wantPaths := []string{
		filepath.Join(base, ".opencode", "skills", "go-style", "SKILL.md"),
		filepath.Join(base, ".opencode", "AGENTS.md"),
		filepath.Join(base, "opencode.json"),
	}
	for _, path := range wantPaths {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("missing %q: %v", path, statErr)
		}
	}
	if !slices.ContainsFunc(report.Notes, func(note string) bool {
		return strings.Contains(note, "wired opencode")
	}) {
		t.Errorf("Notes missing 'wired opencode': %v", report.Notes)
	}
}
