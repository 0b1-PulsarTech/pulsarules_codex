package movepurity

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	dartOld   = "lib/src/draft.dart"
	dartFresh = "lib/src/legacy/draft.dart"
)

func dartTree(files map[string]string) map[string]string {
	files["pubspec.yaml"] = "name: ledger\n"
	return files
}

func dartMove(name, oldSrc, newSrc string) renameCase {
	return renameCase{
		name:    name,
		head:    dartTree(map[string]string{dartOld: oldSrc}),
		staged:  dartTree(map[string]string{dartFresh: newSrc}),
		renames: []pass.Rename{pair(dartOld, dartFresh)},
	}
}

func TestDartEdit_MovedFile(t *testing.T) {
	t.Parallel()

	body := "\nclass Draft {}\n"
	testCases := []renameCase{
		accepted(dartMove("relative imports fixed after the move",
			"import '../expired.dart';\nimport 'dart:async';\n"+body,
			"import '../../expired.dart';\nimport 'dart:async';\n"+body)),
		accepted(dartMove("package import is untouched",
			"import 'package:ledger/src/expired.dart';\n"+body,
			"import 'package:ledger/src/expired.dart';\n"+body)),
		accepted(dartMove("directives re-sorted after a fixup",
			"import '../b.dart';\nimport '../a.dart';\n"+body,
			"import '../../a.dart';\nimport '../../b.dart';\n"+body)),
		dartMove("import switched to an unrelated library",
			"import '../expired.dart';\n"+body, "import '../../scale.dart';\n"+body),
		dartMove("import added", "import '../expired.dart';\n"+body,
			"import '../../expired.dart';\nimport '../../scale.dart';\n"+body),
		dartMove("body edit next to an import fix", "import '../expired.dart';\n"+body,
			"import '../../expired.dart';\n"+body+"class Extra {}\n"),
		dartMove(
			"dart uri changed",
			"import 'dart:async';\n"+body,
			"import 'dart:io';\n"+body,
		),
		accepted(dartMove("conditional import follows",
			"import '../a.dart' if (dart.library.io) '../b.dart';\n"+body,
			"import '../../a.dart' if (dart.library.io) '../../b.dart';\n"+body)),
		accepted(dartMove("part of follows the parent move", "part of '../main.dart';\n"+body,
			"part of '../../main.dart';\n"+body)),
		dartMove(
			"unparsable source",
			"import '../a.dart'\n"+body,
			"import '../../a.dart'\n"+body,
		),
		{
			name: "dart file renamed to a non-dart name",
			head: dartTree(map[string]string{dartOld: "import '../a.dart';\n"}),
			staged: dartTree(
				map[string]string{"lib/src/legacy/draft.txt": "import '../../a.dart';\n"},
			),
			renames: []pass.Rename{pair(dartOld, "lib/src/legacy/draft.txt")},
		},
		{
			name:    "missing package roots make a package import an edit",
			head:    map[string]string{dartOld: "import 'package:ledger/a.dart';\n"},
			staged:  map[string]string{dartFresh: "import 'package:ledger/b.dart';\n"},
			renames: []pass.Rename{pair(dartOld, dartFresh)},
		},
		{
			name:     "unlistable tree",
			head:     dartTree(map[string]string{dartOld: "import 'package:ledger/a.dart';\n"}),
			staged:   dartTree(map[string]string{dartFresh: "import 'package:ledger/b.dart';\n"}),
			failTree: ".",
			renames:  []pass.Rename{pair(dartOld, dartFresh)},
		},
	}
	runRenameCases(t, testCases)
}

