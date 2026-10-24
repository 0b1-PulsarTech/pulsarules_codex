package dartsrc

import (
	"cmp"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	roots := Roots{"ledger": ".", "scale": "packages/scale", "twin": ""}
	testCases := []struct {
		name       string
		importer   string
		uri        string
		wantTarget string
		wantOK     bool
	}{
		{"sibling", "lib/src/a.dart", "b.dart", "lib/src/b.dart", true},
		{"parent", "lib/src/a.dart", "../b.dart", "lib/b.dart", true},
		{"escapes the repository", "a.dart", "../b.dart", "", false},
		{"root package", "lib/a.dart", "package:ledger/src/b.dart", "lib/src/b.dart", true},
		{"nested package", "a.dart", "package:scale/s.dart", "packages/scale/lib/s.dart", true},
		{"unknown package", "a.dart", "package:other/s.dart", "", false},
		{"name two pubspecs claim", "a.dart", "package:twin/s.dart", "", false},
		{"package without path", "a.dart", "package:scale", "", false},
		{"dart scheme", "a.dart", "dart:async", "", false},
		{"file scheme", "a.dart", "file:///x/b.dart", "", false},
		{"absolute path", "a.dart", "/b.dart", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, isOK := Resolve(testCase.importer, testCase.uri, roots)
			if got != testCase.wantTarget || isOK != testCase.wantOK {
				t.Fatalf("Resolve() = %q, %v, want %q, %v",
					got, isOK, testCase.wantTarget, testCase.wantOK)
			}
		})
	}
}

type equivCase struct {
	name         string
	oldSrc       string
	newSrc       string
	oldPath      string
	newPath      string
	roots        Roots
	newRoots     Roots
	wantRewrites int
	isSame       bool
}

func TestEquivalent(t *testing.T) {
	t.Parallel()

	runEquivCases(t, []equivCase{
		{
			isSame:  true,
			name:    "identical",
			oldSrc:  "import 'a.dart';\nvoid f() {}\n",
			oldPath: "lib/x.dart",
		},
		{
			isSame: true,
			name:   "relative fixup after a move", oldSrc: "import '../a.dart';\n",
			newSrc: "import '../../a.dart';\n", oldPath: "lib/src/x.dart",
			newPath: "lib/src/deep/x.dart", wantRewrites: 1,
		},
		{
			isSame: true,
			name:   "package uri follows the target", oldSrc: "import 'package:ledger/old.dart';\n",
			newSrc: "import 'package:ledger/fresh.dart';\n", oldPath: "lib/x.dart", wantRewrites: 1,
		},
		{
			isSame:       true,
			name:         "re-sorted after rewrite",
			oldSrc:       "import 'package:ledger/old.dart';\nimport 'package:ledger/m.dart';\n",
			newSrc:       "import 'package:ledger/m.dart';\nimport 'package:ledger/fresh.dart';\n",
			oldPath:      "lib/x.dart",
			wantRewrites: 1,
		},
		{
			isSame:  true,
			name:    "re-sort alone rewrites nothing",
			oldSrc:  "import 'a.dart';\nimport 'b.dart';\n",
			newSrc:  "import 'b.dart';\nimport 'a.dart';\n",
			oldPath: "lib/x.dart",
		},
		{
			isSame:       true,
			name:         "conditional uris all follow",
			oldSrc:       "import 'a.dart' if (dart.library.io) 'package:ledger/old.dart';\n",
			newSrc:       "import 'a.dart' if (dart.library.io) 'package:ledger/fresh.dart';\n",
			oldPath:      "lib/x.dart",
			wantRewrites: 1,
		},
		{
			isSame: true,
			name:   "part of follows the parent", oldSrc: "part of '../main.dart';\n",
			newSrc: "part of '../../main.dart';\n", oldPath: "lib/src/p.dart",
			newPath: "lib/src/deep/p.dart", wantRewrites: 1,
		},
	})
}

