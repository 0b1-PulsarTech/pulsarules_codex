package gomod

import (
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"slices"
	"strings"
	"testing"
)

type buildIgnoreCase struct {
	name string
	src  string
	want bool
}

func goBuildIgnoreCases() []buildIgnoreCase {
	return []buildIgnoreCase{
		{name: "no constraint", src: "package a\n", want: false},
		{name: "go:build ignore", src: "//go:build ignore\n\npackage a\n", want: true},
		{name: "not ignore", src: "//go:build !ignore\n\npackage a\n", want: false},
		{name: "and with ignore", src: "//go:build linux && ignore\n\npackage a\n", want: true},
		{name: "or with ignore", src: "//go:build linux || ignore\n\npackage a\n", want: false},
		{name: "plus build ignore", src: "// +build ignore\n\npackage a\n", want: true},
		{
			name: "go:build wins over plus build",
			src:  "//go:build linux\n// +build ignore\n\npackage a\n",
			want: false,
		},
		{
			name: "plus build in the package doc",
			src:  "// +build ignore\npackage a\n",
			want: false,
		},
		{
			name: "go:build ignore touching the package clause",
			src:  "//go:build ignore\npackage a\n",
			want: true,
		},
		{
			name: "go:build ignore inside the package doc",
			src:  "// Package a doc.\n//go:build ignore\npackage a\n",
			want: true,
		},
		{
			name: "nested and chain",
			src:  "//go:build (linux && amd64) && ignore\n\npackage a\n",
			want: true,
		},
		{name: "ignore and linux", src: "//go:build ignore && linux\n\npackage a\n", want: true},
	}
}

func plusBuildIgnoreCases() []buildIgnoreCase {
	return []buildIgnoreCase{
		{name: "legacy comma", src: "// +build ignore,linux\n\npackage a\n", want: true},
		{name: "legacy space is or", src: "// +build ignore linux\n\npackage a\n", want: false},
		{
			name: "legacy two lines ignore last",
			src:  "// +build linux\n// +build ignore\n\npackage a\n",
			want: true,
		},
		{
			name: "legacy ignore then go:build linux",
			src:  "// +build ignore\n\n//go:build linux\npackage a\n",
			want: false,
		},
		{name: "constraint after package", src: "package a\n\n//go:build ignore\n", want: false},
		{
			name: "plus build after a block comment",
			src:  "/* c */\n\n// +build ignore\n\npackage a\n",
			want: false,
		},
		{
			name: "plus build after a block comment and a blank line",
			src:  "// +build linux\n\n/* c */\n\n// +build ignore\n\npackage a\n",
			want: false,
		},
		{
			name: "plus build above a block comment",
			src:  "// +build ignore\n\n/* c */\n\npackage a\n",
			want: true,
		},
		{
			name: "go:build after a block comment still counts",
			src:  "/* c */\n//go:build ignore\n\npackage a\n",
			want: true,
		},
		{
			name: "double negation",
			src:  "//go:build !(!ignore || !linux)\n\npackage a\n",
			want: true,
		},
		{name: "negated ignore", src: "//go:build !ignore && linux\n\npackage a\n", want: false},
		{
			name: "negated and, ignore on the right",
			src:  "//go:build !(!linux && ignore)\n\npackage a\n",
			want: false,
		},
		{
			name: "negated and, tautology on the right",
			src:  "//go:build !(!ignore && other)\n\npackage a\n",
			want: false,
		},
		{
			name: "negated and, ignore on the left",
			src:  "//go:build !(ignore && linux)\n\npackage a\n",
			want: false,
		},
	}
}

func TestIsBuildIgnored(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(goBuildIgnoreCases(), plusBuildIgnoreCases())
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(
				token.NewFileSet(), "a.go", testCase.src, parser.ParseComments,
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := IsBuildIgnored(file); got != testCase.want {
				t.Errorf("IsBuildIgnored() = %v, want %v", got, testCase.want)
			}
			context := build.Context{
				GOOS: "linux", GOARCH: "amd64", Compiler: "gc",
				OpenFile: func(string) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(testCase.src)), nil
				},
			}
			matches, err := context.MatchFile(".", "a.go")
			if err != nil {
				t.Fatalf("MatchFile: %v", err)
			}
			if matches == testCase.want {
				t.Errorf("go/build MatchFile() = %v, disagrees with want ignored=%v",
					matches, testCase.want)
			}
		})
	}
}