func TestDartEdit_Dependants(t *testing.T) {
	t.Parallel()

	user := "lib/user.dart"
	moved := []pass.Rename{pair("lib/old.dart", "lib/fresh.dart")}
	row := func(name, newUser string, wantFindings int) repoCase {
		return repoCase{
			name: name,
			head: dartTree(map[string]string{
				"lib/old.dart": "class A {}\n",
				user:           "import 'package:ledger/old.dart';\n\nclass B {}\n",
			}),
			staged: dartTree(map[string]string{
				"lib/fresh.dart": "class A {}\n",
				user:             newUser,
			}),
			renames:      moved,
			changed:      []pass.FileChange{{Path: user, Staged: true}},
			wantFindings: wantFindings,
		}
	}
	runRepoCases(t, []repoCase{
		row(
			"package import follows the move",
			"import 'package:ledger/fresh.dart';\n\nclass B {}\n",
			0,
		),
		row("relative import follows the move", "import 'fresh.dart';\n\nclass B {}\n", 0),
		row(
			"import pointed at an unrelated library",
			"import 'package:ledger/other.dart';\n\nclass B {}\n",
			1,
		),
		row("body edit beside the fix", "import 'package:ledger/fresh.dart';\n\nclass C {}\n", 1),
	})
}

func TestDartEdit_DependantResort(t *testing.T) {
	t.Parallel()

	user := "lib/user.dart"
	unsorted := "import 'package:ledger/other.dart';\nimport 'package:ledger/fresh.dart';\n"
	row := func(name, oldUser, newUser string, wantFindings int) repoCase {
		return repoCase{
			name: name,
			head: dartTree(map[string]string{
				"lib/old.dart": "class A {}\n", user: oldUser,
			}),
			staged: dartTree(map[string]string{
				"lib/fresh.dart": "class A {}\n", user: newUser,
			}),
			renames:      []pass.Rename{pair("lib/old.dart", "lib/fresh.dart")},
			changed:      []pass.FileChange{{Path: user, Staged: true}},
			wantFindings: wantFindings,
		}
	}
	runRepoCases(t, []repoCase{
		row("re-sort with no rewrite is an edit", unsorted,
			"import 'package:ledger/fresh.dart';\nimport 'package:ledger/other.dart';\n", 1),
		row("re-sort beside a rewrite is a consequence",
			"import 'package:ledger/other.dart';\nimport 'package:ledger/old.dart';\n",
			"import 'package:ledger/fresh.dart';\nimport 'package:ledger/other.dart';\n", 0),
		row("unmoved target's uri form changed", "import 'other.dart';\n",
			"import 'package:ledger/other.dart';\n", 1),
	})
}

func TestDartEdit_Rejections(t *testing.T) {
	t.Parallel()

	body := "\nclass Draft {}\n"
	testCases := []renameCase{
		dartMove("library called like a function",
			"library() { return 'a.dart'; }"+body, "library() { return 'b.dart'; }"+body),
		dartMove("callable part hides a body edit", "part(x) => '../expired.dart';\n"+body,
			"part(x) => '../../expired.dart';\n"+body),
		dartMove("callable export hides a body edit", "export() => '../expired.dart';\n"+body,
			"export() => '../../expired.dart';\n"+body),
		dartMove("percent escape in a package path", "import 'package:ledger/a%2Fb.dart';\n"+body,
			"import 'package:ledger/a%2Fb.dart';\n"+body),
		dartMove("dot segment in a package path", "import 'package:ledger/x/../a.dart';\n"+body,
			"import 'package:ledger/x/../a.dart';\n"+body),
		dartMove("second literal outside a condition", "import 'a.dart' 'b.dart';\n"+body,
			"import 'a.dart' 'b.dart';\n"+body),
	}
	runRenameCases(t, testCases)
}

