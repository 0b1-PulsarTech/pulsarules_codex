package glob

import (
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{name: "plain", pattern: "web/**/x.mjs"},
		{name: "leading dot slash", pattern: "./lib/*.go"},
		{name: "class", pattern: "lib/[ab].go"},
		{name: "unterminated class", pattern: "lib/[a.go", wantErr: true},
		{name: "dangling escape", pattern: "lib/a\\", wantErr: true},
		{name: "bad class in later segment", pattern: "**/[", wantErr: true},
		{name: "empty", pattern: "", wantErr: true},
		{name: "only dot slash", pattern: "./", wantErr: true},
		{name: "leading slash", pattern: "/lib/a.go", wantErr: true},
		{name: "trailing slash", pattern: "lib/", wantErr: true},
		{name: "doubled slash", pattern: "lib//a.go", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Compile(testCase.pattern)
			if err != nil && !strings.Contains(err.Error(), testCase.pattern) {
				t.Fatalf("error %q does not name the pattern", err)
			}
			if (err != nil) != testCase.wantErr {
				t.Fatalf(
					"Compile(%q) error = %v, wantErr %v",
					testCase.pattern,
					err,
					testCase.wantErr,
				)
			}
		})
	}
}

type matchCase struct {
	name    string
	pattern string
	path    string
	want    bool
}

func matchCases() []matchCase {
	return []matchCase{
		{
			name:    "leading double star deep",
			pattern: "**/x_test.go",
			path:    "a/b/x_test.go",
			want:    true,
		},
		{name: "leading double star zero", pattern: "**/x_test.go", path: "x_test.go", want: true},
		{name: "leading double star miss", pattern: "**/x_test.go", path: "a/y_test.go"},
		{name: "middle double star", pattern: "web/**/x.mjs", path: "web/a/b/x.mjs", want: true},
		{name: "middle double star zero", pattern: "web/**/x.mjs", path: "web/x.mjs", want: true},
		{name: "middle double star wrong root", pattern: "web/**/x.mjs", path: "app/x.mjs"},
		{name: "trailing double star", pattern: "lib/**", path: "lib/a/b.go", want: true},
		{name: "trailing double star zero", pattern: "lib/**", path: "lib", want: true},
		{name: "trailing double star miss", pattern: "lib/**", path: "other/a.go"},
		{name: "star stays in segment", pattern: "lib/*.go", path: "lib/a/b.go"},
		{name: "star matches name", pattern: "lib/*.go", path: "lib/b.go", want: true},
		{name: "question mark", pattern: "lib/?.go", path: "lib/a.go", want: true},
		{name: "question mark too long", pattern: "lib/?.go", path: "lib/ab.go"},
		{name: "question mark not slash", pattern: "lib?a.go", path: "lib/a.go"},
		{name: "class hit", pattern: "lib/[ab].go", path: "lib/b.go", want: true},
		{name: "class miss", pattern: "lib/[ab].go", path: "lib/c.go"},
		{name: "leading dot slash pattern", pattern: "./lib/*.go", path: "lib/a.go", want: true},
		{name: "leading dot slash path", pattern: "lib/*.go", path: "./lib/a.go", want: true},
		{name: "path longer than pattern", pattern: "lib/a.go", path: "lib/a.go/b"},
		{name: "path shorter than pattern", pattern: "lib/a/b.go", path: "lib/a"},
		{name: "consecutive double stars", pattern: "**/**/a.go", path: "a.go", want: true},
		{name: "double star only", pattern: "**", path: "a/b/c", want: true},
		{name: "negated class hit", pattern: "[!_]*.dart", path: "a.dart", want: true},
		{name: "negated class miss", pattern: "[!_]*.dart", path: "_a.dart"},
		{name: "negated class in dir", pattern: "lib/[!_]*.dart", path: "lib/_a.dart"},
		{name: "escaped bracket literal bang", pattern: `\[!b]`, path: "[!b]", want: true},
		{name: "escaped bracket not caret", pattern: `\[!b]`, path: "[^b]"},
		{name: "escaped bracket mid pattern", pattern: `a\[!b`, path: "a[!b", want: true},
		{name: "bang inside class stays", pattern: "[[!]", path: "!", want: true},
		{name: "bang inside class not caret", pattern: "[[!]", path: "^"},
		{name: "negated class after prefix hit", pattern: "a[!b]", path: "ac", want: true},
		{name: "negated class after prefix miss", pattern: "a[!b]", path: "ab"},
		{name: "second class negates", pattern: "[a][!b]", path: "ac", want: true},
		{name: "second class negated miss", pattern: "[a][!b]", path: "ab"},
		{name: "bang outside class is literal", pattern: "!a", path: "!a", want: true},
	}
}

func TestPattern_Match(t *testing.T) {
	t.Parallel()

	for _, testCase := range matchCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pattern, err := Compile(testCase.pattern)
			if err != nil {
				t.Fatalf("Compile(%q): %v", testCase.pattern, err)
			}
			if got := pattern.Match(testCase.path); got != testCase.want {
				t.Fatalf(
					"%q.Match(%q) = %v, want %v",
					testCase.pattern,
					testCase.path,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestPattern_Match_ManyDoubleStars(t *testing.T) {
	t.Parallel()

	segments := strings.Repeat("a/", 29) + "a"
	testCases := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "worst case miss", pattern: strings.Repeat("**/", 10) + "x", path: segments},
		{
			name:    "worst case hit",
			pattern: strings.Repeat("**/", 10) + "x",
			path:    segments + "/x",
			want:    true,
		},
		{
			name:    "interleaved miss",
			pattern: strings.Repeat("**/a/", 14) + "x",
			path:    segments + strings.Repeat("/a", 30),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pattern, err := Compile(testCase.pattern)
			if err != nil {
				t.Fatalf("Compile(%q): %v", testCase.pattern, err)
			}
			if got := pattern.Match(testCase.path); got != testCase.want {
				t.Fatalf("Match = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCompile_RequiredSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		want    int
	}{
		{name: "only double star", pattern: "**", want: 0},
		{name: "plain path", pattern: "web/lib/x.mjs", want: 3},
		{name: "double stars between", pattern: "**/a/**/**/b", want: 2},
		{name: "wildcard segment", pattern: "lib/*/*.go", want: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pattern, err := Compile(testCase.pattern)
			if err != nil {
				t.Fatalf("Compile(%q): %v", testCase.pattern, err)
			}
			if pattern.required != testCase.want {
				t.Fatalf("required = %d, want %d", pattern.required, testCase.want)
			}
		})
	}
}

//nolint:paralleltest // AllocsPerRun panics inside a parallel test.
func TestPattern_Match_ShortPathSkipsTheMemo(t *testing.T) {
	pattern, err := Compile(strings.Repeat("**/a/", 20) + "x")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	allocs := testing.AllocsPerRun(20, func() {
		if pattern.Match("a/b/x") {
			t.Fatal("short path matched")
		}
	})
	if allocs > 1 {
		t.Fatalf("Match allocated %v times, want at most 1", allocs)
	}
}
