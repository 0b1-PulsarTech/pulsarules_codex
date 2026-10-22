package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestRelocationUnsafe_Rejected(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "the moved file declares a predeclared name",
			head: map[string]string{
				utilA: pkgFile("util", "func max(a, b int) int { return a }\n"),
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "func max(a, b int) int { return a }\n"),
				routingMain: pkgFile("routing", "func R() int { return max(1, 2) }\n"),
			},
			renames: moveF(),
		},
		named(movedBody("func init() { register() }\n"), "a moved file declares init"),
		{
			name: "a moved file imports C",
			head: map[string]string{
				utilA: pkgFile(
					"util",
					"// #include \"conf.h\"\nimport \"C\"\n\nfunc V() int { return int(C.VAL) }\n",
				),
			},
			staged: map[string]string{
				routingA: pkgFile(
					"routing",
					"// #include \"conf.h\"\nimport \"C\"\n\nfunc V() int { return int(C.VAL) }\n",
				),
			},
			renames: moveF(),
		},
		named(movedBody("import \"reg\"\n\nvar _ = reg.Add(\"a\")\n"),
			"a blank var initialiser calls a function"),
		named(movedBody("var _ = []int{f(1)}\n\nfunc f(n int) int { return n }\n"),
			"a call nested in a blank var value"),
		named(movedBody("var _ = I(nil)\n\ntype I interface{}\n"),
			"a named conversion cannot be told from a call"),
		named(
			movedBody("var _ = (func() bool { return f() })()\n\nfunc f() bool { return true }\n"),
			"a parenthesised function literal is invoked at load",
		),
		named(movedBody("import \"reg\"\n\nvar _ = (reg.Add)(\"a\")\n"),
			"a parenthesised selector call"),
		named(
			movedBody("type S struct{ B bool }\n\nvar _ = S{B: (func() bool { return "+
				"true })()}\n"),
			"a literal invoked inside a composite value",
		),
		named(movedBody("import _ \"example.com/reg\"\n"), "a moved file blank-imports a package"),
		named(movedBody("import \"testing\"\n\nfunc TestMain(m *testing.M) { m.Run() }\n"),
			"a moved file declares TestMain"),
	}
	runRenameCases(t, testCases)
}

func movedBody(body string) renameCase {
	return renameCase{
		head:    map[string]string{utilA: pkgFile("util", body), utilStay: "package util\n"},
		staged:  map[string]string{routingA: pkgFile("routing", body), utilStay: "package util\n"},
		renames: moveF(),
	}
}

func named(row renameCase, name string) renameCase {
	row.name = name
	return row
}

