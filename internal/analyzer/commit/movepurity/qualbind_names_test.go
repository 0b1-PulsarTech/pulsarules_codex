package movepurity

import (
	"maps"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestMovedPackage_BySide(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo(map[string]string{}, map[string]string{})
	moves := &moveContext{infos: []gosrc.MoveInfo{{
		OldFull: "example.com/m/a", NewFull: "example.com/m/b", OldPkg: "alpha", NewPkg: "beta",
	}}}
	testCases := []struct {
		name       string
		side       vcs.Side
		importPath string
		want       string
	}{
		{"head names the old package", vcs.Head, "example.com/m/a", "alpha"},
		{"head ignores the new path", vcs.Head, "example.com/m/b", ""},
		{"staged names the new package", vcs.Staged, "example.com/m/b", "beta"},
		{"staged ignores the old path", vcs.Staged, "example.com/m/a", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			at := nameSide{snap: newSnapshot(repo, testCase.side)}
			if got := moves.movedPackage(testCase.importPath, at); got != testCase.want {
				t.Fatalf("movedPackage() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestImportNames_Clashes(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo(map[string]string{}, map[string]string{})
	moves := &moveContext{infos: []gosrc.MoveInfo{}}
	at := nameSide{snap: newSnapshot(repo, vcs.Head)}
	testCases := []struct {
		name        string
		imports     []gosrc.ImportSpec
		wantClashes map[string]bool
		wantPaths   map[string]string
	}{
		{"no imports", nil, map[string]bool{}, map[string]string{}},
		{
			"distinct names",
			[]gosrc.ImportSpec{{Name: "x", Path: "m/a"}, {Name: "y", Path: "m/b"}},
			map[string]bool{},
			map[string]string{"x": "m/a", "y": "m/b"},
		},
		{
			"one name bound twice",
			[]gosrc.ImportSpec{{Name: "x", Path: "m/a"}, {Name: "x", Path: "m/b"}},
			map[string]bool{"x": true},
			map[string]string{"x": "m/a"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := moves.importNames(testCase.imports, at)
			if !maps.Equal(got.clashes, testCase.wantClashes) {
				t.Fatalf("clashes = %v, want %v", got.clashes, testCase.wantClashes)
			}
			if !maps.Equal(got.paths, testCase.wantPaths) {
				t.Fatalf("paths = %v, want %v", got.paths, testCase.wantPaths)
			}
		})
	}
}

func TestAmbiguousNames_UnionsBothSides(t *testing.T) {
	t.Parallel()

	got := ambiguousNames(
		importedNames{clashes: map[string]bool{"a": true}},
		importedNames{clashes: map[string]bool{"b": true}},
	)
	if want := map[string]bool{"a": true, "b": true}; !maps.Equal(got, want) {
		t.Fatalf("ambiguousNames() = %v, want %v", got, want)
	}
}
