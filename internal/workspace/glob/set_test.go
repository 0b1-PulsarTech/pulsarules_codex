package glob

import "testing"

func TestSetMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sources []string
		file    string
		want    bool
	}{
		{name: "no sources match everything", file: "any/file.go", want: true},
		{name: "glob hit", sources: []string{"web/test/**"}, file: "web/test/a.dart", want: true},
		{name: "glob miss", sources: []string{"web/test/**"}, file: "web/lib/b.dart"},
		{name: "second glob hit", sources: []string{"web/**", "*.go"}, file: "x.go", want: true},
		{name: "invalid glob matches nothing", sources: []string{"a//b"}, file: "a/b"},
		{
			name: "invalid beside valid", sources: []string{"a//b", "lib/*"}, file: "lib/x",
			want: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := CompileSet(testCase.sources).Match(testCase.file); got != testCase.want {
				t.Errorf("Match(%q) = %v, want %v", testCase.file, got, testCase.want)
			}
		})
	}
}

func TestSetIsOpen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sources []string
		want    bool
	}{
		{name: "no sources", want: true},
		{name: "one glob", sources: []string{"lib/*"}},
		{name: "invalid glob", sources: []string{"a//b"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := CompileSet(testCase.sources).IsOpen(); got != testCase.want {
				t.Errorf("IsOpen() = %v, want %v", got, testCase.want)
			}
		})
	}
}
