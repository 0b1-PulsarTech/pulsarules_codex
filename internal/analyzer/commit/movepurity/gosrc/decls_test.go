package gosrc

import "testing"

func TestPackageClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		src         string
		wantName    string
		wantIgnored bool
	}{
		{"plain", "package p\n\nfunc F() {}\n", "p", false},
		{"ignored", "//go:build ignore\n\npackage p\n", "p", true},
		{"negated ignore", "//go:build !ignore\n\npackage p\n", "p", false},
		{"ignore with other tag", "//go:build ignore && linux\n\npackage p\n", "p", true},
		{"plus build ignore", "// +build ignore\n\npackage p\n", "p", true},
		{"ignore or linux", "//go:build ignore || linux\n\npackage p\n", "p", false},
		{"linux and ignore negated", "//go:build !linux && ignore\n\npackage p\n", "p", true},
		{"linux and not cgo", "//go:build linux && !cgo\n\npackage p\n", "p", false},
		{"go version range", "//go:build go1.20 && !go1.99\n\npackage p\n", "p", false},
		{"amd64 and not purego", "//go:build amd64 && !purego\n\npackage p\n", "p", false},
		{"unrelated tag", "//go:build linux\n\npackage p\n", "p", false},
		{"doc comment only", "// Package p does x.\npackage p\n", "p", false},
		{
			"go:build wins over plus build",
			"//go:build linux\n// +build ignore\n\npackage p\n",
			"p",
			false,
		},
		{"plus build inside the package doc", "// +build ignore\npackage p\n", "p", false},
		{"no clause", "func F() {}\n", "", false},
		{"empty", "", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			name, isIgnored := PackageClause(testCase.src)
			if name != testCase.wantName || isIgnored != testCase.wantIgnored {
				t.Fatalf("PackageClause() = %q, %v, want %q, %v",
					name, isIgnored, testCase.wantName, testCase.wantIgnored)
			}
		})
	}
}
