package movepurity

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func rebindTree(extra map[string]string) map[string]string {
	tree := map[string]string{
		"v1/util/a1.go": "package util\n\nfunc A() string { return \"a\" }\n",
		"v1/util/a2.go": "package util\n\nfunc Y() string { return \"v1\" }\n",
		"v2/util/b.go":  "package util\n\nfunc Y() string { return \"v2\" }\n",
	}
	maps.Copy(tree, extra)
	return tree
}

func rebindStaged(extra map[string]string) map[string]string {
	tree := rebindTree(extra)
	tree["v2/util/a1.go"] = tree["v1/util/a1.go"]
	delete(tree, "v1/util/a1.go")
	return tree
}

func rebindDependant(name, call, importLine string, want int) repoCase {
	const useFile = "app/main.go"
	return repoCase{
		name: name,
		head: rebindTree(map[string]string{
			useFile: "package main\n\nimport \"example.com/m/v1/util\"\n\n" +
				"func main() { println(util." + call + "()) }\n",
		}),
		staged: rebindStaged(map[string]string{
			useFile: "package main\n\n" + importLine +
				"\n\nfunc main() { println(util." + call + "()) }\n",
		}),
		renames:      scored("v1/util/a1.go", "v2/util/a1.go", 100),
		changed:      []pass.FileChange{{Path: useFile, Staged: true}},
		wantFindings: want,
	}
}

func rebindCrossModule(name, call string, want int) repoCase {
	const (
		useFile = "app/main.go"
		appMod  = "module example.com/app\n\ngo 1.25\n"
		useSrc  = "package main\n\nimport \"example.com/m/%s/util\"\n\n" +
			"func main() { println(util.%s() + cfg.Port) }\n"
		cfg = "package main\n\nvar cfg struct{ Port int }\n"
	)
	return repoCase{
		name: name,
		head: rebindTree(map[string]string{
			useFile: fmt.Sprintf(useSrc, "v1", call), "app/go.mod": appMod, "app/cfg.go": cfg,
		}),
		staged: rebindStaged(map[string]string{
			useFile: fmt.Sprintf(useSrc, "v2", call), "app/go.mod": appMod, "app/cfg.go": cfg,
		}),
		renames:      scored("v1/util/a1.go", "v2/util/a1.go", 100),
		changed:      []pass.FileChange{{Path: useFile, Staged: true}},
		wantFindings: want,
	}
}

func renamedPackageShadow() repoCase {
	return repoCase{
		name: "alias shadows a renamed package",
		head: map[string]string{
			"v1/util/a1.go": "package util\n\nfunc A() string { return \"a\" }\n",
			"v1/util/a2.go": "package util\n\nfunc Y() string { return \"v1\" }\n",
			"v2/web/b.go":   "package web\n\nfunc Y() string { return \"v2\" }\n",
			"app/main.go": "package main\n\nimport \"example.com/m/v1/util\"\n\n" +
				"func main() { println(util.Y()) }\n",
		},
		staged: map[string]string{
			"v1/util/a2.go": "package util\n\nfunc Y() string { return \"v1\" }\n",
			"v2/web/a1.go":  "package web\n\nfunc A() string { return \"a\" }\n",
			"v2/web/b.go":   "package web\n\nfunc Y() string { return \"v2\" }\n",
			"app/main.go": "package main\n\nimport util \"example.com/m/v2/web\"\n\n" +
				"func main() { println(util.Y()) }\n",
		},
		renames:      scored("v1/util/a1.go", "v2/web/a1.go", 60),
		changed:      []pass.FileChange{{Path: "app/main.go", Staged: true}},
		wantFindings: 1,
	}
}

func TestRebindingQualifiers_Dependants(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		rebindDependant(
			"dependant keeps the same symbol", "A", "import \"example.com/m/v2/util\"", 0,
		),
		rebindDependant(
			"dependant rebinds a shadowing symbol", "Y", "import \"example.com/m/v2/util\"", 1,
		),
		rebindDependant(
			"aliased dependant rebinds a shadowing symbol",
			"Y", "import util \"example.com/m/v2/util\"", 1,
		),
		renamedPackageShadow(),
		rebindCrossModule("cross-module dependant with a sibling var", "A", 0),
		rebindCrossModule("cross-module dependant rebinds a shadow", "Y", 1),
	}

	runRepoCases(t, testCases)
}

