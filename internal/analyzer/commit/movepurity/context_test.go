package movepurity

import (
	"maps"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestDeclared(t *testing.T) {
	t.Parallel()

	head := map[string]string{
		"d/a.go": "package d\n\nfunc Top() {}\n\ntype T struct{}\n\nconst (\n" +
			"\tC1 = 1\n\tC2 = 2\n)\n\nvar V, W int\n",
		"d/m.go":      "package d\n\nfunc (T) Method() {}\n",
		"d/t_test.go": "package d\n\nfunc InTest() {}\n",
		"d/moved.go":  "package d\n\nfunc Moved() {}\n",
		"d/gone.go":   "package d\n\nfunc Gone() {}\n",
		"d/other.go":  "package other\n\nfunc OtherPkg() {}\n",
		"d/bad.go":    "package d\n\nfunc (\n",
		"d/note.txt":  "func NotGo() {}\n",
		"d/edit.go":   "package d\n\nfunc HeadOnly() {}\n",
	}
	staged := maps.Clone(head)
	delete(staged, "d/gone.go")
	delete(staged, "d/moved.go")
	staged["e/moved.go"] = head["d/moved.go"]
	staged["d/edit.go"] = "package d\n\nfunc StagedOnly() {}\n"
	staged["d/added.go"] = "package d\n\nfunc Added() {}\n"
	moves := newMoveContext(
		newFakeRepo(head, staged),
		[]pass.Rename{pair("d/moved.go", "e/moved.go")},
	)

	testCases := []struct {
		name string
		want map[string]bool
		not  []string
	}{
		{
			name: "same package top-level names only",
			want: map[string]bool{
				"Top":      true,
				"T":        true,
				"C1":       true,
				"C2":       true,
				"V":        true,
				"W":        true,
				"HeadOnly": true,
			},
			not: []string{
				"Method",
				"InTest",
				"Moved",
				"Gone",
				"OtherPkg",
				"Added",
				"NotGo",
				"StagedOnly",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := moves.declaredIn("d", "d")
			for name := range testCase.want {
				if !got[name] {
					t.Fatalf("declared[%s] = false, want true", name)
				}
			}
			for _, name := range testCase.not {
				if got[name] {
					t.Fatalf("declared[%s] = true, want false", name)
				}
			}
		})
	}
}

func TestMoveInfos(t *testing.T) {
	t.Parallel()

	head := map[string]string{
		"a/x.go":      "package a\n\nfunc One() {}\n",
		"a/y.go":      "package a\n\nvar Two int\n",
		"a/z_test.go": "package a\n\nfunc InTest() {}\n",
		"a/doc.txt":   "text\n",
		"a/gone.go":   "package a\n",
	}
	staged := map[string]string{
		"b/x.go": "package b\n\nfunc One() {}\n", "b/y.go": "package b\n\nvar Two int\n",
		"b/z_test.go": "package b\n", "b/doc.txt": "text\n",
	}
	moves := newMoveContext(newFakeRepo(head, staged), []pass.Rename{
		pair("a/x.go", "b/x.go"), pair("a/y.go", "b/y.go"), pair("a/z_test.go", "b/z_test.go"),
		pair("a/doc.txt", "b/doc.txt"), pair("a/gone.go", "b/gone.go"),
	})

	pairs := moves.moveInfos()
	if len(pairs) != 1 {
		t.Fatalf("moveInfos() = %+v, want one directory pair", pairs)
	}
	moved := pairs[0]
	if moved.OldDir != "a" || moved.NewDir != "b" || moved.OldPkg != "a" || moved.NewPkg != "b" ||
		moved.OldFull != "example.com/m/a" || moved.NewFull != "example.com/m/b" {
		t.Fatalf("moved = %+v, want a -> b with packages a and b", moved)
	}
	for name, want := range map[string]bool{"One": true, "Two": true, "InTest": false} {
		if moved.Decls[name] != want {
			t.Fatalf("decls[%s] = %v, want %v", name, moved.Decls[name], want)
		}
	}
}

func TestModuleOf_WalksUp(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo(
		map[string]string{"sub/go.mod": "module example.com/sub\n"},
		map[string]string{},
	)
	testCases := []struct {
		name   string
		dir    string
		isHead bool
		want   string
	}{
		{"nested module wins", "sub/pkg/deep", true, "example.com/sub/pkg/deep"},
		{"falls to the root module", "other/pkg", true, "example.com/m/other/pkg"},
		{"staged side has no nested go.mod", "sub/pkg", false, "example.com/m/sub/pkg"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(repo, nil)
			got := moves.staged.module(testCase.dir)
			if testCase.isHead {
				got = moves.head.module(testCase.dir)
			}
			if gotPath := got.ImportPath(testCase.dir); gotPath != testCase.want {
				t.Fatalf("ImportPath(%q) = %q, want %q", testCase.dir, gotPath, testCase.want)
			}
		})
	}
}

func TestModuleOf_NoGoMod(t *testing.T) {
	t.Parallel()

	moves := newMoveContext(emptyRepo{}, nil)
	if got := moves.head.module("a/b"); got != (gosrc.Module{}) {
		t.Fatalf("moduleOf() = %+v, want the zero module", got)
	}
}

func TestMoveInfos_SamePackageNameTwice(t *testing.T) {
	t.Parallel()

	comment := func(ref string) string {
		return "package use\n\n// see " + ref + "\ntype A struct{}\n"
	}
	row := func(name, newRef string, want int) repoCase {
		return repoCase{
			name: name,
			head: map[string]string{
				"a/util/n.go": "package util\n\nfunc New() int { return 1 }\n",
				"b/util/n.go": "package util\n\nfunc New() int { return 1 }\n",
				"use/f.go":    comment("util.New"),
			},
			staged: map[string]string{
				"x/xa/n.go": "package xa\n\nfunc New() int { return 1 }\n",
				"y/yb/n.go": "package yb\n\nfunc New() int { return 1 }\n",
				"use/f.go":  comment(newRef),
			},
			renames: []pass.Rename{
				pair("a/util/n.go", "x/xa/n.go"),
				pair("b/util/n.go", "y/yb/n.go"),
			},
			changed:      []pass.FileChange{{Path: "use/f.go", Staged: true}},
			wantFindings: want,
		}
	}
	runRepoCases(t, []repoCase{
		row("qualifier follows the first move", "xa.New", 0),
		row("qualifier follows the second move", "yb.New", 0),
		row("qualifier no move maps to", "zz.New", 1),
	})
}