func TestRelocationUnsafe_Accepted(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "a same-directory rename may import C",
			head: map[string]string{
				utilA: pkgFile("util", "import \"C\"\n\nvar V = C.VAL\n"),
			},
			staged: map[string]string{
				"internal/util/b.go": pkgFile("util", "import \"C\"\n\nvar V = C.VAL\n"),
			},
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
		{
			name: "a same-directory rename may declare init",
			head: map[string]string{
				utilA: pkgFile("util", "func init() { register() }\n"),
			},
			staged: map[string]string{
				"internal/util/b.go": pkgFile("util", "func init() { register() }\n"),
			},
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestRelocationUnsafe_SameDirectory(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "a same-directory rename is no relocation",
			head: map[string]string{
				utilA:     pkgFile("util", useFoo),
				utilGoMod: pkgFile("util", "func Foo() int { return 1 }\n"),
			},
			staged: map[string]string{
				"internal/util/b.go": pkgFile("util", useFoo),
				utilGoMod:            pkgFile("util", "func Foo() int { return 1 }\n"),
			},
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
		{
			name: "a same-directory rename may keep a dot import and a builtin shadow",
			head: map[string]string{
				utilA: pkgFile("util", "import . \"fmt\"\n\n"+
					"func max(a, b int) int { return a }\n"),
			},
			staged: map[string]string{
				"internal/util/b.go": pkgFile(
					"util",
					"import . \"fmt\"\n\nfunc max(a, b int) int { return a }\n",
				),
			},
			renames: []pass.Rename{pair(utilA, "internal/util/b.go")},
			want:    true,
		},
		{
			name: "an ordinary declaration moves",
			head: map[string]string{utilA: pkgFile("util", "func Fresh() int { return 1 }\n")},
			staged: map[string]string{
				routingA: pkgFile("routing", "func Fresh() int { return 1 }\n"),
			},
			renames: moveF(),
			want:    true,
		},
		accepted(named(movedBody("type T struct{}\n\ntype I interface{}\n\nvar _ I = (*T)(nil)\n"),
			"a blank interface assertion")),
		accepted(named(movedBody("var _ = []byte(\"x\")\n\nvar _ = func() int { return 1 }\n"),
			"blank values with a literal conversion or a function literal")),
		accepted(
			named(
				movedBody(
					"import \"errors\"\n\nvar ErrX = errors.New(\"x\")\n\n"+
						"func F() error { return ErrX }\n",
				),
				"a referenced named sentinel built by a call",
			),
		),
	}
	runRenameCases(t, testCases)
}

func TestCaptures_RootDirectory(t *testing.T) {
	t.Parallel()

	const useLen = "func F(s string) int { return len(s) }\n"
	shadow := pkgFile("util", "func len(string) int { return 42 }\n")
	testCases := []renameCase{
		{
			name: "a move into the repo root keeps an unmoved sibling's shadow",
			head: map[string]string{
				utilA:     pkgFile("util", useLen),
				utilGoMod: shadow,
				"root.go": pkgFile("m", "const Other = 1\n"),
			},
			staged: map[string]string{
				"a.go":    pkgFile("m", useLen),
				utilGoMod: shadow,
				"root.go": pkgFile("m", "const Other = 1\n"),
			},
			renames: []pass.Rename{pair(utilA, "a.go")},
		},
		{
			name: "a move into the repo root with the shadow moved along",
			head: map[string]string{
				utilA:     pkgFile("util", useLen),
				utilGoMod: shadow,
				"root.go": pkgFile("m", "const Other = 1\n"),
			},
			staged: map[string]string{
				"a.go":    pkgFile("m", useLen),
				"c.go":    pkgFile("m", "func len(string) int { return 42 }\n"),
				"root.go": pkgFile("m", "const Other = 1\n"),
			},
			renames: []pass.Rename{pair(utilA, "a.go"), pair(utilGoMod, "c.go")},
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestRelocationUnsafe_RejectedDotImports(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "dot import",
			head: map[string]string{
				utilA: pkgFile(
					"util",
					"import . \"example.com/m/dot\"\n\nfunc F() int { return Foo() }\n",
				),
				"dot/d.go": pkgFile("dot", "func Bar() int { return 0 }\n"),
			},
			staged: map[string]string{
				routingA: pkgFile(
					"routing",
					"import . \"example.com/m/dot\"\n\nfunc F() int { return Foo() }\n",
				),
				"dot/d.go": pkgFile("dot", "func Bar() int { return 0 }\n"),
			},
			renames: moveF(),
		},
	}
	runRenameCases(t, testCases)
}

func TestTakesOverName(t *testing.T) {
	t.Parallel()

	const (
		routingOld = "internal/routing/old.go"
		fooSrc     = "func Foo() int { return 1 }\n"
	)
	testCases := []renameCase{
		{
			name: "the moved-in file takes a name whose owner moved away",
			head: map[string]string{
				utilA:       pkgFile("util", fooSrc),
				routingOld:  pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			staged: map[string]string{
				routingA:     pkgFile("routing", fooSrc),
				"other/o.go": pkgFile("other", fooSrc),
				routingMain:  routingOther,
			},
			renames: moveF(pair(routingOld, "other/o.go")),
		},
		{
			name: "the moved-in file takes a name whose owner was deleted",
			head: map[string]string{
				utilA:       pkgFile("util", fooSrc),
				routingOld:  pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			renames: moveF(),
		},
		{
			name: "a blank identifier is no name",
			head: map[string]string{
				utilA:       pkgFile("util", "var _ = 1\n\nfunc F() {}\n"),
				routingOld:  pkgFile("routing", "var _ = 2\n"),
				routingMain: routingOther,
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "var _ = 1\n\nfunc F() {}\n"),
				routingMain: routingOther,
			},
			renames: moveF(),
			want:    true,
		},
		{
			name: "the previous owner stays",
			head: map[string]string{
				utilA:       pkgFile("util", "func Bar() int { return 1 }\n"),
				routingOld:  pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "func Bar() int { return 1 }\n"),
				routingOld:  pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			renames: moveF(),
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestTakesOverName_OwnerKinds(t *testing.T) {
	t.Parallel()

	const (
		routingOld = "internal/routing/old.go"
		fooSrc     = "func Foo() int { return 1 }\n"
	)
	testCases := []renameCase{
		{
			name: "the owner of the name declares another package clause",
			head: map[string]string{
				utilA:       pkgFile("util", fooSrc),
				routingOld:  pkgFile("elsewhere", fooSrc),
				routingMain: routingOther,
			},
			staged: map[string]string{
				routingA:     pkgFile("routing", fooSrc),
				"other/o.go": pkgFile("elsewhere", fooSrc),
				routingMain:  routingOther,
			},
			renames: moveF(pair(routingOld, "other/o.go")),
		},
		{
			name: "a production file takes a name from an internal test file that left",
			head: map[string]string{
				utilA:                        pkgFile("util", fooSrc),
				"internal/routing/x_test.go": pkgFile("routing", fooSrc),
				routingMain:                  routingOther,
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", fooSrc),
				routingMain: routingOther,
			},
			renames: moveF(),
		},
		{
			name: "a test file takes a name from a production file that left",
			head: map[string]string{
				"internal/util/a_test.go": pkgFile("util", fooSrc),
				routingOld:                pkgFile("routing", fooSrc),
				routingMain:               routingOther,
			},
			staged: map[string]string{
				"internal/routing/a_test.go": pkgFile("routing", fooSrc),
				routingMain:                  routingOther,
			},
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
		},
	}
	runRenameCases(t, testCases)
}

func sameDirSrc(dir string) string {
	return pkgFile(
		"util",
		"import (\n\t_ \"embed\"\n\n\t\"example.com/m/"+dir+"\"\n)\n\n"+
			"//go:embed public.txt\nvar Data string\n\nvar K = old.K\n",
	)
}

func TestEmbedRelocation(t *testing.T) {
	t.Parallel()

	const embedPublic, embedStatic = "//go:embed public.txt", "//go:embed static"
	const oldSub, newSub = "internal/util/static/sub/z.js", "internal/routing/static/sub/z.js"
	public := []pass.Rename{pair("internal/util/public.txt", "internal/routing/public.txt")}
	static := []pass.Rename{
		pair("internal/util/static/x.js", "internal/routing/static/x.js"),
		pair("internal/util/static/y.js", "internal/routing/static/y.js"),
	}
	strayY := pair("internal/util/static/y.js", "internal/routing/other/y.js")
	dir := func(name string, comoved []pass.Rename) renameCase {
		return embedRow(name, embedStatic, comoved)
	}
	testCases := []renameCase{
		embedRow("S19 embedded file stays behind while newDir has its own", embedPublic, nil),
		accepted(embedRow("directory move with the embedded file co-moved", embedPublic, public)),
		embedRow("glob pattern", "//go:embed *.txt", public),
		embedRow("all: prefix", "//go:embed all:public.txt", public),
		dir("directory pattern", public),
		accepted(dir("directory pattern all co-moved", static)),
		dir("directory pattern one left behind", static[:1]),
		dir("directory pattern one moved elsewhere", []pass.Rename{static[0], strayY}),
		accepted(withFile(dir("nested directory all co-moved", static), oldSub, newSub)),
		withFile(dir("nested file left behind", static), oldSub, ""),
		embedRow("one of several patterns left behind", embedPublic+" static/x.js", public),
		embedRow("unterminated quote", "//go:embed \"public.txt", public),
		accepted(embedRow("quoted literal co-moved", "//go:embed \"public.txt\"", public)),
		embedRow("embedded file renamed to another name", embedPublic, []pass.Rename{
			pair("internal/util/public.txt", "internal/routing/other.txt"),
		}),
		embedLiteralNamed("a*.txt"),
		embedLiteralNamed("a?.txt"),
		embedLiteralNamed("a[1].txt"),
		embedLiteralNamed("a\\b.txt"),
		embedLiteralNamed("all:x.txt"),
		accepted(embedRow("no embed at all", "// plain comment", nil)),
		{
			name: "same-directory rename with a rewritten import keeps its embed",
			head: map[string]string{
				utilA:                      sameDirSrc("old"),
				"internal/util/public.txt": "public\n",
				"old/k.go":                 pkgFile("old", "const K = 1\n"),
			},
			staged: map[string]string{
				"internal/util/b.go":       sameDirSrc("fresh"),
				"internal/util/public.txt": "public\n",
				"fresh/k.go":               pkgFile("old", "const K = 1\n"),
			},
			renames: []pass.Rename{
				pair(utilA, "internal/util/b.go"),
				pair("old/k.go", "fresh/k.go"),
			},
			want: true,
		},
	}
	runRenameCases(t, testCases)
}