func TestRebindingQualifiers_MovedFile(t *testing.T) {
	t.Parallel()

	const (
		oldRun = "package app\n\nimport \"example.com/m/v1/util\"\n\n" +
			"func Run() string { return util.%s() }\n"
		newRun = "package app\n\nimport \"example.com/m/v2/util\"\n\n" +
			"func Run() string { return util.%s() }\n"
	)
	run := func(format, call string) map[string]string {
		return map[string]string{"app/run.go": fmt.Sprintf(format, call)}
	}
	moved := []pass.Rename{
		pair("app/run.go", "srv/run.go"), pair("v1/util/a1.go", "v2/util/a1.go"),
	}
	testCases := []renameCase{
		{
			name:    "moved file keeps the same symbol",
			head:    rebindTree(run(oldRun, "A")),
			staged:  rebindStaged(map[string]string{"srv/run.go": fmt.Sprintf(newRun, "A")}),
			renames: moved,
			want:    true,
		},
		{
			name:    "moved file rebinds a shadowing symbol",
			head:    rebindTree(run(oldRun, "Y")),
			staged:  rebindStaged(map[string]string{"srv/run.go": fmt.Sprintf(newRun, "Y")}),
			renames: moved,
		},
	}
	runRenameCases(t, testCases)
}

const (
	dependant = "d/d.go"
	importOld = "package d\n\nimport \"example.com/m/a/go-util\"\n\n"
	importNew = "package d\n\nimport \"example.com/m/b/go-util\"\n\n"
)

func goUtilDependant(name, body string, moved []pass.Rename, want int) repoCase {
	head := map[string]string{
		"a/go-util/u.go":      pkgFile("util", "func Y() int { return 1 }\n"),
		"a/go-util/x.go":      pkgFile("util", "func X() int { return 1 }\n"),
		"a/go-util/README.md": "readme\n",
		"b/go-util/u.go":      pkgFile("util", "func Y() int { return 2 }\n"),
		"d/cfg.go":            pkgFile("d", "var cfg struct{ Port int }\n"),
		dependant:             importOld + body,
	}
	staged := maps.Clone(head)
	staged[dependant] = importNew + body
	for _, rename := range moved {
		staged[rename.NewPath] = staged[rename.OldPath]
		delete(staged, rename.OldPath)
	}
	return repoCase{
		name: name, head: head, staged: staged, renames: moved,
		changed:      []pass.FileChange{{Path: dependant, Staged: true}},
		wantFindings: want,
	}
}

func TestRebindingQualifiers_PackageNames(t *testing.T) {
	t.Parallel()

	runRepoCases(t, slices.Concat(packageNameDirCases(), packageNameFileCases()))
}

