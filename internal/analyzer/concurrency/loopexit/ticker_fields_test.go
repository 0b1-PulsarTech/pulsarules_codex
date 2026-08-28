package loopexit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCollectTickerFieldNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "a *time.Ticker field is collected",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type poller struct{ ticker *time.Ticker }\n",
			want: "ticker",
		},
		{
			name: "a time.Timer field is collected",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type poller struct{ timer time.Timer }\n",
			want: "timer",
		},
		{
			name: "a plain chan field is not collected",
			source: "package foo\n" +
				"type Watcher struct{ C chan int }\n",
			want: "",
		},
		{
			name: "a struct with no fields at all is not collected",
			source: "package foo\n" +
				"type empty struct{}\n",
			want: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := parseFile(t, testCase.source)
			fieldNames := collectTickerFieldNames(file)
			if testCase.want != "" && !fieldNames[testCase.want] {
				t.Fatalf("fieldNames = %v, want %q tracked", fieldNames, testCase.want)
			}
			if testCase.want == "" && len(fieldNames) != 0 {
				t.Fatalf("fieldNames = %v, want none tracked", fieldNames)
			}
		})
	}
}

func parseFile(t *testing.T, source string) *ast.File {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
