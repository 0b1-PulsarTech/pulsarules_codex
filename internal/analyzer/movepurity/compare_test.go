package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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

	moveA := []core.Rename{pair(utilA, routingA)}
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
		importRow(
			"unrelated fmt import added",
			"func F() {}\n",
			"import \"fmt\"\n\nfunc F() {}\n",
		),
		importRow(
			"cgo import added",
			"func F() int { return 1 }\n",
			"import \"C\"\n\nfunc F() int { return 1 }\n",
		),
		importRow(
			"dot import swapped",
			"import . \"example.com/m/internal/util\"\n",
			"import . \"example.com/m/internal/routing\"\n",
		),
	}
	runRenameCases(t, testCases)
}

func rewriteRow(name, alias, call string, dirMove ...core.Rename) renameCase {
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

	moveA := []core.Rename{pair(utilA, routingA)}
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
			renames: []core.Rename{pair(utilA, routingA), pair("x/k.go", "y/k.go")},
		},
	}
	runRenameCases(t, testCases)
}

const (
	utilStay = "internal/util/stay.go"
	linkBody = "import _ \"example.com/m/internal/routing\"\n"
	initBody = "func init() { register() }\n"
)