func packageNameDirCases() []repoCase {
	readme := scored("a/go-util/README.md", "b/go-util/README.md", 100)
	xMove := scored("a/go-util/x.go", "b/go-util/x.go", 100)
	useY := "func D() int { return util.Y() }\n"
	useX := "func D() int { return util.X() }\n"
	return []repoCase{
		goUtilDependant(
			"S5 README-only move, package differs from the directory", useY, readme, 1),
		goUtilDependant("S24 control, Y stays in a/go-util", useY, xMove, 1),
		goUtilDependant("moved X follows its directory", useX, xMove, 0),
		goUtilDependant(
			"a local named like the import is no qualifier",
			"func D(util T) int { return util.Y() + util.X() }\n", xMove, 0,
		),
		goUtilDependant(
			"selector on a sibling variable stays quiet",
			"func D() int { return util.X() + cfg.Port }\n", xMove, 0,
		),
		{
			name: "build-ignored and test files do not name the package",
			head: map[string]string{
				"a/go-util/aa_gen.go":  "//go:build ignore\n\npackage main\n",
				"a/go-util/ab_test.go": pkgFile("util_test", "func TestY() {}\n"),
				"a/go-util/u.go":       pkgFile("util", "func Y() int { return 1 }\n"),
				"a/go-util/x.go":       pkgFile("util", "func X() int { return 1 }\n"),
				"b/go-util/aa_gen.go":  "//go:build ignore\n\npackage main\n",
				"b/go-util/ab_test.go": pkgFile("util_test", "func TestY() {}\n"),
				"b/go-util/u.go":       pkgFile("util", "func Y() int { return 2 }\n"),
				dependant:              importOld + useY,
			},
			staged: map[string]string{
				"a/go-util/aa_gen.go":  "//go:build ignore\n\npackage main\n",
				"a/go-util/ab_test.go": pkgFile("util_test", "func TestY() {}\n"),
				"a/go-util/u.go":       pkgFile("util", "func Y() int { return 1 }\n"),
				"b/go-util/x.go":       pkgFile("util", "func X() int { return 1 }\n"),
				"b/go-util/aa_gen.go":  "//go:build ignore\n\npackage main\n",
				"b/go-util/ab_test.go": pkgFile("util_test", "func TestY() {}\n"),
				"b/go-util/u.go":       pkgFile("util", "func Y() int { return 2 }\n"),
				dependant:              importNew + useY,
			},
			renames:      xMove,
			changed:      []pass.FileChange{{Path: dependant, Staged: true}},
			wantFindings: 1,
		},
	}
}

func packageNameFileCases() []repoCase {
	useY := "func D() int { return util.Y() }\n"
	return []repoCase{
		{
			name: "S5b only a test file renamed",
			head: map[string]string{
				"a/go-util/u.go":      pkgFile("util", "func Y() int { return 1 }\n"),
				"a/go-util/u_test.go": pkgFile("util", "func TestU() {}\n"),
				"b/go-util/u.go":      pkgFile("util", "func Y() int { return 2 }\n"),
				dependant:             importOld + useY,
			},
			staged: map[string]string{
				"a/go-util/u.go":      pkgFile("util", "func Y() int { return 1 }\n"),
				"b/go-util/u_test.go": pkgFile("util", "func TestU() {}\n"),
				"b/go-util/u.go":      pkgFile("util", "func Y() int { return 2 }\n"),
				dependant:             importNew + useY,
			},
			renames:      scored("a/go-util/u_test.go", "b/go-util/u_test.go", 100),
			changed:      []pass.FileChange{{Path: dependant, Staged: true}},
			wantFindings: 1,
		},
		{
			name: "package name unresolvable on both sides",
			head: map[string]string{
				"a/go-tst/x_test.go": pkgFile("tst", "func TestX() {}\n"),
				dependant: "package d\n\nimport \"example.com/m/a/go-tst\"\n\n" +
					"func D() int { return tst.Y() }\n",
			},
			staged: map[string]string{
				"b/go-tst/x_test.go": pkgFile("tst", "func TestX() {}\n"),
				dependant: "package d\n\nimport \"example.com/m/b/go-tst\"\n\n" +
					"func D() int { return tst.Y() }\n",
			},
			renames:      scored("a/go-tst/x_test.go", "b/go-tst/x_test.go", 100),
			changed:      []pass.FileChange{{Path: dependant, Staged: true}},
			wantFindings: 1,
		},
	}
}

func splitTree(extra map[string]string) map[string]string {
	tree := map[string]string{
		"a/x/foo.go": pkgFile("x", "func Foo() int { return 1 }\n"),
		"a/x/bar.go": pkgFile("x", "func Bar() int { return 1 }\n"),
	}
	maps.Copy(tree, extra)
	return tree
}

