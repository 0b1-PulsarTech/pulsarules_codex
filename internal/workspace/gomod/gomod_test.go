package gomod

import (
	"slices"
	"testing"
)

func TestModulePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want string
	}{
		{name: "plain", src: "module example.com/m\n\ngo 1.25\n", want: "example.com/m"},
		{name: "quoted", src: "module \"example.com/q\"\n", want: "example.com/q"},
		{name: "after a comment", src: "// lead\nmodule example.com/o\n", want: "example.com/o"},
		{name: "missing directive", src: "go 1.25\n", want: ""},
		{name: "empty", src: "", want: ""},
		{
			name: "syntax error elsewhere",
			src:  "module example.com/x\n\nrequire (\n",
			want: "example.com/x",
		},
		{name: "bare module", src: "module\n", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ModulePath([]byte(testCase.src))
			if got != testCase.want {
				t.Fatalf("ModulePath() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestWorkUses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		src     string
		want    []string
		wantErr bool
	}{
		{
			name: "single lines",
			src:  "go 1.22\n\nuse ./foo\n\nuse ./bar\n",
			want: []string{"./foo", "./bar"},
		},
		{
			name: "block with comments and quoted paths",
			src:  "go 1.22\n\nuse (\n\t./a // first\n\t// skipped\n\t\"./b\"\n)\n",
			want: []string{"./a", "./b"},
		},
		{
			name: "block beside replace",
			src:  "use (\n\t./a\n\t./b\n)\n\nreplace x.com/o => ../n\n",
			want: []string{"./a", "./b"},
		},
		{name: "no use", src: "go 1.22\n", want: []string{}},
		{name: "syntax error", src: "use (\n", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := WorkUses([]byte(testCase.src))
			if (err != nil) != testCase.wantErr {
				t.Fatalf("WorkUses() error = %v, wantErr %v", err, testCase.wantErr)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("WorkUses() = %q, want %q", got, testCase.want)
			}
		})
	}
}
