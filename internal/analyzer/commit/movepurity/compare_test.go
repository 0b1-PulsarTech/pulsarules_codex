package movepurity

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func importRow(name, oldBody, newBody string) renameCase {
	return renameCase{
		name:    name,
		head:    map[string]string{utilA: pkgFile("util", oldBody)},
		staged:  map[string]string{routingA: pkgFile("routing", newBody)},
		renames: moveF(),
	}
}

func TestImports_RenameRows_Swaps(t *testing.T) {
	t.Parallel()

	moveA := []pass.Rename{pair(utilA, routingA)}
	body := "\nfunc Token() int { return rand.Int() }\n"
	testCases := []renameCase{
		{
			name: "import path swapped for a same-named package",
			head: map[string]string{utilA: "package util\n\nimport \"math/rand\"\n" + body},
			staged: map[string]string{
				routingA: "package routing\n\nimport \"crypto/rand\"\n" + body,
			},
			renames: moveA,
		},
		{
			name: "import swapped for a same-named fake in a group",
			head: map[string]string{
				utilA: "package util\n\nimport (\n\t\"fmt\"\n\n" +
					"\t\"example.com/m/internal/store\"\n)\n\n" +
					"func S() { store.Put(fmt.Sprint(1)) }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nimport (\n\t\"fmt\"\n\n" +
					"\tstore \"example.com/m/internal/fakestore\"\n)\n\n" +
					"func S() { store.Put(fmt.Sprint(1)) }\n",
			},
			renames: moveA,
		},
		importRow("unrelated fmt import added", "func F() {}\n",
			"import \"fmt\"\n\nfunc F() {}\n"),
		importRow(
			"cgo import added",
			"func F() int { return 1 }\n",
			"import \"C\"\n\nfunc F() int { return 1 }\n",
		),
		accepted(
			importRow("blank import identical", "import _ \"embed\"\n", "import _ \"embed\"\n"),
		),
		importRow(
			"dot import swapped",
			"import . \"example.com/m/internal/util\"\n",
			"import . \"example.com/m/internal/routing\"\n",
		),
	}
	runRenameCases(t, testCases)
}

func rewriteRow(name, alias, call string, dirMove ...pass.Rename) renameCase {
	const kBody = "package x\n\nfunc K() int { return 1 }\n"
	fn := "\n\nfunc F() int { return " + call + " }\n"
	row := importRow(name,
		"import "+alias+"\"example.com/m/internal/x\""+fn,
		"import "+alias+"\"example.com/m/internal/y\""+fn)
	if len(dirMove) > 0 {
		row.head["internal/x/k.go"], row.staged["internal/y/k.go"] = kBody, kBody
		row.renames = moveF(dirMove...)
		row.want = true
	}
	return row
}

func TestImports_RenameRows_Rewrites(t *testing.T) {
	t.Parallel()

	moveA := []pass.Rename{pair(utilA, routingA)}
	xToY := pair("internal/x/k.go", "internal/y/k.go")
	testCases := []renameCase{
		{
			name: "foreign package with the old basename",
			head: map[string]string{utilA: "package util\n\nfunc F() int { return Parse() }\n"},
			staged: map[string]string{
				routingA: "package routing\n\nimport \"github.com/evil/util\"\n\n" +
					"func F() int { return util.Parse() }\n",
			},
			renames: moveA,
		},
		{
			name: "foreign package with the new basename",
			head: map[string]string{
				utilA: "package util\n\nimport \"github.com/evil/routing\"\n\n" +
					"func F() int { return routing.Parse() }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nfunc F() int { return Parse() }\n",
			},
			renames: moveA,
		},
		rewriteRow("dir rename import rewrite", "", "x.K()", xToY),
		rewriteRow("import rewrite without a directory rename", "", "x.K()"),
		rewriteRow("aliased import rewrite keeps the alias", "a ", "a.K()", xToY),
		{
			name: "no go.mod means no import rule applies",
			head: map[string]string{"go.mod": "", utilA: "package util\n\nimport \"a/x\"\n"},
			staged: map[string]string{
				"go.mod": "",
				routingA: "package routing\n\nimport \"a/y\"\n",
			},
			renames: []pass.Rename{pair(utilA, routingA), pair("x/k.go", "y/k.go")},
		},
	}
	runRenameCases(t, testCases)
}

