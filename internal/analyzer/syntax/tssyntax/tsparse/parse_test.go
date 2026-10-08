package tsparse

import (
	"strings"
	"testing"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"
)

func TestFirstError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		text       string
		kind       core.ScriptKind
		wantPrefix string
	}{
		{name: "valid file", text: "const a = 1;\n", kind: core.ScriptKindTS},
		{
			name:       "broken js file",
			text:       "let a = 1;\nconst = ;\n",
			kind:       core.ScriptKindJS,
			wantPrefix: "line 2: ",
		},
		{
			name: "types in a js file are not a parse error",
			text: "const a: number = 1;\n",
			kind: core.ScriptKindJS,
		},
		{
			name:       "second line error",
			text:       "const a = 1;\nconst = ;\n",
			kind:       core.ScriptKindTS,
			wantPrefix: "line 2: ",
		},
		{
			name:       "jsx in plain ts",
			text:       "const a = <div/>;\n",
			kind:       core.ScriptKindTS,
			wantPrefix: "line 1: ",
		},
		{name: "jsx in tsx", text: "const a = <div/>;\n", kind: core.ScriptKindTSX},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := Parse(testCase.text, testCase.kind)
			got := FirstError(root, testCase.text)
			if testCase.wantPrefix == "" && got != "" {
				t.Fatalf("unexpected error %q", got)
			}
			if !strings.HasPrefix(got, testCase.wantPrefix) {
				t.Fatalf("error %q, want prefix %q", got, testCase.wantPrefix)
			}
		})
	}
}
