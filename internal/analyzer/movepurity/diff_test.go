package movepurity

import (
	"testing"
)

type importOnlyDiffCase struct {
	name string
	diff string
	want bool
}

var importOnlyDiffCases = []importOnlyDiffCase{
	{
		name: "no content lines",
		diff: "diff --git c/old.go i/new.go\n" +
			"similarity index 100%\n" +
			"rename from old.go\n" +
			"rename to new.go\n",
		want: true,
	},
	{
		name: "bare import path change",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -3,1 +3,1 @@\n" +
			`-	"repo/old/thing"` + "\n" +
			`+	"repo/new/thing"` + "\n",
		want: true,
	},
	{
		name: "aliased import path change",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -3,1 +3,1 @@\n" +
			`-	foov1 "repo/old/foo"` + "\n" +
			`+	foov1 "repo/new/foo"` + "\n",
		want: true,
	},
	{
		name: "package clause change",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -1,1 +1,1 @@\n" +
			"-package old\n" +
			"+package new\n",
		want: true,
	},
	{
		name: "blank import separator and block punctuation",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -1,3 +1,5 @@\n" +
			"+import (\n" +
			`+	"fmt"` + "\n" +
			"+\n" +
			`+	"os"` + "\n" +
			"+)\n",
		want: true,
	},
	{
		name: "a real statement change is not import-only",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -1,1 +1,2 @@\n" +
			" package a\n" +
			"+var extra = 1\n",
		want: false,
	},
	{
		name: "one import line and one real edit",
		diff: "diff --git c/a.go i/a.go\n" +
			"--- c/a.go\n" +
			"+++ i/a.go\n" +
			"@@ -1,2 +1,2 @@\n" +
			`-	"repo/old/thing"` + "\n" +
			`+	"repo/new/thing"` + "\n" +
			"-func Old() {}\n" +
			"+func New() {}\n",
		want: false,
	},
}

func TestIsImportOnlyDiff(t *testing.T) {
	t.Parallel()

	for _, testCase := range importOnlyDiffCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isImportOnlyDiff(testCase.diff); got != testCase.want {
				t.Fatalf("isImportOnlyDiff() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsImportOnlyLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{"blank", "", true},
		{"open paren", "(", true},
		{"close paren", ")", true},
		{"import block open", "import (", true},
		{"package clause", "package foo", true},
		{"single-line import", `import "fmt"`, true},
		{"bare import path", `"github.com/x/y"`, true},
		{"blank-identifier import", `_ "github.com/x/y"`, true},
		{"aliased import", `foov1 "github.com/x/foov1"`, true},
		{"a statement", "var x = 1", false},
		{"a func signature", "func Foo() {}", false},
		{"two tokens but not an import spec", "not an import spec here", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isImportOnlyLine(testCase.line); got != testCase.want {
				t.Fatalf("isImportOnlyLine(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}

func TestDiffContentLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		line        string
		wantContent string
		wantOK      bool
	}{
		{"added line", "+var x = 1", "var x = 1", true},
		{"removed line", "-var x = 1", "var x = 1", true},
		{"file header old", "--- c/a.go", "", false},
		{"file header new", "+++ i/a.go", "", false},
		{"context line", " package a", "", false},
		{"hunk header", "@@ -1,1 +1,1 @@", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			content, ok := diffContentLine(testCase.line)
			if ok != testCase.wantOK || content != testCase.wantContent {
				t.Fatalf(
					"diffContentLine(%q) = (%q, %v), want (%q, %v)",
					testCase.line, content, ok, testCase.wantContent, testCase.wantOK,
				)
			}
		})
	}
}
