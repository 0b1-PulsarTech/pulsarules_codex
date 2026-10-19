package facts

import "testing"

func TestSourceNameOr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		inClass bool
		want    string
	}{
		{name: "identifier", text: "function f() {}", want: "f"},
		{
			name:    "string literal key",
			text:    "class A { 'quoted'() {} }",
			inClass: true,
			want:    "quoted",
		},
		{
			name:    "computed key keeps its source",
			text:    "class A { [a + b]() {} }",
			inClass: true,
			want:    "[a + b]",
		},
		{
			name: "missing name takes the fallback",
			text: "export default function () {}",
			want: "fb",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, src := parseOf(t, testCase.text)
			node := root.Statements()[0]
			if testCase.inClass {
				node = node.Members()[0]
			}
			if got := src.nameOr(node.Name(), "fb"); got != testCase.want {
				t.Fatalf("nameOr = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestSourceLineAt(t *testing.T) {
	t.Parallel()

	_, src := parseOf(t, "\n\n  // note\n  let a;\n")
	if got := src.lineAt(0); got != 4 {
		t.Fatalf("lineAt skips leading trivia: line %d, want 4", got)
	}
}
