package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestPackageAllowed_Rejected(t *testing.T) {
	t.Parallel()

	const sample = "\nfunc Sample() int { return 1 }\n"
	utilBody := "package util\n" + sample
	routingTree := map[string]string{
		"internal/routing/r.go": "package routing\n\nfunc R() int { return 2 }\n",
	}
	moveA := []pass.Rename{pair(utilA, routingA)}
	extTest := "package util_test\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"
	intTest := "package util\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"
	testCases := []renameCase{
		{
			name:    "moved file becomes package main",
			head:    map[string]string{utilA: utilBody},
			staged:  map[string]string{routingA: "package main\n" + sample},
			renames: moveA,
		},
		{
			name: "same-directory rename changes the package",
			head: map[string]string{
				utilA:                utilBody,
				"internal/util/b.go": "package util\n\nfunc Other() int { return Sample() }\n",
			},
			staged: map[string]string{
				"internal/util/c.go": "package other\n" + sample,
				"internal/util/b.go": "package util\n\nfunc Other() int { return Sample() }\n",
			},
			renames: []pass.Rename{pair(utilA, "internal/util/c.go")},
		},
		{
			name:    "moved file declares a different package",
			head:    map[string]string{utilA: utilBody},
			staged:  withFiles(routingTree, routingA, "package wrong\n"+sample),
			renames: moveA,
		},
		{
			name: "external test becomes an internal test",
			head: map[string]string{"internal/util/a_test.go": extTest},
			staged: withFiles(
				routingTree,
				"internal/routing/a_test.go",
				"package routing\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n",
			),
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
		},
		{
			name:    "same-directory rename flips the test package",
			head:    map[string]string{"internal/util/a_test.go": extTest},
			staged:  map[string]string{"internal/util/b_test.go": intTest},
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/util/b_test.go")},
		},
		{
			name: "new directory mixes packages",
			head: map[string]string{utilA: utilBody},
			staged: map[string]string{
				routingA:                "package routing\n" + sample,
				"internal/routing/x.go": "package other\n",
				"internal/routing/y.go": "package routing\n",
			},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func TestPackageAllowed_Pure(t *testing.T) {
	t.Parallel()

	const sample = "\nfunc Sample() int { return 1 }\n"
	routingTree := map[string]string{
		"internal/routing/r.go": "package routing\n\nfunc R() int { return 2 }\n",
	}
	extTest := "package util_test\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"
	movedExt := "package routing_test\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"
	testCases := []renameCase{
		{
			name:    "whole directory main move",
			head:    map[string]string{"cmd/a/main.go": "package main\n\nfunc main() {}\n"},
			staged:  map[string]string{"cmd/b/main.go": "package main\n\nfunc main() {}\n"},
			renames: []pass.Rename{pair("cmd/a/main.go", "cmd/b/main.go")},
			want:    true,
		},
		{
			name: "multi-file split into a new directory",
			head: map[string]string{
				"internal/util/a.go": "package util\n" + sample,
				"internal/util/b.go": "package util\n\nfunc Other() int { return 2 }\n",
			},
			staged: map[string]string{
				"internal/split/a.go": "package split\n" + sample,
				"internal/split/b.go": "package split\n\nfunc Other() int { return 2 }\n",
			},
			renames: []pass.Rename{
				pair("internal/util/a.go", "internal/split/a.go"),
				pair("internal/util/b.go", "internal/split/b.go"),
			},
			want: true,
		},
		{
			name:    "external test moves to a matching external test package",
			head:    map[string]string{"internal/util/a_test.go": extTest},
			staged:  withFiles(routingTree, "internal/routing/a_test.go", movedExt),
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
			want:    true,
		},
		{
			name: "build-ignored file keeps its package clause",
			head: map[string]string{
				"internal/util/gen.go": "//go:build ignore\n\npackage main\n\nfunc main() {}\n",
			},
			staged: withFiles(
				routingTree, "internal/routing/gen.go",
				"//go:build ignore\n\npackage main\n\nfunc main() {}\n",
			),
			renames: []pass.Rename{pair("internal/util/gen.go", "internal/routing/gen.go")},
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestPackageAllowed_IgnoredFiles(t *testing.T) {
	t.Parallel()

	routingTree := map[string]string{
		"internal/routing/r.go": "package routing\n\nfunc R() int { return 2 }\n",
	}
	testCases := []renameCase{
		{
			name: "build-ignored file changes its package",
			head: map[string]string{
				"internal/util/gen.go": "//go:build ignore\n\npackage main\n\nfunc main() {}\n",
			},
			staged: withFiles(
				routingTree, "internal/routing/gen.go",
				"//go:build ignore\n\npackage routing\n\nfunc main() {}\n",
			),
			renames: []pass.Rename{pair("internal/util/gen.go", "internal/routing/gen.go")},
		},
		{
			name: "legacy build-ignored file changes its package",
			head: map[string]string{
				"internal/util/gen.go": "// +build ignore\n\npackage main\n\nfunc main() {}\n",
			},
			staged: withFiles(
				routingTree, "internal/routing/gen.go",
				"// +build ignore\n\npackage routing\n\nfunc main() {}\n",
			),
			renames: []pass.Rename{pair("internal/util/gen.go", "internal/routing/gen.go")},
		},
	}
	runRenameCases(t, testCases)
}

type pkgMove struct {
	from string
	to   string
}

func mainMoveRow(name string, move pkgMove, newDir string, extra map[string]string) renameCase {
	src := func(pkg, dir string) string {
		return pkgFile(pkg, "import \"example.com/m/"+dir+"\"\n\nfunc main() { println(old.K) }\n")
	}
	return renameCase{
		name: name,
		head: map[string]string{
			"cmd/a/main.go": src(move.from, "old"),
			"old/k.go":      pkgFile("old", "const K = 1\n"),
		},
		staged: withFiles(extra, "cmd/"+newDir+"/main.go", src(move.to, "fresh")),
		renames: []pass.Rename{
			pair("cmd/a/main.go", "cmd/"+newDir+"/main.go"), pair("old/k.go", "fresh/k.go"),
		},
		want: true,
	}
}

func TestPackageAllowed_MainRule(t *testing.T) {
	t.Parallel()

	fresh := map[string]string{"fresh/k.go": pkgFile("old", "const K = 1\n")}
	withMain := withFiles(fresh, "cmd/b/other.go", pkgFile("main", "func other() {}\n"))
	lib := mainMoveRow(
		"lib moves into a directory that already holds main",
		pkgMove{from: "lib", to: "main"},
		"b",
		withMain,
	)
	libToMain := mainMoveRow(
		"lib becomes main in a fresh directory", pkgMove{from: "lib", to: "main"}, "b", fresh,
	)
	libToMain.want = false
	testCases := []renameCase{
		mainMoveRow(
			"main moves to a fresh directory", pkgMove{from: "main", to: "main"}, "b", fresh,
		),
		lib,
		libToMain,
	}
	runRenameCases(t, testCases)
}

func TestPackageAllowed_SingleFile(t *testing.T) {
	t.Parallel()

	testCases := singleFileCases()
	runRenameCases(t, testCases)
}

func singleFileCases() []renameCase {
	const body = "func A() int {\n\treturn 1 + 2 + 3 + 4 + 5 + 6\n}\n"
	ignored := "//go:build ignore\n\npackage main\n\nfunc main() {}\n"
	routing := map[string]string{routingMain: pkgFile("routing", "const Other = 1\n")}
	return []renameCase{
		{
			name:    "sole file renamed in place changes its package",
			head:    map[string]string{utilA: pkgFile("util", body)},
			staged:  map[string]string{"internal/util/b.go": pkgFile("zzz", body)},
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
		},
		{
			name: "a stray test file does not name the new package",
			head: map[string]string{utilA: pkgFile("util", body)},
			staged: withFiles(
				withFiles(
					routing,
					"internal/routing/x_test.go",
					pkgFile("bogus", "func TestX() {}\n"),
				),
				routingA,
				pkgFile("routing", body),
			),
			renames: moveF(),
			want:    true,
		},
		{
			name: "a build-ignored sibling does not name the new package",
			head: map[string]string{utilA: pkgFile("util", body)},
			staged: withFiles(
				withFiles(routing, "internal/routing/gen.go", ignored),
				routingA, pkgFile("routing", body),
			),
			renames: moveF(),
			want:    true,
		},
		{
			name: "a constraint comment after the clause does not ignore the file",
			head: map[string]string{utilA: pkgFile("util", "//go:build ignore\n"+body)},
			staged: withFiles(
				routing, routingA, pkgFile("routing", "//go:build ignore\n"+body),
			),
			renames: moveF(),
			want:    true,
		},
		{
			name: "a plain comment mentioning ignore is no constraint",
			head: map[string]string{utilA: "// please ignore this\n\n" + pkgFile("util", body)},
			staged: withFiles(
				routing, routingA, "// please ignore this\n\n"+pkgFile("routing", body),
			),
			renames: moveF(),
			want:    true,
		},
		{
			name: "a non-Go file never names the new package",
			head: map[string]string{utilA: pkgFile("util", body)},
			staged: withFiles(
				withFiles(routing, "internal/routing/NOTES.md", "package zzz\n"),
				routingA, pkgFile("routing", body),
			),
			renames: moveF(),
			want:    true,
		},
		{
			name: "an ignored tag that merely contains the word is no constraint",
			head: map[string]string{utilA: "//go:build !ignored\n\n" + pkgFile("util", body)},
			staged: withFiles(
				routing, routingA, "//go:build !ignored\n\n"+pkgFile("routing", body),
			),
			renames: moveF(),
			want:    true,
		},
	}
}
