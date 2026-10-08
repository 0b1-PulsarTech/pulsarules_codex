package facts

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/tsparse"
)

func collectOf(t *testing.T, text string, options Options) syntax.File {
	t.Helper()

	facts, err := Collect(tsparse.Parse(text, core.ScriptKindTS), text, options)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return facts
}

func TestCollect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		wantErr error
		wantFns int
	}{
		{name: "empty file", text: ""},
		{name: "one function", text: "function f() {}\n", wantFns: 1},
		{
			name:    "chain just under the tree limit",
			text:    "const a = 1" + strings.Repeat("+1", MaxTreeDepth/2) + ";",
			wantFns: 0,
		},
		{
			name:    "chain over the tree limit",
			text:    "const a = 1" + strings.Repeat("+1", MaxTreeDepth+10) + ";",
			wantErr: errTooDeep,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := tsparse.Parse(testCase.text, core.ScriptKindTS)
			facts, err := Collect(root, testCase.text, Options{})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error %v, want %v", err, testCase.wantErr)
			}
			if len(facts.Functions) != testCase.wantFns {
				t.Fatalf("%d functions, want %d", len(facts.Functions), testCase.wantFns)
			}
		})
	}
}

func TestCollectCommonJSExportsAreCountedOnce(t *testing.T) {
	t.Parallel()

	text := "module.exports = { f: function (a) {} };\nexports.g = (b) => { return b; };\n"
	facts, err := Collect(tsparse.Parse(text, core.ScriptKindJS), text, Options{})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	names := make([]string, 0, len(facts.Functions))
	for _, fn := range facts.Functions {
		names = append(names, fn.Name)
	}
	if want := []string{"f", "g"}; !slices.Equal(names, want) {
		t.Fatalf("functions %v, want %v", names, want)
	}
	if len(facts.Names) != 2 {
		t.Fatalf("names %+v, want the two parameters once", facts.Names)
	}
}
