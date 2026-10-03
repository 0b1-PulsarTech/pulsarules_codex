package filecount

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestCountByDirOnlyWantedDirs(t *testing.T) {
	t.Parallel()

	ctx := buildCtx(t, merged(numbered("a", 2), numbered("b", 3)), nil, nil)
	eligible := func(pass.FileChange) bool { return true }
	got := countByDir(ctx, eligible, map[string]bool{"a": true})
	if len(got) != 1 || got["a"].files != 2 {
		t.Fatalf("counts = %+v, want only a with 2 files", got)
	}
}

func TestSummarizeTieBreaksByName(t *testing.T) {
	t.Parallel()

	infos := []srcInfo{{pkg: "b"}, {pkg: "b"}, {pkg: "a", ent: true}, {pkg: "a", ent: true}}
	want := dirCount{files: 2, isEntSchema: true}
	for range 64 {
		if got := summarize(infos); got != want {
			t.Fatalf("summarize = %+v, want %+v", got, want)
		}
	}
}
