package install

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestPrintProfiles(t *testing.T) {
	t.Parallel()

	const header = "Available layout profiles (press Enter to skip):\n"
	testCases := []struct {
		name     string
		profiles []knowledge.Profile
		want     string
	}{
		{
			name:     "benign profile is byte-identical",
			profiles: []knowledge.Profile{{ID: "p1", Axis: "arch", Description: "Layered."}},
			want:     header + "  p1 [arch] - Layered.\n",
		},
		{
			name: "forged fields stay on one line",
			profiles: []knowledge.Profile{
				{ID: "a\n[ERROR] i", Axis: "b\n[ERROR] x", Description: "c\n[ERROR] d"},
			},
			want: header + `  a\n[ERROR] i [b\n[ERROR] x] - c\n[ERROR] d` + "\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			printProfiles(&out, testCase.profiles)
			if out.String() != testCase.want {
				t.Errorf("output = %q, want %q", out.String(), testCase.want)
			}
		})
	}
}