const (
	utilStay = "internal/util/stay.go"
	linkBody = "import _ \"example.com/m/internal/routing\"\n"
	initBody = "func init() { register() }\n"
	initFile = "a/go-util/init.go"
	useUtilX = "func D() int { return util.X() }\n"
)

func linked(name, body string, stay map[string]string) renameCase {
	head := map[string]string{utilA: pkgFile("util", body), utilStay: "package util\n"}
	staged := map[string]string{routingA: pkgFile("routing", body), utilStay: "package util\n"}
	for file, src := range stay {
		head[file], staged[file] = src, src
	}
	return renameCase{name: name, head: head, staged: staged, renames: moveF()}
}

func TestLoadFollowsDir(t *testing.T) {
	t.Parallel()

	const (
		varBody     = "var registered = reg()\n\nfunc reg() int { return 1 }\n"
		testMain    = "func TestMain(m *testing.M) { m.Run() }\n"
		oldTest     = "internal/util/a_test.go"
		linkSibling = "internal/util/b.go"
		newTest     = "internal/routing/a_test.go"
	)
	link := map[string]string{linkSibling: pkgFile("util", linkBody)}
	testCases := []renameCase{
		linked("init with a staying file", initBody, nil),
		linked("init with a staying importer", initBody, link),
		linked("a named call with a staying importer", varBody, link),
		linked("a blank call with a staying importer", "var _ = reg()\n"+varBody, link),
		linked("TestMain with a staying importer", testMain, link),
		accepted(linked(
			"an errors.New sentinel needs no link",
			"import \"errors\"\n\nvar ErrX = errors.New(\"x\")\n",
			nil)),
		accepted(linked("no load work needs no link", "func F() int { return 1 }\n", nil)),
		{
			name: "two directories merge into one new directory",
			head: map[string]string{
				utilA:                 pkgFile("util", initBody),
				"internal/other/c.go": "package util\n",
			},
			staged: map[string]string{
				routingA:                pkgFile("routing", initBody),
				"internal/routing/c.go": "package routing\n",
			},
			renames: moveF(pair("internal/other/c.go", "internal/routing/c.go")),
		},
		{
			name: "a moved test file with an init and a staying importer",
			head: map[string]string{
				oldTest:     pkgFile("util", initBody),
				linkSibling: pkgFile("util", linkBody),
			},
			staged: map[string]string{
				newTest:     pkgFile("routing", initBody),
				linkSibling: pkgFile("util", linkBody),
			},
			renames: []pass.Rename{pair(oldTest, newTest)},
		},
	}
	runRenameCases(t, testCases)
}

func TestLoadFollowsDir_WholeDir(t *testing.T) {
	t.Parallel()

	const (
		testMain  = "func TestMain(m *testing.M) { m.Run() }\n"
		oldTest   = "internal/util/a_test.go"
		newTest   = "internal/routing/a_test.go"
		otherMove = "internal/other/c.go"
	)
	moveRow := func(name, body string, shared map[string]string, extra ...pass.Rename) renameCase {
		head := map[string]string{oldTest: pkgFile("util", body)}
		staged := map[string]string{newTest: pkgFile("routing", body)}
		for file, src := range shared {
			head[file], staged[file] = src, src
		}
		return renameCase{
			name: name, head: head, staged: staged,
			renames: append([]pass.Rename{pair(oldTest, newTest)}, extra...),
		}
	}
	split := moveRow("the directory splits across two destinations", initBody,
		map[string]string{}, pair("internal/util/c.go", otherMove))
	split.head["internal/util/c.go"], split.staged[otherMove] = "package util\n", "package other\n"
	prod := renameCase{
		name:    "a whole production directory moves with an init",
		head:    map[string]string{utilA: pkgFile("util", initBody)},
		staged:  map[string]string{routingA: pkgFile("routing", initBody)},
		renames: moveF(),
	}
	testCases := []renameCase{
		prod,
		accepted(moveRow("the whole directory moves with a test init", initBody, nil)),
		accepted(moveRow("the whole directory moves with a TestMain", testMain, nil)),
		moveRow("a Go file stays in the old directory", initBody,
			map[string]string{"internal/util/k.go": "package util\n"}),
		moveRow("a test file stays in the old directory", testMain,
			map[string]string{"internal/util/k_test.go": "package util\n"}),
		moveRow("the new directory already holds Go files", initBody,
			map[string]string{"internal/routing/k.go": "package routing\n"}),
		split,
		accepted(moveRow("a non-Go file may stay behind", initBody,
			map[string]string{"internal/util/README.md": "x\n"})),
	}
	runRenameCases(t, testCases)
}

