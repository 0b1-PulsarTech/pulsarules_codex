package names

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func kindsOf(names []syntax.Name) []syntax.NameKind {
	kinds := make([]syntax.NameKind, 0, len(names))
	for _, name := range names {
		kinds = append(kinds, name.Kind)
	}
	return kinds
}

func TestDeclarationKinds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want []syntax.NameKind
	}{
		{
			name: "method receiver and result",
			body: "package pkg\n\nfunc (box Box) Get() (out int) { return }\n",
			want: []syntax.NameKind{
				syntax.NamePackage, syntax.NameMethod, syntax.NameReceiver, syntax.NameResult,
			},
		},
		{
			name: "receiver type parameters",
			body: "package pkg\n\nfunc (box *Box[Item]) Get() {}\n" +
				"func (pair (Pair[Key, Val])) Swap() {}\n",
			want: []syntax.NameKind{
				syntax.NamePackage, syntax.NameMethod, syntax.NameReceiver, syntax.NameTypeParam,
				syntax.NameMethod, syntax.NameReceiver, syntax.NameTypeParam, syntax.NameTypeParam,
			},
		},
		{
			name: "types fields and interface methods",
			body: "package pkg\n\ntype Box[Item any] struct{ size int }\n\n" +
				"type Port interface{ Run() }\n",
			want: []syntax.NameKind{
				syntax.NamePackage, syntax.NameType, syntax.NameTypeParam, syntax.NameField,
				syntax.NameType, syntax.NameMethod,
			},
		},
		{
			name: "constants variables and import alias",
			body: "package pkg\n\nimport al \"fmt\"\n\nconst limit = 1\n\nvar count int\n",
			want: []syntax.NameKind{
				syntax.NamePackage, syntax.NameImportAlias, syntax.NameConstant,
				syntax.NameVariable,
			},
		},
		{
			name: "statements",
			body: "package pkg\n\nfunc run(val any, rows []int) {\n\tcur := 1\n\tcur = 2\n" +
				"\tfor idx, row := range rows {\n\t}\n\tfor pos := 0; pos < 1; pos++ {\n\t}\n" +
				"\tswitch bound := val.(type) {\n\t}\n\tfor idx = range rows {\n\t}\nstop:\n}\n",
			want: []syntax.NameKind{
				syntax.NamePackage, syntax.NameFunction, syntax.NameParameter,
				syntax.NameParameter, syntax.NameVariable, syntax.NameLoopVariable,
				syntax.NameLoopVariable, syntax.NameLoopVariable, syntax.NameSwitchBinding,
				syntax.NameLabel,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := kindsOf(collectNames(t, testCase.body))
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("kinds = %v, want %v", got, testCase.want)
			}
		})
	}
}