func TestDartEdit_Rejections_Layout(t *testing.T) {
	t.Parallel()

	body := "\nclass Draft {}\n"
	twinTree := func(files map[string]string) map[string]string {
		files["packages/one/pubspec.yaml"] = "name: ledger\n"
		files["packages/two/pubspec.yaml"] = "name: ledger\n"
		return files
	}
	noPubspec := func(name, oldSrc, newSrc string) renameCase {
		row := dartMove(name, oldSrc, newSrc)
		row.head, row.staged = map[string]string{
			dartOld: oldSrc,
		}, map[string]string{
			dartFresh: newSrc,
		}
		return row
	}
	testCases := []renameCase{
		{
			name: "package from test is not the relative path to lib",
			head: dartTree(
				map[string]string{"test/draft.dart": "import 'package:ledger/a.dart';\n"},
			),
			staged: dartTree(
				map[string]string{"test/deep/draft.dart": "import '../../lib/a.dart';\n"},
			),
			renames: []pass.Rename{pair("test/draft.dart", "test/deep/draft.dart")},
		},
		{
			name: "relative from bin is not the package path",
			head: dartTree(map[string]string{"bin/draft.dart": "import '../lib/a.dart';\n"}),
			staged: dartTree(
				map[string]string{"bin/deep/draft.dart": "import 'package:ledger/a.dart';\n"},
			),
			renames: []pass.Rename{pair("bin/draft.dart", "bin/deep/draft.dart")},
		},
		{
			name: "package name two pubspecs claim",
			head: twinTree(map[string]string{
				"packages/two/lib/draft.dart": "import 'package:ledger/old.dart';\n",
			}),
			staged: twinTree(map[string]string{
				"packages/two/lib/legacy/draft.dart": "import 'package:ledger/fresh.dart';\n",
			}),
			renames: []pass.Rename{
				pair("packages/two/lib/draft.dart", "packages/two/lib/legacy/draft.dart"),
				pair("packages/two/lib/old.dart", "packages/two/lib/fresh.dart"),
			},
		},
		noPubspec(
			"package import with no pubspec at all",
			"import 'package:ledger/old.dart';\n"+body,
			"import 'package:ledger/fresh.dart';\n"+body,
		),
	}
	runRenameCases(t, testCases)
}

func TestDartEdit_Rejections_Reads(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name:    "dart source renamed from a non-dart name",
			head:    dartTree(map[string]string{"lib/src/notes.txt": "import '../a.dart';\n"}),
			staged:  dartTree(map[string]string{dartFresh: "import '../../a.dart';\n"}),
			renames: []pass.Rename{pair("lib/src/notes.txt", dartFresh)},
		},
		{
			name:    "old blob cannot be read",
			head:    dartTree(map[string]string{dartOld: "import '../a.dart';\n"}),
			staged:  dartTree(map[string]string{dartFresh: "import '../../a.dart';\n"}),
			renames: []pass.Rename{pair(dartOld, dartFresh)}, failBlob: dartOld,
		},
		{
			name:    "new blob cannot be read",
			head:    dartTree(map[string]string{dartOld: "import '../a.dart';\n"}),
			staged:  dartTree(map[string]string{dartFresh: "import '../../a.dart';\n"}),
			renames: []pass.Rename{pair(dartOld, dartFresh)}, failStaged: dartFresh,
		},
		{
			name:    "old source cannot be parsed and the new one is empty",
			head:    dartTree(map[string]string{dartOld: "import '../a.dart'\n"}),
			staged:  dartTree(map[string]string{dartFresh: ""}),
			renames: []pass.Rename{pair(dartOld, dartFresh)},
		},
		{
			name: "directives only re-sorted by a move",
			head: dartTree(
				map[string]string{dartOld: "import 'dart:io';\nimport 'dart:async';\n"},
			),
			staged: dartTree(
				map[string]string{dartFresh: "import 'dart:async';\nimport 'dart:io';\n"},
			),
			renames: []pass.Rename{pair(dartOld, dartFresh)},
		},
	}
	runRenameCases(t, testCases)
}

func TestDartEdit_Pubspec(t *testing.T) {
	t.Parallel()

	importer := "import 'package:ledger/a.dart';\n"
	moves := []pass.Rename{
		pair("lib/draft.dart", "app/lib/draft.dart"),
		pair("pubspec.yaml", "app/pubspec.yaml"),
	}
	spec := "name: ledger\n"
	testCases := []renameCase{
		accepted(renameCase{
			name: "package moves with its target",
			head: map[string]string{
				"lib/draft.dart": importer, "lib/a.dart": "class A {}\n", "pubspec.yaml": spec,
			},
			staged: map[string]string{
				"app/lib/draft.dart": importer, "app/lib/a.dart": "class A {}\n",
				"app/pubspec.yaml": spec,
			},
			renames: append(slices.Clone(moves), pair("lib/a.dart", "app/lib/a.dart")),
		}),
		{
			name: "package moves but its target stays behind",
			head: map[string]string{
				"lib/draft.dart": importer, "lib/a.dart": "class A {}\n", "pubspec.yaml": spec,
			},
			staged: map[string]string{
				"app/lib/draft.dart": importer, "lib/a.dart": "class A {}\n",
				"app/pubspec.yaml": spec,
			},
			renames: moves,
		},
		{
			name: "package renamed in place",
			head: map[string]string{
				"lib/draft.dart": importer, "lib/a.dart": "class A {}\n", "pubspec.yaml": spec,
			},
			staged: map[string]string{
				"lib/legacy/draft.dart": importer, "lib/a.dart": "class A {}\n",
				"pubspec.yaml": "name: scale\n",
			},
			renames: []pass.Rename{pair("lib/draft.dart", "lib/legacy/draft.dart")},
		},
	}
	runRenameCases(t, testCases)
}

