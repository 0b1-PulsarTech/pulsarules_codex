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

func TestPromptLayout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		input        string
		noProfiles   bool
		want         string
		wantPrompted bool
	}{
		{name: "picks a profile", input: "monorepo\n", want: "monorepo", wantPrompted: true},
		{name: "enter skips", input: "\n", want: "", wantPrompted: true},
		{name: "eof skips", input: "", want: "", wantPrompted: true},
		{name: "no profiles never prompts", input: "monorepo\n", noProfiles: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			idx := testServices(t).Index
			if testCase.noProfiles {
				idx.Profiles = nil
			}
			var out strings.Builder
			console := newPrompter(strings.NewReader(testCase.input), &out)
			if got := promptLayout(idx, console); got != testCase.want {
				t.Errorf("layout = %q, want %q", got, testCase.want)
			}
			if prompted := out.Len() > 0; prompted != testCase.wantPrompted {
				t.Fatalf(
					"prompted = %v, want %v (out %q)",
					prompted,
					testCase.wantPrompted,
					out.String(),
				)
			}
			if !testCase.wantPrompted {
				return
			}
			for _, want := range []string{"Available layout profiles", idx.Profiles[0].ID, "layout> "} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output %q lacks %q", out.String(), want)
				}
			}
		})
	}
}
