package install

import (
	"errors"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/selection"
)

func TestPrintWarning_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	printWarning(&out, "binary", errors.New("copy a\n[ERROR] forged"))
	want := `warning: binary: copy a\n[ERROR] forged` + "\n"
	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestPrintDependencyPulls(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		pulled []selection.DependencyPull
		want   string
	}{
		{name: "none prints nothing", pulled: nil, want: ""},
		{
			name:   "benign pull is byte-identical",
			pulled: []selection.DependencyPull{{Skill: "a", RequiredBy: "b"}},
			want:   "pulled in by dependency: a (required by b)\n",
		},
		{
			name:   "forged names stay on one line",
			pulled: []selection.DependencyPull{{Skill: "a\n[ERROR] x", RequiredBy: "b\n[ERROR] y"}},
			want: `pulled in by dependency: a\n[ERROR] x (required by b\n[ERROR] y)` +
				"\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			printDependencyPulls(&out, testCase.pulled)
			if out.String() != testCase.want {
				t.Errorf("output = %q, want %q", out.String(), testCase.want)
			}
		})
	}
}