func splitDependant(name, call, target string, want int) repoCase {
	const useFile = "app/main.go"
	source := func(dir string) string {
		return pkgFile(
			"main",
			"import \"example.com/m/"+dir+"\"\n\nfunc main() { println(x."+call+"()) }\n",
		)
	}
	return repoCase{
		name: name,
		head: splitTree(map[string]string{useFile: source("a/x")}),
		staged: map[string]string{
			"b/x/foo.go": pkgFile("x", "func Foo() int { return 1 }\n"),
			"c/x/bar.go": pkgFile("x", "func Bar() int { return 1 }\n"),
			useFile:      source(target),
		},
		renames: []pass.Rename{
			{OldPath: "a/x/foo.go", NewPath: "b/x/foo.go", Score: 100},
			{OldPath: "a/x/bar.go", NewPath: "c/x/bar.go", Score: 100},
		},
		changed:      []pass.FileChange{{Path: useFile, Staged: true}},
		wantFindings: want,
	}
}

func TestRebindingQualifiers_SplitTargets(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		splitDependant("dependant of a split follows the half it uses", "Bar", "c/x", 0),
		splitDependant("dependant rewritten to the wrong half of a split", "Foo", "c/x", 1),
		{
			name: "two imports sharing a local name never rebind",
			head: splitTree(map[string]string{
				"b/x/other.go": pkgFile("x", "func Other() int { return 1 }\n"),
				dependant: pkgFile(
					"d",
					"import (\n\t\"example.com/m/a/x\"\n\t\"example.com/m/b/x\"\n)\n\n"+
						"func D() int { return x.Foo() }\n",
				),
			}),
			staged: map[string]string{
				"c/x/foo.go":   pkgFile("x", "func Foo() int { return 1 }\n"),
				"a/x/bar.go":   pkgFile("x", "func Bar() int { return 1 }\n"),
				"b/x/other.go": pkgFile("x", "func Other() int { return 1 }\n"),
				dependant: pkgFile(
					"d",
					"import (\n\t\"example.com/m/c/x\"\n\t\"example.com/m/b/x\"\n)\n\n"+
						"func D() int { return x.Foo() }\n",
				),
			},
			renames:      scored("a/x/foo.go", "c/x/foo.go", 100),
			changed:      []pass.FileChange{{Path: dependant, Staged: true}},
			wantFindings: 1,
		},
	}
	runRepoCases(t, testCases)
}

func movedAwayDependant(name, fooTarget string, want int) repoCase {
	const body = "\n\nfunc D() int { return util.Foo() + moved.K }\n"
	use := func(dir string) string {
		return pkgFile(
			"d",
			"import (\n\t\"example.com/m/util\"\n\tmoved \"example.com/m/"+dir+"\"\n)"+body,
		)
	}
	return repoCase{
		name: name,
		head: map[string]string{
			"util/foo.go":  pkgFile("util", "func Foo() int { return 1 }\n"),
			"util/keep.go": pkgFile("util", "func Keep() int { return 1 }\n"),
			"old/k.go":     pkgFile("old", "const K = 1\n"),
			dependant:      use("old"),
		},
		staged: map[string]string{
			fooTarget:      pkgFile("util", "func Foo() int { return 1 }\n"),
			"util/keep.go": pkgFile("util", "func Keep() int { return 1 }\n"),
			"fresh/k.go":   pkgFile("old", "const K = 1\n"),
			dependant:      use("fresh"),
		},
		renames: []pass.Rename{
			{OldPath: "util/foo.go", NewPath: fooTarget, Score: 100},
			{OldPath: "old/k.go", NewPath: "fresh/k.go", Score: 100},
		},
		changed:      []pass.FileChange{{Path: dependant, Staged: true}},
		wantFindings: want,
	}
}

