package hook

import (
	"os"
	"testing"
)

//nolint:paralleltest // t.Chdir moves the whole process.
func TestDispatcher_SkillInstalledSkipsEmptyEntries(t *testing.T) {
	cwd, otherDir := t.TempDir(), t.TempDir()
	installSkillFixture(t, cwd, "only-in-cwd")
	t.Chdir(cwd)

	disp := NewDispatcher(Deps{SkillsDir: string(os.PathListSeparator) + otherDir})
	if disp.skillInstalled("only-in-cwd") {
		t.Error("skillInstalled found a skill through an empty entry, via the working directory")
	}
}
