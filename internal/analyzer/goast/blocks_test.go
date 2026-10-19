package goast

import (
	"go/ast"
	"testing"
)

func TestWalkFileElseVisitor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{
			name:   "plain else",
			source: "func run(ok bool) {\n\tif ok {\n\t} else {\n\t}\n}",
			want:   1,
		},
		{
			name:   "else if only",
			source: "func run(ok bool) {\n\tif ok {\n\t} else if !ok {\n\t}\n}",
		},
		{
			name: "else in a case body",
			source: "func run(ok bool) {\n\tswitch {\n\tcase ok:\n\t\tif ok {\n\t\t} else {\n\t\t}\n" +
				"\t}\n}",
		},
		{name: "bodiless declaration", source: "func run()"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := 0
			WalkFile(parse(t, testCase.source), Visitor{Else: func(*ast.BlockStmt) { got++ }})
			if got != testCase.want {
				t.Fatalf("else visits = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestWalkFileBlockDepth(t *testing.T) {
	t.Parallel()

	source := "func run(ok bool) {\n\tif ok {\n\t\tfor {\n\t\t}\n\t}\n}"
	var depths []int
	WalkFile(parse(t, source), Visitor{Block: func(_ *ast.BlockStmt, depth int) {
		depths = append(depths, depth)
	}})
	if len(depths) != 3 || depths[0] != 0 || depths[1] != 1 || depths[2] != 2 {
		t.Fatalf("depths = %v, want [0 1 2]", depths)
	}
}
