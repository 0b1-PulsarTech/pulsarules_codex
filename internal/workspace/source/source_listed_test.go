package source

import (
	"slices"
	"testing"
)

type recordingProvider struct {
	files map[string]string
	reads []string
}

func (p *recordingProvider) Read(path string) ([]byte, bool) {
	p.reads = append(p.reads, path)
	content, ok := p.files[path]
	return []byte(content), ok
}

func (p *recordingProvider) Walk(func(path string, ext string) bool) {}

func TestListedProvider_Walk(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		paths    []string
		stopAt   int
		wantPath []string
		wantExt  []string
	}{
		{
			"order preserved",
			[]string{"b.go", "a/c.MD"},
			0,
			[]string{"b.go", "a/c.MD"},
			[]string{".go", ".md"},
		},
		{
			"skip set applied",
			[]string{"a.go", "vendor/x.go", "node_modules/y/z.js"},
			0,
			[]string{"a.go"},
			[]string{".go"},
		},
		{
			"stops on false",
			[]string{"a.go", "b.go", "c.go"},
			2,
			[]string{"a.go", "b.go"},
			[]string{".go", ".go"},
		},
		{"empty list", nil, 0, nil, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider := NewListedProvider(
				&recordingProvider{},
				testCase.paths,
				NewSkipSet(DefaultSkipDirs()),
			)
			var gotPaths, gotExts []string
			provider.Walk(func(path, ext string) bool {
				gotPaths = append(gotPaths, path)
				gotExts = append(gotExts, ext)
				return testCase.stopAt == 0 || len(gotPaths) < testCase.stopAt
			})
			if !slices.Equal(gotPaths, testCase.wantPath) ||
				!slices.Equal(gotExts, testCase.wantExt) {
				t.Fatalf(
					"walk = %v %v, want %v %v",
					gotPaths,
					gotExts,
					testCase.wantPath,
					testCase.wantExt,
				)
			}
		})
	}
}

func TestListedProvider_ReadDelegates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		path   string
		want   string
		wantOK bool
	}{
		{"known file", "a/b.go", "package b\n", true},
		{"unlisted but known to base", "other.go", "package o\n", true},
		{"missing file", "a/missing.go", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := &recordingProvider{files: map[string]string{
				"a/b.go":   "package b\n",
				"other.go": "package o\n",
			}}
			provider := NewListedProvider(base, []string{"a/b.go"}, NewSkipSet(nil))
			got, ok := provider.Read(testCase.path)
			if ok != testCase.wantOK || string(got) != testCase.want {
				t.Fatalf("Read = %q, %v, want %q, %v", got, ok, testCase.want, testCase.wantOK)
			}
			if !slices.Equal(base.reads, []string{testCase.path}) {
				t.Fatalf("base reads = %v, want [%s]", base.reads, testCase.path)
			}
		})
	}
}
