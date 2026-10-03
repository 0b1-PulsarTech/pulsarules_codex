package movepurity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	utilGoMod    = "internal/util/c.go"
	routingMain  = "internal/routing/r.go"
	useFoo       = "func F() int { return Foo() }\n"
	useMax       = "func F() int { return max(1, 2) }\n"
	routingOther = "package routing\n\nconst Other = 1\n"
	oldUtilSrc   = "package util\n\n" + useFoo
)

func keptBody(dep string) string {
	return "import \"example.com/m/" + dep + "\"\n\nfunc F() int { return Foo() + old.K }\n"
}

func pkgFile(pkg, body string) string { return "package " + pkg + "\n\n" + body }

func moveF(extra ...pass.Rename) []pass.Rename {
	return append([]pass.Rename{pair(utilA, routingA)}, extra...)
}

func TestCaptures_RejectedBuiltins(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "test helper captured",
			head: map[string]string{
				"internal/util/a_test.go": pkgFile(
					"util",
					"func TestA() int { return setup() }\n",
				),
				"internal/util/helpers_test.go": pkgFile(
					"util", "func setup() int { return 1 }\n",
				),
				routingMain: routingOther,
			},
			staged: map[string]string{
				"internal/routing/a_test.go": pkgFile(
					"routing",
					"func TestA() int { return setup() }\n",
				),
				"internal/util/helpers_test.go": pkgFile(
					"util",
					"func setup() int { return 1 }\n",
				),
				"internal/routing/helpers_test.go": pkgFile(
					"routing",
					"func setup() int { return 2 }\n",
				),
				routingMain: routingOther,
			},
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
		},
		{
			name: "builtin len captured by a newDir declaration",
			head: map[string]string{
				utilA: pkgFile("util", "func A(s string) int { return len(s) }\n"),
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "func A(s string) int { return len(s) }\n"),
				routingMain: pkgFile("routing", "func len(s string) int { return 7 }\n"),
			},
			renames: moveF(),
		},
		{
			name: "builtin un-shadowed by the move",
			head: map[string]string{
				utilA:     pkgFile("util", useMax),
				utilGoMod: pkgFile("util", "func max(a, b int) int { return a }\n"),
			},
			staged: map[string]string{
				routingA:  pkgFile("routing", useMax),
				utilGoMod: pkgFile("util", "func max(a, b int) int { return a }\n"),
			},
			renames: moveF(),
		},
		{
			name: "S3b builtin un-shadowed when the shadow moves to a third dir",
			head: map[string]string{
				utilA:                pkgFile("util", useMax),
				"internal/util/m.go": pkgFile("util", "func max(a, b int) int { return a }\n"),
			},
			staged: map[string]string{
				routingA:              pkgFile("routing", useMax),
				"internal/other/m.go": pkgFile("other", "func max(a, b int) int { return a }\n"),
			},
			renames: moveF(pair("internal/util/m.go", "internal/other/m.go")),
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_RejectedDeclarations(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "S3c test sibling shadows the builtin for a moved test",
			head: map[string]string{
				"internal/util/a_test.go": pkgFile("util", useMax),
				"internal/util/x_test.go": pkgFile(
					"util", "func max(a, b int) int { return a }\n",
				),
				routingMain: routingOther,
			},
			staged: map[string]string{
				"internal/routing/a_test.go": pkgFile("routing", useMax),
				"internal/util/x_test.go": pkgFile(
					"util",
					"func max(a, b int) int { return a }\n",
				),
				routingMain: routingOther,
			},
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
		},
		{
			name: "declaration moved to a third dir while newDir has its own",
			head: map[string]string{
				utilA:                pkgFile("util", useFoo),
				"internal/util/f.go": pkgFile("util", "func Foo() int { return 1 }\n"),
			},
			staged: map[string]string{
				routingA:          pkgFile("routing", useFoo),
				"internal/r/f.go": pkgFile("r", "func Foo() int { return 1 }\n"),
				routingMain:       pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(pair("internal/util/f.go", "internal/r/f.go")),
		},
		{
			name: "newDir declares a name oldDir never did",
			head: map[string]string{utilA: pkgFile("util", useFoo)},
			staged: map[string]string{
				routingA: pkgFile(
					"routing",
					useFoo,
				),
				routingMain: pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(),
		},
		{
			name: "a third-dir file moved into newDir declares the name",
			head: map[string]string{
				utilA:                 pkgFile("util", useFoo),
				"internal/util/f.go":  pkgFile("util", "func Foo() int { return 1 }\n"),
				"internal/other/x.go": pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			staged: map[string]string{
				routingA:                pkgFile("routing", useFoo),
				"internal/util/f.go":    pkgFile("util", "func Foo() int { return 1 }\n"),
				"internal/routing/x.go": pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(pair("internal/other/x.go", "internal/routing/x.go")),
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_AcceptedDangling(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "new package declares nothing of that name",
			head: map[string]string{
				utilA:     oldUtilSrc,
				utilGoMod: pkgFile("util", "const Foo = 1\n"),
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", useFoo),
				utilGoMod:   pkgFile("util", "const Foo = 1\n"),
				routingMain: pkgFile("routing", "const Other = 1\n"),
			},
			renames: moveF(),
			want:    true,
		},
		{
			name:    "bare name declared by neither package",
			head:    map[string]string{utilA: oldUtilSrc},
			staged:  map[string]string{routingA: pkgFile("routing", useFoo)},
			renames: moveF(),
			want:    true,
		},
		{
			name: "sibling moves along",
			head: map[string]string{
				utilA:     oldUtilSrc,
				utilGoMod: pkgFile("util", "const Foo = 1\n"),
			},
			staged: map[string]string{
				routingA:                pkgFile("routing", useFoo),
				"internal/routing/c.go": pkgFile("routing", "const Foo = 1\n"),
			},
			renames: moveF(pair(utilGoMod, "internal/routing/c.go")),
			want:    true,
		},
		{
			name: "a shadowing sibling moves along",
			head: map[string]string{
				utilA:     pkgFile("util", useMax),
				utilGoMod: pkgFile("util", "func max(a, b int) int { return a }\n"),
			},
			staged: map[string]string{
				routingA: pkgFile("routing", useMax),
				"internal/routing/c.go": pkgFile(
					"routing",
					"func max(a, b int) int { return a }\n",
				),
			},
			renames: moveF(pair(utilGoMod, "internal/routing/c.go")),
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_AcceptedScopes(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "a test file shadowing the builtin does not touch a non-test move",
			head: map[string]string{
				utilA: pkgFile("util", useMax),
				"internal/util/x_test.go": pkgFile(
					"util", "func max(a, b int) int { return a }\n",
				),
			},
			staged: map[string]string{
				routingA: pkgFile("routing", useMax),
				"internal/util/x_test.go": pkgFile(
					"util", "func max(a, b int) int { return a }\n",
				),
			},
			renames: moveF(),
			want:    true,
		},
		{
			name: "an internal test package does not shadow for an external test move",
			head: map[string]string{
				"internal/util/a_test.go": pkgFile("util_test", useMax),
				"internal/util/x_test.go": pkgFile(
					"util", "func max(a, b int) int { return a }\n",
				),
				routingMain: pkgFile("routing", "const Other = 1\n"),
			},
			staged: map[string]string{
				"internal/routing/a_test.go": pkgFile("routing_test", useMax),
				"internal/util/x_test.go": pkgFile(
					"util",
					"func max(a, b int) int { return a }\n",
				),
				routingMain: pkgFile("routing", "const Other = 1\n"),
			},
			renames: []pass.Rename{pair("internal/util/a_test.go", "internal/routing/a_test.go")},
			want:    true,
		},
		{
			name: "selectors and methods are not bare names",
			head: map[string]string{
				utilA: pkgFile("util", "func F(v T) int { return v.Foo() + v.Foo }\n"),
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "func F(v T) int { return v.Foo() + v.Foo }\n"),
				routingMain: pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(),
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_AcceptedBareNames(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "a local variable is not a package-level name",
			head: map[string]string{
				utilA: pkgFile("util", "func F() int {\n\tFoo := 1\n\treturn Foo\n}\n"),
			},
			staged: map[string]string{
				routingA:    pkgFile("routing", "func F() int {\n\tFoo := 1\n\treturn Foo\n}\n"),
				routingMain: pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(),
			want:    true,
		},
		{
			name: "a same-directory rename keeps a bare sibling name while an import is rewritten",
			head: map[string]string{
				utilA: pkgFile(
					"util",
					"import \"example.com/m/old\"\n\nfunc F() int { return Foo() + old.K }\n",
				),
				utilGoMod:  pkgFile("util", "func Foo() int { return 1 }\n"),
				"old/k.go": pkgFile("old", "const K = 1\n"),
			},
			staged: map[string]string{
				"internal/util/b.go": pkgFile(
					"util",
					"import \"example.com/m/fresh\"\n\nfunc F() int { return Foo() + old.K }\n",
				),
				utilGoMod:    pkgFile("util", "func Foo() int { return 1 }\n"),
				"fresh/k.go": pkgFile("old", "const K = 1\n"),
			},
			renames: []pass.Rename{
				pair(utilA, "internal/util/b.go"),
				pair("old/k.go", "fresh/k.go"),
			},
			want: true,
		},
		{
			name: "a non-Go file in newDir declares nothing",
			head: map[string]string{utilA: oldUtilSrc},
			staged: map[string]string{
				routingA: pkgFile(
					"routing",
					useFoo,
				),
				"internal/routing/notes.txt": pkgFile("routing", "func Foo() int { return 2 }\n"),
			},
			renames: moveF(),
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_LiteralKeys(t *testing.T) {
	t.Parallel()

	keyed := func(literal string) map[string]string {
		return map[string]string{utilA: pkgFile("util", literal)}
	}
	takenConst := pkgFile("routing", "const Key = \"b\"\n")
	testCases := []renameCase{
		{
			name: "map key captured by a newDir declaration",
			head: keyed("var M = map[string]int{Key: 1}\n"),
			staged: map[string]string{
				routingA:    pkgFile("routing", "var M = map[string]int{Key: 1}\n"),
				routingMain: takenConst,
			},
			renames: moveF(),
		},
		{
			name: "array index key captured by a newDir declaration",
			head: keyed("var N = [...]string{Key: \"zero\"}\n"),
			staged: map[string]string{
				routingA:    pkgFile("routing", "var N = [...]string{Key: \"zero\"}\n"),
				routingMain: takenConst,
			},
			renames: moveF(),
		},
		{
			name: "map key that a left-behind declaration shadows in the universe",
			head: map[string]string{
				utilA:     pkgFile("util", "var M = map[bool]string{true: \"yes\"}\n"),
				utilGoMod: pkgFile("util", "const true = false\n"),
			},
			staged: map[string]string{
				routingA:  pkgFile("routing", "var M = map[bool]string{true: \"yes\"}\n"),
				utilGoMod: pkgFile("util", "const true = false\n"),
			},
			renames: moveF(),
		},
		{
			name: "struct field key that no package declares",
			head: keyed("var S = T{Name: 1}\n"),
			staged: map[string]string{
				routingA:    pkgFile("routing", "var S = T{Name: 1}\n"),
				routingMain: routingOther,
			},
			renames: moveF(),
			want:    true,
		},
	}
	runRenameCases(t, testCases)
}

func TestCaptures_CacheKeys(t *testing.T) {
	t.Parallel()

	const useHelper = "func F() int { return helper() }\n"
	head := map[string]string{
		"a/x.go":      pkgFile("util", "func G() int { return 1 }\n"),
		"a/x_test.go": pkgFile("util", useHelper),
		"b/h_test.go": pkgFile("routing", "func helper() int { return 2 }\n"),
	}
	staged := map[string]string{
		"b/x.go":      pkgFile("routing", "func G() int { return 1 }\n"),
		"b/x_test.go": pkgFile("routing", useHelper),
		"b/h_test.go": pkgFile("routing", "func helper() int { return 2 }\n"),
	}
	renames := []pass.Rename{pair("a/x.go", "b/x.go"), pair("a/x_test.go", "b/x_test.go")}
	moves := newMoveContext(newFakeRepo(head, staged), renames)
	if moves.renameVerdict(renames[0]) != verdictPure {
		t.Fatalf("renameVerdict(%v) = false, want true", renames[0])
	}
	if moves.renameVerdict(renames[1]) == verdictPure {
		t.Fatalf(
			"renameVerdict(%v) = true, want false: a test helper captures the name",
			renames[1],
		)
	}
}

func TestCaptures_RejectedBareNames(t *testing.T) {
	t.Parallel()

	const (
		mixedImport = "import \"example.com/m/lib\"\n\nvar V = winlib + lib.F()\n"
		winlibSrc   = "package winlib\n\nfunc F() string { return \"w\" }\n"
		libSrc      = "package lib\n\nfunc F() string { return \"l\" }\n"
	)
	testCases := []renameCase{
		{
			name: "a bare name that a mixed-package import also names",
			head: map[string]string{
				utilA:              pkgFile("util", mixedImport),
				utilGoMod:          pkgFile("util", "var winlib = \"a\"\n"),
				"lib/a_windows.go": winlibSrc,
				"lib/b_linux.go":   libSrc,
			},
			staged: map[string]string{
				routingA:           pkgFile("routing", mixedImport),
				utilGoMod:          pkgFile("util", "var winlib = \"a\"\n"),
				routingMain:        pkgFile("routing", "var winlib = \"b\"\n"),
				"lib/a_windows.go": winlibSrc,
				"lib/b_linux.go":   libSrc,
			},
			renames: moveF(),
		},
		{
			name: "package clause kept: newDir's declaration still captures a bare name",
			head: map[string]string{
				"a/x.go":   pkgFile("shared", keptBody("old")),
				"b/k.go":   pkgFile("shared", "const Other = 1\n"),
				"old/k.go": pkgFile("old", "const K = 1\n"),
			},
			staged: map[string]string{
				"b/x.go":     pkgFile("shared", keptBody("fresh")),
				"b/k.go":     pkgFile("shared", "const Other = 1\nfunc Foo() int { return 2 }\n"),
				"fresh/k.go": pkgFile("old", "const K = 1\n"),
			},
			renames: []pass.Rename{pair("a/x.go", "b/x.go"), pair("old/k.go", "fresh/k.go")},
		},
	}
	runRenameCases(t, testCases)
}