func TestStrandsCompiled(t *testing.T) {
	t.Parallel()

	const (
		asmOld, asmNew = "internal/util/add_amd64.s", "internal/routing/add_amd64.s"
		cNew           = "internal/routing/x.c"
		body           = "func F() int { return 1 }\n"
	)
	moveRow := func(
		name string, head, staged map[string]string, renames []pass.Rename,
	) renameCase {
		head[utilA], staged[routingA] = pkgFile("util", body), pkgFile("routing", body)
		return renameCase{name: name, head: head, staged: staged, renames: renames}
	}
	testCases := []renameCase{
		moveRow("the old directory keeps an assembly file",
			map[string]string{asmOld: "TEXT\n"}, map[string]string{asmOld: "TEXT\n"}, moveF()),
		moveRow("the new directory holds a stray C file",
			map[string]string{cNew: "int x;\n"}, map[string]string{cNew: "int x;\n"}, moveF()),
		accepted(moveRow("the assembly file moves along",
			map[string]string{asmOld: "TEXT\n"}, map[string]string{asmNew: "TEXT\n"},
			moveF(pair(asmOld, asmNew)))),
		accepted(moveRow("a staying non-compiled file is no hazard",
			map[string]string{"internal/util/doc.txt": "x\n"},
			map[string]string{"internal/util/doc.txt": "x\n"}, moveF())),
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(newFakeRepo(testCase.head, testCase.staged), testCase.renames)
			want := verdictStrands
			if testCase.want {
				want = verdictPure
			}
			if got := moves.renameVerdict(testCase.renames[0]); got != want {
				t.Fatalf("renameVerdict() = %v, want %v", got, want)
			}
		})
	}
}

type noHeadDirRepo struct{ fakeRepo }

func (noHeadDirRepo) HeadDir(string) ([]string, error) { return nil, errMissing }

type noDirRepo struct{ fakeRepo }

func (noDirRepo) StagedDir(string) ([]string, error) { return nil, errMissing }

func TestDirListingFailure(t *testing.T) {
	t.Parallel()

	const (
		oldTest = "internal/util/a_test.go"
		newTest = "internal/routing/a_test.go"
		plain   = "func F() int { return 1 }\n"
	)
	testInit := newFakeRepo(
		map[string]string{oldTest: pkgFile("util", initBody)},
		map[string]string{newTest: pkgFile("routing", initBody)},
	)
	plainMove := newFakeRepo(
		map[string]string{utilA: pkgFile("util", plain)},
		map[string]string{routingA: pkgFile("routing", plain)},
	)
	testCases := []struct {
		name    string
		repo    gitReader
		renames []pass.Rename
	}{
		{
			"an unlistable new directory",
			noHeadDirRepo{testInit},
			[]pass.Rename{pair(oldTest, newTest)},
		},
		{
			"an unlistable old directory",
			noDirRepo{testInit},
			[]pass.Rename{pair(oldTest, newTest)},
		},
		{"an unlistable directory beside compiled files", noDirRepo{plainMove}, moveF()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(testCase.repo, testCase.renames)
			if moves.renameVerdict(testCase.renames[0]) == verdictPure {
				t.Fatal("an unlistable directory must not read as free of load work")
			}
		})
	}
}

func unlinkRow(name string, want int) repoCase {
	moved := scored("a/go-util/x.go", "b/go-util/x.go", 100)
	return goUtilDependant(name, useUtilX, moved, want)
}

func withStaying(row repoCase, file, body string) repoCase {
	row.head = maps.Clone(row.head)
	row.staged = maps.Clone(row.staged)
	row.head[file], row.staged[file] = body, body
	return row
}

func unlinking(row repoCase) repoCase {
	row.wantContains = "drops the last import of example.com/m/a/go-util, " +
		"whose init/load-time work"
	return row
}

