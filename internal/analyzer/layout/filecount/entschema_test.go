package filecount

import (
	"go/parser"
	"go/token"
	"testing"
)

type inspectEntTestCase struct {
	name string
	src  string
	want bool
}

func inspectEntCases() []inspectEntTestCase {
	const head = "package s\n\n"
	return []inspectEntTestCase{
		{
			"embedded ent.Schema",
			head + "import \"entgo.io/ent\"\n\ntype U struct{ ent.Schema }\n",
			true,
		},
		{
			"aliased ent import",
			head + "import e \"entgo.io/ent\"\n\ntype U struct{ e.Schema }\n",
			true,
		},
		{
			"embedded mixin.Schema",
			head + "import \"entgo.io/ent/schema/mixin\"\n\ntype M struct{ mixin.Schema }\n",
			true,
		},
		{
			"named field is not an embed",
			head + "import \"entgo.io/ent\"\n\ntype U struct{ s ent.Schema }\n",
			false,
		},
		{
			"schema of another package is not an embed",
			head + "import \"entgo.io/ent\"\n\ntype U struct{ other.Schema }\n",
			false,
		},
		{
			"SchemaConfig is not an embed",
			head + "import \"entgo.io/ent\"\n\ntype U struct{ ent.SchemaConfig }\n",
			false,
		},
		{
			"non struct type is not an embed",
			head + "import \"entgo.io/ent\"\n\ntype U ent.Schema\n",
			false,
		},
		{
			"field subpackage import only",
			head + "import \"entgo.io/ent/schema/field\"\n",
			false,
		},
		{
			"dialect import is not a schema import",
			head + "import \"entgo.io/ent/dialect/sql\"\n",
			false,
		},
		{"lookalike prefix", head + "import \"entgo.io/entx\"\n", false},
		{"var block is skipped", head + "var x = 1\n", false},
	}
}

func TestInspectEnt(t *testing.T) {
	t.Parallel()

	for _, testCase := range inspectEntCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(
				token.NewFileSet(),
				"s.go",
				testCase.src,
				parser.ParseComments,
			)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := inspectEnt(file); got != testCase.want {
				t.Fatalf("inspectEnt = %v, want %v", got, testCase.want)
			}
		})
	}
}
