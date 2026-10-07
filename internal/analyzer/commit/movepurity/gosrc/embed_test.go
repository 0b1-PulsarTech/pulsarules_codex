package gosrc

import (
	"errors"
	"go/parser"
	"go/token"
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestEmbedPatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want []string
	}{
		{"none", "package p\n\n// go:embed notes\nvar V int\n", nil},
		{"one literal", embedSrc("//go:embed public.txt"), []string{"public.txt"}},
		{
			"several on a line and a second directive",
			"package p\n\n//go:embed a.txt  b/*.txt\n//go:embed c.txt\nvar V string\n",
			[]string{"a.txt", "b/*.txt", "c.txt"},
		},
		{
			"quoted with a space",
			embedSrc("//go:embed \"my file.txt\" `raw.txt`"),
			[]string{"my file.txt", "raw.txt"},
		},
		{"unterminated quote kept raw", embedSrc("//go:embed \"broken"), []string{"\"broken"}},
		{"a longer directive name is not an embed", embedSrc("//go:embedded x"), nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(
				token.NewFileSet(),
				"",
				testCase.src,
				parser.ParseComments,
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := embedPatterns(file); !slices.Equal(got, testCase.want) {
				t.Fatalf("embedPatterns() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func embedSrc(directive string) string {
	return "package p\n\nimport _ \"embed\"\n\n" + directive + "\nvar Data string\n"
}

func TestEmbedCovers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		patterns []string
		rel      string
		want     bool
	}{
		{"no patterns", nil, "a.txt", false},
		{"literal file", []string{"a.txt"}, "a.txt", true},
		{"literal file elsewhere", []string{"a.txt"}, "b/a.txt", false},
		{"directory prefix", []string{"static"}, "static/css/a.css", true},
		{"sibling with a shared prefix", []string{"static"}, "static2/a.css", false},
		{"glob over the base name", []string{"*.txt"}, "a.txt", true},
		{"glob does not cross a slash", []string{"*.txt"}, "b/a.txt", false},
		{"glob over a directory", []string{"s*"}, "static/a.css", true},
		{"hidden file under a plain directory", []string{"static"}, "static/.keep", false},
		{"underscore directory under a plain directory", []string{"static"}, "static/_x/a", false},
		{"all prefix takes a hidden file", []string{"all:static"}, "static/.keep", true},
		{"all prefix takes an underscore directory", []string{"all:static"}, "static/_x/a", true},
		{"glob names a hidden file", []string{"static/*"}, "static/.keep", true},
		{"explicit hidden name", []string{"static/.keep"}, "static/.keep", true},
		{"malformed pattern", []string{"[a"}, "a", false},
		{"any pattern may cover", []string{"x", "static"}, "static/a", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := EmbedCovers(testCase.patterns, testCase.rel); got != testCase.want {
				t.Fatalf(
					"EmbedCovers(%q, %q) = %v, want %v",
					testCase.patterns, testCase.rel, got, testCase.want,
				)
			}
		})
	}
}

func treeChildren(files []string, listed *[]string, failDir string) func(string) ([]string, error) {
	return func(dir string) ([]string, error) {
		*listed = append(*listed, dir)
		if dir == failDir {
			return nil, errors.New("listing failed")
		}
		prefix := dir + "/"
		if dir == "." {
			prefix = ""
		}
		var names []string
		for _, file := range files {
			if rest, isUnder := strings.CutPrefix(file, prefix); isUnder {
				first, _, _ := strings.Cut(rest, "/")
				if !slices.Contains(names, prefix+first) {
					names = append(names, prefix+first)
				}
			}
		}
		return names, nil
	}
}

func TestEmbedFiles(t *testing.T) {
	t.Parallel()

	tree := []string{
		"pkg/a.txt", "pkg/b.go", "pkg/static/x.js", "pkg/static/.keep", "pkg/static/sub/z.js",
		"pkg/static/_gen/g.js", "pkg/big/deep/one", "pkg/big/deep/two", "other/o.txt", "pkg/lone",
	}
	testCases := []struct {
		name       string
		patterns   []string
		root       string
		failDir    string
		want       []string
		wantUnseen string
		wantErr    bool
	}{
		{
			"directory pattern",
			[]string{"static"},
			"pkg",
			"",
			[]string{"static/sub/z.js", "static/x.js"},
			"pkg/big", false,
		},
		{
			"all prefix takes hidden names",
			[]string{"all:static"},
			"pkg",
			"",
			[]string{"static/.keep", "static/_gen/g.js", "static/sub/z.js", "static/x.js"},
			"pkg/big", false,
		},
		{
			"glob over the base name",
			[]string{"*.txt"},
			"pkg",
			"",
			[]string{"a.txt"},
			"pkg/big", false,
		},
		{
			"glob below a literal directory",
			[]string{"static/*.js"},
			"pkg",
			"",
			[]string{"static/x.js"},
			"pkg/big", false,
		},
		{
			"literal file below directories",
			[]string{"big/deep/one"},
			"pkg",
			"",
			[]string{"big/deep/one"},
			"pkg/static", false,
		},
		{"module root", []string{"pkg/a.txt"}, ".", "", []string{"pkg/a.txt"}, "other", false},
		{"nothing matches", []string{"missing"}, "pkg", "", nil, "", false},
		{"root with no files is not a file", []string{"gone"}, "gone", "", nil, "", false},
		{"file where a pattern wants a directory", []string{"lone/x"}, "pkg", "", nil, "", false},
		{"listing fails", []string{"static"}, "pkg", "pkg/static", nil, "", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var listed []string
			got, err := EmbedFiles(
				testCase.patterns, testCase.root, treeChildren(tree, &listed, testCase.failDir),
			)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("EmbedFiles() error = %v, want error %v", err, testCase.wantErr)
			}
			if gotFiles := slices.Sorted(maps.Keys(got)); !slices.Equal(gotFiles, testCase.want) {
				t.Fatalf("EmbedFiles() = %q, want %q", gotFiles, testCase.want)
			}
			if testCase.wantUnseen != "" && slices.Contains(listed, testCase.wantUnseen) {
				t.Fatalf("listed %q, want it pruned: %q", testCase.wantUnseen, listed)
			}
		})
	}
}
