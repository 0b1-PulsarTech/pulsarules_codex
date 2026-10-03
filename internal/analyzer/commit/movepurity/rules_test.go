package movepurity

import (
	"maps"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func routingImporting(importPath, body string) string {
	return "package routing\n\nimport \"" + importPath + "\"\n\n" + body
}

func TestQualifiers_RenameRows_Additions(t *testing.T) {
	t.Parallel()

	moveA := []pass.Rename{pair(utilA, routingA)}
	imp := "import \"" + utilPkg + "\"\n\n"
	keep := map[string]string{
		"internal/util/k.go": "package util\n\nfunc Keep() int { return 1 }\n",
	}
	testCases := []renameCase{
		{
			name: "forced addition, X stays in an unchanged sibling",
			head: withFiles(keep, utilA, "package util\n\nfunc F() int { return Keep() }\n"),
			staged: withFiles(keep, routingA,
				"package routing\n\n"+imp+"func F() int { return util.Keep() }\n"),
			renames: moveA,
			want:    true,
		},
		{
			name: "old package imported under an alias",
			head: withFiles(keep, utilA, "package util\n\nfunc F() int { return Keep() }\n"),
			staged: withFiles(
				keep, routingA,
				"package routing\n\nimport u \""+utilPkg+"\"\n\n"+
					"func F() int { return u.Keep() }\n",
			),
			renames: moveA,
			want:    true,
		},
		{
			name: "hyphenated directory name",
			head: map[string]string{
				"internal/foo-bar/a.go": "package foobar\n\nfunc F() int { return Keep() }\n",
				"internal/foo-bar/k.go": "package foobar\n\nfunc Keep() int { return 1 }\n",
			},
			staged: map[string]string{
				routingA: routingImporting("example.com/m/internal/foo-bar",
					"func F() int { return foobar.Keep() }\n"),
				"internal/foo-bar/k.go": "package foobar\n\nfunc Keep() int { return 1 }\n",
			},
			renames: []pass.Rename{pair("internal/foo-bar/a.go", routingA)},
			want:    true,
		},
		{
			name: "version suffix directory",
			head: map[string]string{
				"internal/x/v2/a.go": "package x\n\nfunc F() int { return Keep() }\n",
				"internal/x/v2/k.go": "package x\n\nfunc Keep() int { return 1 }\n",
			},
			staged: map[string]string{
				routingA: routingImporting("example.com/m/internal/x/v2",
					"func F() int { return x.Keep() }\n"),
				"internal/x/v2/k.go": "package x\n\nfunc Keep() int { return 1 }\n",
			},
			renames: []pass.Rename{pair("internal/x/v2/a.go", routingA)},
			want:    true,
		},
		{
			name: "name declared in the moved file itself",
			head: map[string]string{
				utilA: "package util\n\nfunc Helper() int { return 1 }\n\n" +
					"func F() int { return Helper() }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\n" + imp + "func Helper() int { return 1 }\n\n" +
					"func F() int { return util.Helper() }\n",
			},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func TestQualifiers_RenameRows_Rejected(t *testing.T) {
	t.Parallel()

	runRenameCases(t, slices.Concat(rejectedSelfQualCases(), rejectedMovedCases()))
}

func rejectedSelfQualCases() []renameCase {
	moveA := []pass.Rename{pair(utilA, routingA)}
	imp := "import \"" + utilPkg + "\"\n\n"
	return []renameCase{
		{
			name: "name appears in a new sibling only",
			head: map[string]string{
				utilA: "package util\n\nfunc Helper() int { return 1 }\n\n" +
					"func F() int { return Helper() }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\n" + imp + "func Helper() int { return 1 }\n\n" +
					"func F() int { return util.Helper() }\n",
				"internal/util/h.go": "package util\n\nfunc Helper() int { return 99 }\n",
			},
			renames: moveA,
		},
		{
			name: "name lives in a file moved in the same commit",
			head: map[string]string{
				utilA:                "package util\n\nfunc A() int { return B() }\n",
				"internal/util/b.go": "package util\n\nfunc B() int { return 1 }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\n" + imp +
					"func A() int { return util.B() }\n",
				"internal/routing/b.go": "package routing\n\nfunc B() int { return 1 }\n",
			},
			renames: []pass.Rename{
				pair(utilA, routingA),
				pair("internal/util/b.go", "internal/routing/b.go"),
			},
		},
		{
			name: "addition of an unexported name",
			head: withFiles(
				map[string]string{
					"internal/util/k.go": "package util\n\nfunc keep() int { return 1 }\n",
				},
				utilA,
				"package util\n\nfunc F() int { return keep() }\n",
			),
			staged: withFiles(
				map[string]string{
					"internal/util/k.go": "package util\n\nfunc keep() int { return 1 }\n",
				},
				routingA,
				"package routing\n\n"+imp+"func F() int { return util.keep() }\n",
			),
			renames: moveA,
		},
	}
}

func rejectedMovedCases() []renameCase {
	moveA := []pass.Rename{pair(utilA, routingA)}
	imp := "import \"" + utilPkg + "\"\n\n"
	keep := map[string]string{
		"internal/util/k.go": "package util\n\nfunc Keep() int { return 1 }\n",
	}
	return []renameCase{
		{
			name: "addition where X was a parameter, not a package-level name",
			head: withFiles(keep, utilA, "package util\n\nfunc F(Keep int) int { return Keep }\n"),
			staged: withFiles(keep, routingA,
				"package routing\n\n"+imp+"func F(Keep int) int { return util.Keep }\n"),
			renames: moveA,
		},
		{
			name: "addition of an unrelated qualifier",
			head: withFiles(keep, utilA, "package util\n\nfunc F() int { return Keep() }\n"),
			staged: withFiles(keep, routingA,
				"package routing\n\nimport \"os\"\n\nfunc F() int { return os.Keep() }\n"),
			renames: moveA,
		},
	}
}

func TestQualifiers_RemovalRows(t *testing.T) {
	t.Parallel()

	moveA := []pass.Rename{pair(utilA, routingA)}
	routingR := map[string]string{
		"internal/routing/r.go": "package routing\n\nfunc R() int { return 1 }\n",
	}
	const routedF = "package routing\n\nfunc F() int { return R() }\n"
	oldBody := "package util\n\nimport \"example.com/m/internal/routing\"\n\n" +
		"func F() int { return routing.R() }\n"
	testCases := []renameCase{
		{
			name:    "forced removal, X lives in a non-moving newDir file",
			head:    withFiles(routingR, utilA, oldBody),
			staged:  withFiles(routingR, routingA, routedF),
			renames: moveA,
			want:    true,
		},
		{
			name: "new package imported under an alias",
			head: withFiles(routingR, utilA,
				"package util\n\nimport r \"example.com/m/internal/routing\"\n\n"+
					"func F() int { return r.R() }\n"),
			staged:  withFiles(routingR, routingA, routedF),
			renames: moveA,
			want:    true,
		},
		{
			name: "name declared in a file new at the staged tree",
			head: map[string]string{utilA: oldBody},
			staged: map[string]string{
				routingA: "package routing\n\nfunc F() int { return R() }\n\n" +
					"func R() int { return 0 }\n",
			},
			renames: moveA,
		},
		{
			name: "removal where the sibling is itself moving in",
			head: withFiles(
				map[string]string{
					"internal/util/r.go": "package util\n\nfunc R() int { return 1 }\n",
				},
				utilA,
				oldBody,
			),
			staged: map[string]string{
				routingA:                routedF,
				"internal/routing/r.go": "package routing\n\nfunc R() int { return 1 }\n",
			},
			renames: []pass.Rename{
				pair(utilA, routingA),
				pair("internal/util/r.go", "internal/routing/r.go"),
			},
		},
		{
			name: "removal where the sibling vanished from the staged tree",
			head: withFiles(routingR, utilA, oldBody),
			staged: map[string]string{
				routingA: routedF,
			},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func withFiles(base map[string]string, name, content string) map[string]string {
	files := map[string]string{name: content}
	maps.Copy(files, base)
	return files
}

const (
	oldImp = "import \"example.com/m/fx\"\n\n"
	newImp = "import \"example.com/m/gen/genfixtures\"\n\n"
)

func dependantFlagged(name, dir, oldBody, newBody string) repoCase {
	row := dependantClean(name, dir, oldBody, newBody)
	row.wantFindings = 1
	return row
}

func dependantClean(name, dir, oldBody, newBody string) repoCase {
	const (
		tmplHead = "package fixtures\n\ntype TemplateFixture struct{}\n\n" +
			"func NewTemplate() int { return 1 }\n\nfunc private() {}\n"
		tmplNew = "package genfixtures\n\ntype TemplateFixture struct{}\n\n" +
			"func NewTemplate() int { return 1 }\n\nfunc private() {}\n"
	)
	file := dir + "/f.go"
	return repoCase{
		name:    name,
		head:    map[string]string{"fx/template.go": tmplHead, file: oldBody},
		staged:  map[string]string{"gen/genfixtures/template.go": tmplNew, file: newBody},
		renames: scored("fx/template.go", "gen/genfixtures/template.go", 90),
		changed: []pass.FileChange{{Path: file, Staged: true}},
	}
}

func inFile(row repoCase, file string) repoCase {
	moved := func(files map[string]string, body string) map[string]string {
		out := map[string]string{}
		for path, src := range files {
			if path != "use/f.go" {
				out[path] = src
			}
		}
		out["use/"+file] = body
		return out
	}
	row.head = moved(row.head, row.head["use/f.go"])
	row.staged = moved(row.staged, row.staged["use/f.go"])
	row.changed = []pass.FileChange{{Path: "use/" + file, Staged: true}}
	return row
}

func TestDependantRules_Accepted(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		dependantClean(
			"qualifier swap in a dependant elsewhere", "use",
			"package use\n\n"+oldImp+"var V = fixtures.NewTemplate()\n",
			"package use\n\n"+newImp+"var V = genfixtures.NewTemplate()\n"),
		dependantClean(
			"bare name in the old directory gains the qualifier", "fx",
			"package fixtures\n\ntype A struct{ T *TemplateFixture }\n",
			"package fixtures\n\n"+newImp+"type A struct{ T *genfixtures.TemplateFixture }\n"),
		dependantClean(
			"qualified name in the new directory turns bare", "gen/genfixtures",
			"package genfixtures\n\n"+oldImp+"var V = fixtures.NewTemplate()\n",
			"package genfixtures\n\nvar V = NewTemplate()\n"),
		dependantClean(
			"aliased imports on both sides", "use",
			"package use\n\nimport f \"example.com/m/fx\"\n\nvar V = f.NewTemplate()\n",
			"package use\n\nimport g \"example.com/m/gen/genfixtures\"\n\n"+
				"var V = g.NewTemplate()\n"),
	}
	runRepoCases(t, testCases)
}

func TestDependantRules_Rejected(t *testing.T) {
	t.Parallel()

	testCases := []repoCase{
		dependantFlagged(
			"name the moved files never declared", "use",
			"package use\n\n"+oldImp+"var V = fixtures.Other()\n",
			"package use\n\n"+newImp+"var V = genfixtures.Other()\n"),
		dependantFlagged(
			"unexported name", "use",
			"package use\n\n"+oldImp+"var V = fixtures.private\n",
			"package use\n\n"+newImp+"var V = genfixtures.private\n"),
		dependantFlagged(
			"qualifier swap beside a literal edit", "use",
			"package use\n\n"+oldImp+"var V = fixtures.NewTemplate() + 1\n",
			"package use\n\n"+newImp+"var V = genfixtures.NewTemplate() + 2\n"),
		dependantFlagged(
			"unrelated import added alongside", "use",
			"package use\n\n"+oldImp+"var V = fixtures.NewTemplate()\n",
			"package use\n\nimport \"os\"\n\n"+newImp+"var V = genfixtures.NewTemplate()\n"),
		dependantFlagged(
			"qualifier is a shadowing local", "use",
			"package use\n\n"+oldImp+"func F(fixtures T) int { return fixtures.NewTemplate() }\n",
			"package use\n\n"+newImp+"func F(fixtures T) int { return "+
				"genfixtures.NewTemplate() }\n"),
		dependantFlagged(
			"bare name in an unrelated package", "use",
			"package use\n\nvar V = NewTemplate()\n",
			"package use\n\n"+newImp+"var V = genfixtures.NewTemplate()\n"),
	}
	runRepoCases(t, testCases)
}

func TestDependantRules_StayingFile(t *testing.T) {
	t.Parallel()

	const (
		pkgFx = "package fixtures\n\n"
		pkgGf = "package fixtures\n\n" + newImp
	)
	testCases := []repoCase{
		dependantClean(
			"field and var qualify the moved names", "fx",
			pkgFx+"type A struct {\n\tT TemplateFixture\n}\n\nvar _ TemplateFixture = A{}.T\n",
			pkgGf+"type A struct {\n\tT genfixtures.TemplateFixture\n}\n\n"+
				"var _ genfixtures.TemplateFixture = A{}.T\n"),
		dependantClean(
			"slice of the moved name is qualified", "fx",
			pkgFx+"func G() []TemplateFixture { return nil }\n",
			pkgGf+"func G() []genfixtures.TemplateFixture { return nil }\n"),
		dependantClean(
			"staying file needs no change", "fx",
			pkgFx+"func G() int { return 1 }\n", pkgFx+"func G() int { return 1 }\n"),
		dependantFlagged(
			"shadowing parameter is not qualified", "fx",
			pkgFx+"func G(NewTemplate int) int { return NewTemplate }\n",
			pkgGf+"func G(NewTemplate int) int { return genfixtures.NewTemplate }\n"),
		dependantFlagged(
			"composite literal key is not qualified", "fx",
			pkgFx+"type A struct{ NewTemplate int }\n\nvar V = A{NewTemplate: 1}\n",
			pkgGf+"type A struct{ NewTemplate int }\n\nvar V = A{genfixtures.NewTemplate: 1}\n"),
		dependantFlagged(
			"selector is not qualified", "fx",
			pkgFx+"func G(v struct{ NewTemplate int }) int { return v.NewTemplate }\n",
			pkgGf+"func G(v struct{ NewTemplate int }) int { "+
				"return v.genfixtures.NewTemplate }\n"),
		dependantFlagged(
			"name declared by a staying file", "fx",
			pkgFx+"func Stays() int { return 1 }\n\nvar V = Stays()\n",
			pkgGf+"func Stays() int { return 1 }\n\nvar V = genfixtures.Stays()\n"),
		dependantFlagged(
			"dot import of the new directory", "fx", pkgFx+"var V = NewTemplate()\n",
			"package fixtures\n\nimport . \"example.com/m/gen/genfixtures\"\n\n"+
				"var V = NewTemplate()\n"),
		dependantFlagged(
			"qualifier is a local variable", "fx",
			pkgFx+"func G(genfixtures int) int { return NewTemplate() + genfixtures }\n",
			pkgGf+"func G(genfixtures int) int { return genfixtures.NewTemplate() "+
				"+ genfixtures }\n"),
	}
	runRepoCases(t, testCases)
}

func TestDependantRules_Comments(t *testing.T) {
	t.Parallel()

	const (
		pkgUse = "package use\n\n"
		field  = "type A struct{}\n"
	)
	testCases := []repoCase{
		dependantClean(
			"comment names the moved decl under the new qualifier", "fx",
			"package fixtures\n\n// see fixtures.NewTemplate for the factory\n"+field,
			"package fixtures\n\n// see genfixtures.NewTemplate for the factory\n"+field),
		dependantClean(
			"comment qualifier swap in an unrelated package", "use",
			pkgUse+"/* fixtures.TemplateFixture and fixtures.NewTemplate */\n"+field,
			pkgUse+"/* genfixtures.TemplateFixture and genfixtures.NewTemplate */\n"+field),
		dependantFlagged(
			"comment names a decl the move never carried", "use",
			pkgUse+"// see fixtures.Other\n"+field, pkgUse+"// see genfixtures.Other\n"+field),
		dependantFlagged(
			"comment names an unexported decl", "use",
			pkgUse+"// see fixtures.private\n"+field, pkgUse+"// see genfixtures.private\n"+field),
		dependantFlagged(
			"directive comment changes", "use",
			pkgUse+"//go:generate gen fixtures.NewTemplate\n"+field,
			pkgUse+"//go:generate gen genfixtures.NewTemplate\n"+field),
		dependantFlagged(
			"comment words change beyond the qualifier", "use",
			pkgUse+"// see fixtures.NewTemplate\n"+field,
			pkgUse+"// use genfixtures.NewTemplate\n"+field),
		dependantFlagged(
			"comment swap beside a code edit", "use",
			pkgUse+"// see fixtures.NewTemplate\nvar V = 1\n",
			pkgUse+"// see genfixtures.NewTemplate\nvar V = 2\n"),
		dependantFlagged(
			"analysistest want comment under testdata", "lint/testdata/src/a",
			pkgUse+"var V = 1 // want `fixtures.NewTemplate`\n",
			pkgUse+"var V = 1 // want `genfixtures.NewTemplate`\n"),
		inFile(dependantFlagged(
			"example output line in a test file", "use",
			exampleSrc("fixtures.NewTemplate"), exampleSrc("genfixtures.NewTemplate")),
			"example_test.go"),
		inFile(dependantClean(
			"comment outside an example in a test file", "use",
			"package use_test\n\n// see fixtures.NewTemplate\nfunc helper() {}\n",
			"package use_test\n\n// see genfixtures.NewTemplate\nfunc helper() {}\n"),
			"helper_test.go"),
	}
	runRepoCases(t, testCases)
}

func exampleSrc(last string) string {
	return "package use_test\n\nfunc ExampleName() {\n\tfmt.Println(\"fixtures.NewTemplate\")\n" +
		"\t// Output:\n\t// " + last + "\n}\n"
}