func TestEquivalent_Forms(t *testing.T) {
	t.Parallel()

	runEquivCases(t, []equivCase{
		{
			isSame: true,
			name:   "relative becomes package inside lib", oldSrc: "import '../a.dart';\n",
			newSrc: "import 'package:ledger/a.dart';\n", oldPath: "lib/src/x.dart",
			newPath: "lib/src/deep/x.dart", wantRewrites: 1,
		},
		{
			isSame:       true,
			name:         "package becomes relative inside lib",
			oldSrc:       "import 'package:ledger/a.dart';\n",
			newSrc:       "import '../../a.dart';\n",
			oldPath:      "lib/src/x.dart",
			newPath:      "lib/src/deep/x.dart",
			wantRewrites: 1,
		},
		{
			isSame:  true,
			name:    "unresolvable relative uri stays in one directory",
			oldSrc:  "import '../c.dart';\n",
			oldPath: "a.dart",
			newPath: "b.dart",
		},
		{
			isSame: true,
			name:   "unknown package and dart uri stay across a move",
			oldSrc: "import 'package:other/o.dart';\nimport 'dart:async';\n",
			newSrc: "import 'package:other/o.dart';\nimport 'dart:async';\n", oldPath: "lib/x.dart",
			newPath: "lib/deep/x.dart",
		},
	})
}

func TestEquivalent_Rejections(t *testing.T) {
	t.Parallel()

	runEquivCases(t, []equivCase{
		{
			name:    "body edit beside a fix",
			oldSrc:  "import 'package:ledger/old.dart';\nvoid f() {}\n",
			newSrc:  "import 'package:ledger/fresh.dart';\nvoid g() {}\n",
			oldPath: "lib/x.dart",
		},
		{
			name: "unrelated library", oldSrc: "import 'package:ledger/old.dart';\n",
			newSrc: "import 'package:ledger/other.dart';\n", oldPath: "lib/x.dart",
		},
		{
			name: "added import", oldSrc: "import 'a.dart';\n",
			newSrc: "import 'a.dart';\nimport 'b.dart';\n", oldPath: "lib/x.dart",
		},
		{
			name: "added import with no space", oldSrc: "import 'x.dart';",
			newSrc: "import 'x.dart';import 'y.dart';", oldPath: "lib/x.dart",
		},
		{
			name:    "duplicate directive is not a different one",
			oldSrc:  "import 'a.dart';import 'a.dart';",
			newSrc:  "import 'a.dart';import 'b.dart';",
			oldPath: "lib/x.dart",
		},
		{
			name: "dart uri cannot change", oldSrc: "import 'dart:async';\n",
			newSrc: "import 'dart:io';\n", oldPath: "lib/x.dart",
		},
		{
			name: "show clause differs", oldSrc: "import 'a.dart' show A;\n",
			newSrc: "import 'a.dart' show B;\n", oldPath: "lib/x.dart",
		},
		{
			name:    "conditional uri count differs",
			oldSrc:  "import 'a.dart' if (dart.library.io) 'b.dart';\n",
			newSrc:  "import 'a.dart' if (dart.library.io) 'b.dart' if (dart.library.html) 'c.dart';\n",
			oldPath: "lib/x.dart",
		},
		{
			name:    "package from test is not the relative path to lib",
			oldSrc:  "import 'package:ledger/a.dart';\n",
			newSrc:  "import '../../lib/a.dart';\n",
			oldPath: "test/x.dart",
			newPath: "test/deep/x.dart",
		},
		{
			name:    "relative from bin is not the package path",
			oldSrc:  "import '../lib/a.dart';\n",
			newSrc:  "import 'package:ledger/a.dart';\n",
			oldPath: "bin/x.dart",
			newPath: "bin/deep/x.dart",
		},
	})
}