func TestRebindingQualifiers_MovedAway(t *testing.T) {
	t.Parallel()

	const useUtilFoo = "import \"example.com/m/util\"\n\nfunc A() int { return util.Foo() }\n"
	testCases := []repoCase{
		movedAwayDependant(
			"S20 dependant keeps the old import for a moved declaration", "util2/foo.go", 1),
		movedAwayDependant(
			"renamed within its directory the declaration stays reachable", "util/foo2.go", 0),
		{
			name: "moved file keeps the old import for a moved declaration",
			head: map[string]string{
				"p/a.go":      pkgFile("p", useUtilFoo),
				"util/foo.go": pkgFile("util", "func Foo() int { return 1 }\n"),
				"util/k.go":   pkgFile("util", "func K() int { return 1 }\n"),
			},
			staged: map[string]string{
				"q/a.go":       pkgFile("q", useUtilFoo),
				"util2/foo.go": pkgFile("util", "func Foo() int { return 1 }\n"),
				"util/k.go":    pkgFile("util", "func K() int { return 1 }\n"),
			},
			renames: []pass.Rename{
				pair("p/a.go", "q/a.go"),
				pair("util/foo.go", "util2/foo.go"),
			},
			wantFindings: 1,
		},
	}
	runRepoCases(t, testCases)
}

func tstRow(name, keepDir string) repoCase {
	const (
		oldTst = "package d\n\nimport \"example.com/m/a/go-tst\"\n\n"
		newTst = "package d\n\nimport \"example.com/m/b/go-tst\"\n\n"
		usage  = "func D() int { return foo.Bar() }\n"
	)
	testSrc := pkgFile("tst", "func TestX() {}\n")
	prodSrc := pkgFile("tst", "func Y() int { return 1 }\n")
	prod := keepDir + "/go-tst/u.go"
	return repoCase{
		name: name,
		head: map[string]string{
			"a/go-tst/x_test.go": testSrc, prod: prodSrc, dependant: oldTst + usage,
		},
		staged: map[string]string{
			"b/go-tst/x_test.go": testSrc, prod: prodSrc, dependant: newTst + usage,
		},
		renames:      scored("a/go-tst/x_test.go", "b/go-tst/x_test.go", 100),
		changed:      []pass.FileChange{{Path: dependant, Staged: true}},
		wantFindings: 1,
	}
}

func TestRebindingQualifiers_UnknownNames(t *testing.T) {
	t.Parallel()

	runRepoCases(t, []repoCase{
		tstRow("only the old import has an unresolvable package name", "b"),
		tstRow("only the new import has an unresolvable package name", "a"),
	})
}

func TestRebindingQualifiers_ExplicitNames(t *testing.T) {
	t.Parallel()

	const useX = "func D() int { return x.Foo() }\n"
	head := map[string]string{
		"a/foo.go": pkgFile("a", "func Foo() int { return 1 }\n"),
		"a/m.go":   pkgFile("a", "func Bar() int { return 3 }\n"),
		"b/foo.go": pkgFile("b", "func Foo() int { return 2 }\n"),
		dependant:  "package d\n\nimport x \"example.com/m/a\"\n\n" + useX,
	}
	staged := maps.Clone(head)
	staged["b/m.go"] = staged["a/m.go"]
	delete(staged, "a/m.go")
	staged[dependant] = "package d\n\nimport x \"example.com/m/b\"\n\n" + useX
	runRepoCases(t, []repoCase{{
		name: "an explicit alias rewritten to a package that never held the member",
		head: head, staged: staged, renames: scored("a/m.go", "b/m.go", 100),
		changed: []pass.FileChange{{Path: dependant, Staged: true}}, wantFindings: 1,
	}})
}

func TestRebindingQualifiers_CrossModuleMove(t *testing.T) {
	t.Parallel()

	const body = "import \"example.com/m/go-util\"\n\nfunc R() int { return util.Y() }\n"
	utilSrc := pkgFile("util", "func Y() int { return 1 }\n")
	runRenameCases(t, []renameCase{{
		name: "a moved file's import cannot be named in the new module",
		head: map[string]string{"go-util/u.go": utilSrc, "a/run.go": pkgFile("a", body)},
		staged: map[string]string{
			"go-util/u.go": utilSrc,
			"app/run.go":   pkgFile("app", body),
			"app/go.mod":   "module example.com/app\n\ngo 1.25\n",
		},
		renames: []pass.Rename{pair("a/run.go", "app/run.go")},
	}})
}
