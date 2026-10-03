package filecount

import "testing"

func TestInspectSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want srcInfo
	}{
		{"plain", "package s\n", srcInfo{pkg: "s"}},
		{"build ignore", "//go:build ignore\n\npackage main\n", srcInfo{pkg: "main", skip: true}},
		{"build other tag", "//go:build linux\n\npackage s\n", srcInfo{pkg: "s"}},
		{"build ignore or linux", "//go:build ignore || linux\n\npackage s\n", srcInfo{pkg: "s"}},
		{"build ignore after package", "package s\n\n//go:build ignore\n", srcInfo{pkg: "s"}},
		{
			"legacy build ignore",
			"// +build ignore\n\npackage main\n",
			srcInfo{pkg: "main", skip: true},
		},
		{"legacy build other tag", "// +build linux\n\npackage s\n", srcInfo{pkg: "s"}},
		{
			"legacy ignore with go build linux",
			"//go:build linux\n// +build ignore\n\npackage s\n",
			srcInfo{pkg: "s"},
		},
		{"legacy ignore after package", "package s\n\n// +build ignore\n", srcInfo{pkg: "s"}},
		{
			"build ignore after package in a fully parsed file",
			"package s\n\nimport \"entgo.io/ent\"\n\n//go:build ignore\n",
			srcInfo{pkg: "s"},
		},
		{"unparsable", "import \"x\"\n", srcInfo{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := inspectSource("s.go", []byte(testCase.src)); got != testCase.want {
				t.Fatalf("inspectSource = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

type constraintFormTestCase struct {
	name string
	src  string
	want srcInfo
}

func constraintFormCases() []constraintFormTestCase {
	return []constraintFormTestCase{
		{
			"build ignore and linux",
			"//go:build ignore && linux\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"build linux and ignore",
			"//go:build linux && ignore\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"build nested and chain with ignore",
			"//go:build (linux && amd64) && ignore\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{"build not ignore", "//go:build !ignore\n\npackage s\n", srcInfo{pkg: "s"}},
		{
			"build ignore touching the package clause",
			"//go:build ignore\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"build ignore inside the package doc",
			"// Package s doc.\n//go:build ignore\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"legacy ignore then go build linux touching the package clause",
			"// +build ignore\n\n//go:build linux\npackage s\n",
			srcInfo{pkg: "s"},
		},
		{
			"legacy ignore comma linux",
			"// +build ignore,linux\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{"legacy ignore space linux", "// +build ignore linux\n\npackage s\n", srcInfo{pkg: "s"}},
		{
			"legacy two lines ignore first",
			"// +build ignore\n// +build linux\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"legacy two lines ignore last",
			"// +build linux\n// +build ignore\n\npackage s\n",
			srcInfo{pkg: "s", skip: true},
		},
		{
			"legacy ignore touching the package clause",
			"// +build ignore\npackage s\n",
			srcInfo{pkg: "s"},
		},
		{
			"legacy ignore inside the package doc",
			"// Package s does things.\n// +build ignore\npackage s\n",
			srcInfo{pkg: "s"},
		},
	}
}

func TestInspectSourceConstraintForms(t *testing.T) {
	t.Parallel()

	for _, testCase := range constraintFormCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := inspectSource("s.go", []byte(testCase.src)); got != testCase.want {
				t.Fatalf("inspectSource = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestAnalyzePackageScope(t *testing.T) {
	t.Parallel()

	runFileCountCases(t, []fileCountCase{
		{
			name: "build ignore file not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/gen.go": "//go:build ignore\n\npackage main\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "legacy build ignore file not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/gen.go": "// +build ignore\n\npackage p\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "file with another package name not counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/doc.go": "package other\n",
			}),
			changed: []string{"pkg/f00.go"},
		},
		{
			name: "unparsable file still counted",
			files: merged(numbered("pkg", 10), map[string]string{
				"pkg/bad.go": "import \"x\"\n",
			}),
			changed: []string{"pkg/f00.go"},
			want:    []string{"pkg/f00.go:1:package pkg has 11 non-test source files, max 10"},
		},
		{
			name: "package tie resolves to the same count",
			files: merged(numbered("pkg", 6), map[string]string{
				"pkg/a1.go": "package a\n", "pkg/a2.go": "package a\n", "pkg/a3.go": "package a\n",
				"pkg/a4.go": "package a\n", "pkg/a5.go": "package a\n", "pkg/a6.go": "package a\n",
			}),
			changed: []string{"pkg/f00.go"},
			params:  map[string]any{"max_files": 5},
			want:    []string{"pkg/f00.go:1:package pkg has 6 non-test source files, max 5"},
		},
	})
}