func TestDartIdentical(t *testing.T) {
	t.Parallel()

	identical := func(name, from, to, src string, want verdict) renameCase {
		return renameCase{
			name:        name,
			head:        dartTree(map[string]string{from: src}),
			staged:      dartTree(map[string]string{to: src}),
			renames:     []pass.Rename{pair(from, to)},
			wantVerdict: want,
		}
	}
	testCases := []renameCase{
		identical("relative import resolves to another file", "lib/a/f.dart", "lib/b/c/f.dart",
			"import 'g.dart';\n", verdictUnsafe),
		identical(
			"relative import to a parent still resolves there",
			"lib/a/f.dart",
			"lib/b/f.dart",
			"import '../g.dart';\n",
			verdictPure,
		),
		identical("malformed percent escape", "lib/a/f.dart", "lib/b/f.dart",
			"import 'package:ledger/a%2Fb.dart';\n", verdictUnsafe),
		identical("package and dart imports only", "lib/a/f.dart", "lib/b/c/f.dart",
			"import 'package:ledger/g.dart';\nimport 'dart:io';\n", verdictPure),
		identical("renamed in place keeps its relative imports", "lib/a/f.dart", "lib/a/h.dart",
			"import 'g.dart';\n", verdictPure),
		{
			name: "sibling an in-place rename leaves behind",
			head: dartTree(
				map[string]string{
					"lib/a/f.dart": "import 'g.dart';\n",
					"lib/a/g.dart": "class G {}\n",
				},
			),
			staged: dartTree(
				map[string]string{
					"lib/a/h.dart": "import 'g.dart';\n",
					"lib/b/g.dart": "class G {}\n",
				},
			),
			renames: []pass.Rename{
				pair("lib/a/f.dart", "lib/a/h.dart"),
				pair("lib/a/g.dart", "lib/b/g.dart"),
			},
			wantVerdict: verdictUnsafe,
		},
		identical(
			"no directives at all",
			"lib/a/f.dart",
			"lib/b/f.dart",
			"class F {}\n",
			verdictPure,
		),
	}
	runVerdictCases(t, testCases)
}

func TestDartIdentical_Reads(t *testing.T) {
	t.Parallel()

	testCases := []renameCase{
		{
			name: "empty file whose old blob cannot be read", failBlob: "lib/a/f.dart",
			head:        dartTree(map[string]string{"lib/a/f.dart": ""}),
			staged:      dartTree(map[string]string{"lib/b/f.dart": ""}),
			renames:     []pass.Rename{pair("lib/a/f.dart", "lib/b/f.dart")},
			wantVerdict: verdictUnsafe,
		},
		{
			name: "empty file whose new blob cannot be read", failStaged: "lib/b/f.dart",
			head:        dartTree(map[string]string{"lib/a/f.dart": ""}),
			staged:      dartTree(map[string]string{"lib/b/f.dart": ""}),
			renames:     []pass.Rename{pair("lib/a/f.dart", "lib/b/f.dart")},
			wantVerdict: verdictUnsafe,
		},
	}
	runVerdictCases(t, testCases)
}

func runVerdictCases(t *testing.T, testCases []renameCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			moves := newMoveContext(testCase.repo(), testCase.renames)
			if got := moves.renameVerdict(testCase.renames[0]); got != testCase.wantVerdict {
				t.Fatalf("renameVerdict() = %v, want %v", got, testCase.wantVerdict)
			}
		})
	}
}