func TestEquivalent_Rejections_Links(t *testing.T) {
	t.Parallel()

	runEquivCases(t, []equivCase{
		{
			name:    "importer leaves lib while swapping to the package form",
			oldSrc:  "import '../a.dart';\n",
			newSrc:  "import 'package:ledger/a.dart';\n",
			oldPath: "lib/src/x.dart",
			newPath: "test/x.dart",
		},
		{
			name:    "importer enters lib while swapping to the package form",
			oldSrc:  "import '../lib/a.dart';\n",
			newSrc:  "import 'package:ledger/a.dart';\n",
			oldPath: "test/x.dart",
			newPath: "lib/src/x.dart",
		},
		{
			name:    "package name changes",
			oldSrc:  "import 'package:ledger/a.dart';\n",
			newSrc:  "import 'package:alias/a.dart';\n",
			oldPath: "lib/x.dart",
			newPath: "lib/deep/x.dart",
		},
		{
			name:    "unmoved importer swaps an unmoved target's uri form",
			oldSrc:  "import 'a.dart';\n",
			newSrc:  "import 'package:ledger/a.dart';\n",
			oldPath: "lib/x.dart",
		},
		{
			name: "unchanged relative uri resolves elsewhere", oldSrc: "import 'g.dart';\n",
			newSrc: "import 'g.dart';\n", oldPath: "lib/a/f.dart", newPath: "lib/b/c/f.dart",
		},
		{
			name:    "unchanged package uri to a moved target",
			oldSrc:  "import 'package:ledger/old.dart';\n",
			newSrc:  "import 'package:ledger/old.dart';\n",
			oldPath: "lib/x.dart",
		},
		{
			name:    "unchanged relative uri escapes the root differently",
			oldSrc:  "import '../../b.dart';\n",
			newSrc:  "import '../../b.dart';\n",
			oldPath: "x/a.dart",
			newPath: "y/a.dart",
		},
		{
			name: "unchanged relative uri starts to resolve", oldSrc: "import '../b.dart';\n",
			newSrc: "import '../b.dart';\n", oldPath: "a.dart", newPath: "sub/a.dart",
		},
		{
			name: "package a pubspec lost", oldSrc: "import 'package:ledger/a.dart';\n",
			newSrc: "import 'package:ledger/a.dart';\n", oldPath: "lib/x.dart", newRoots: Roots{},
		},
		{
			name: "package two pubspecs claim", oldSrc: "import 'package:twin/a.dart';\n",
			newSrc: "import 'package:twin/a.dart';\n", oldPath: "lib/x.dart",
			roots: Roots{"twin": ""},
		},
		{
			name:    "rewritten through a name two pubspecs claim",
			oldSrc:  "import 'package:twin/old.dart';\n",
			newSrc:  "import 'package:twin/fresh.dart';\n",
			oldPath: "lib/x.dart",
			roots:   Roots{"twin": ""},
		},
	})
}

func runEquivCases(t *testing.T, testCases []equivCase) {
	t.Helper()

	ledgerRoots := Roots{"ledger": ".", "alias": "."}
	movedTarget := map[string]string{"lib/old.dart": "lib/fresh.dart"}

	follows := func(oldTarget, newTarget string) bool {
		if moved, isMoved := movedTarget[oldTarget]; isMoved {
			return newTarget == moved
		}
		return newTarget == oldTarget
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			newSrc, newPath := cmp.Or(
				testCase.newSrc,
				testCase.oldSrc,
			), cmp.Or(
				testCase.newPath,
				testCase.oldPath,
			)
			oldRoots, newRoots := testCase.roots, testCase.newRoots
			if oldRoots == nil {
				oldRoots = ledgerRoots
			}
			if newRoots == nil {
				newRoots = oldRoots
			}
			oldFile, oldErr := Parse(testCase.oldSrc)
			newFile, newErr := Parse(newSrc)
			if oldErr != nil || newErr != nil {
				t.Fatalf("Parse() errors: %v, %v", oldErr, newErr)
			}
			pair := Pair{
				Old: Source{
					File:  oldFile,
					Path:  testCase.oldPath,
					Roots: func() Roots { return oldRoots },
				},
				New: Source{File: newFile, Path: newPath, Roots: func() Roots { return newRoots }},
			}
			got, rewrites := Equivalent(pair, follows)
			if got != testCase.isSame {
				t.Fatalf("Equivalent() = %v, want %v", got, testCase.isSame)
			}
			if testCase.isSame && rewrites != testCase.wantRewrites {
				t.Fatalf("rewrites = %d, want %d", rewrites, testCase.wantRewrites)
			}
		})
	}
}