func secondAlias() repoCase {
	const blank = "import _ \"example.com/m/a/go-util\"\n\n"
	row := unlinkRow("a dependant keeping the import under a second alias stays linked", 0)
	row.head[dependant] = importOld[:len(importOld)-1] + blank + useUtilX
	row.staged[dependant] = importNew[:len(importNew)-1] + blank + useUtilX
	return row
}

func TestUnlinksLoad(t *testing.T) {
	t.Parallel()

	utilInit := pkgFile("util", initBody)
	testCases := []repoCase{
		unlinkRow("a dependant drops the import with no load work staying", 0),
		unlinking(withStaying(
			unlinkRow("a dependant drops the import of a dir keeping an init", 1),
			initFile, utilInit)),
		unlinking(withStaying(
			unlinkRow("a dependant drops the import of a dir keeping a blank import", 1),
			initFile, pkgFile("util", linkBody))),
		withStaying(
			unlinkRow("a staying test file with an init does not link", 0),
			"a/go-util/init_test.go", utilInit),
		withStaying(
			unlinkRow("a staying unparsable file is treated as load work", 1),
			initFile, "package {\n"),
		withStaying(
			unlinkRow("a staying underscore file with an init does not link", 0),
			"a/go-util/_x.go", utilInit),
		withStaying(
			unlinkRow("a staying ignored file with an init does not link", 0),
			initFile, "//go:build ignore\n\n"+utilInit),
		withStaying(secondAlias(), initFile, utilInit),
	}
	runRepoCases(t, testCases)
}

func movedDependant(row repoCase) repoCase {
	const movedTo = "d2/d.go"
	row.staged = maps.Clone(row.staged)
	row.staged[movedTo] = row.staged[dependant]
	delete(row.staged, dependant)
	row.renames = append(slices.Clone(row.renames), scored(dependant, movedTo, 100)...)
	row.changed = nil
	return row
}

func TestUnlinksLoad_MovedDependant(t *testing.T) {
	t.Parallel()

	withInit := withStaying(
		unlinkRow("a moved dependant drops the last import of a dir keeping an init", 1),
		initFile, pkgFile("util", initBody))
	control := unlinkRow("a moved dependant with no load work staying", 0)
	runRepoCases(t, []repoCase{unlinking(movedDependant(withInit)), movedDependant(control)})
}

func TestUnlinksLoad_Suggestion(t *testing.T) {
	t.Parallel()

	row := withStaying(unlinkRow("x", 1), initFile, pkgFile("util", initBody))
	analyzer := NewAnalyzer(newFakeRepo(row.head, row.staged))
	findings := analyzer.Analyze(&pass.Context{
		StagedRenames: row.renames, ChangedFiles: row.changed,
	})
	want := "keep a blank import of example.com/m/a/go-util " +
		"in that file, or move the load-time work first"
	if len(findings) != 1 || findings[0].Suggestion != want {
		t.Fatalf("findings = %+v, want one with suggestion %q", findings, want)
	}
}

func TestUnlinksLoad_StagedDirFailure(t *testing.T) {
	t.Parallel()

	row := unlinkRow("x", 0)
	analyzer := NewAnalyzer(noDirRepo{newFakeRepo(row.head, row.staged)})
	findings := analyzer.Analyze(&pass.Context{
		StagedRenames: row.renames, ChangedFiles: row.changed,
	})
	if len(findings) == 0 {
		t.Fatal("an unlistable old directory must not read as free of load work")
	}
}

func TestClauseOnlyVerdict_AnyScore(t *testing.T) {
	t.Parallel()

	row := linked("x", initBody, nil)
	const lead = "differs only by mechanical package/import fixups but is not behaviour-neutral"
	testCases := []struct {
		name  string
		score int
	}{{"a low-similarity clause-only move", 50}, {"a high-similarity clause-only move", 100}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analyzer := NewAnalyzer(newFakeRepo(row.head, row.staged))
			findings := analyzer.Analyze(&pass.Context{
				StagedRenames: scored(utilA, routingA, testCase.score),
			})
			if len(findings) != 1 || !strings.Contains(findings[0].Message, lead) ||
				!strings.Contains(findings[0].Suggestion, "package clause and import fixups") ||
				!strings.Contains(findings[0].Suggestion, "whole directory") {
				t.Fatalf("findings = %+v, want one with %q and the relocation advice",
					findings, lead)
			}
		})
	}
}
