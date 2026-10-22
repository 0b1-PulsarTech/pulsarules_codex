package list

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestJoinOrDash(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		items []string
		want  string
	}{
		{name: "nil renders a dash", items: nil, want: "-"},
		{name: "empty renders a dash", items: []string{}, want: "-"},
		{name: "single item stands alone", items: []string{"go-style"}, want: "go-style"},
		{
			name:  "several items are comma separated",
			items: []string{"go-style", "commits", "security"},
			want:  "go-style, commits, security",
		},
		{name: "an empty entry is kept, not skipped", items: []string{"a", ""}, want: "a, "},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := joinOrDash(testCase.items); got != testCase.want {
				t.Errorf("joinOrDash(%q) = %q, want %q", testCase.items, got, testCase.want)
			}
		})
	}
}

func TestWriteList(t *testing.T) {
	t.Parallel()

	const forged = "x\n[ERROR] forged"
	const escaped = `x\n[ERROR] forged`
	idx := &knowledge.Index{
		Rules:     []knowledge.Rule{{ID: "r1", Name: "Style", Dependencies: []string{"a"}}},
		Patterns:  []knowledge.Pattern{{ID: forged, Name: forged, Composes: []string{forged}}},
		Workflows: []knowledge.Workflow{{ID: "w1", Name: forged}},
		Skills:    []knowledge.Skill{{ID: forged, Description: forged + ". More.", Order: 2}},
	}
	forgedRules := &knowledge.Index{
		Rules: []knowledge.Rule{{ID: forged, Name: forged, Dependencies: []string{forged}}},
	}
	testCases := []struct {
		name    string
		idx     *knowledge.Index
		kind    string
		want    string
		wantErr bool
	}{
		{
			name: "benign rules are byte-identical",
			idx:  idx,
			kind: "rules",
			want: "ID  NAME   DEPS\nr1  Style  a\n",
		},
		{
			name: "forged rule text stays on one line",
			idx:  forgedRules,
			kind: "rules",
			want: "ID" + strings.Repeat(" ", len(escaped)-2+2) + "NAME" +
				strings.Repeat(" ", len(escaped)-4+2) + "DEPS\n" +
				escaped + "  " + escaped + "  " + escaped + "\n",
		},
		{
			name: "forged pattern text stays on one line",
			idx:  idx,
			kind: "patterns",
			want: "ID" + strings.Repeat(" ", len(escaped)-2+2) + "NAME" +
				strings.Repeat(" ", len(escaped)-4+2) + "COMPOSES\n" +
				escaped + "  " + escaped + "  " + escaped + "\n",
		},
		{
			name: "forged workflow text stays on one line",
			idx:  idx,
			kind: "workflows",
			want: "ID  NAME" + strings.Repeat(" ", len(escaped)-4+2) + "COMPOSES\n" +
				"w1  " + escaped + "  -\n",
		},
		{
			name: "forged skill text stays on one line",
			idx:  idx,
			kind: "skills",
			want: "SKILL" + strings.Repeat(" ", len(escaped)-5+2) +
				"LOAD    ORDER  DESCRIPTION\n" +
				escaped + "  signal  2      " + escaped + ".\n",
		},
		{name: "unknown kind is rejected", idx: idx, kind: "nope", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			err := writeList(&out, testCase.idx, testCase.kind)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if out.String() != testCase.want {
				t.Errorf("output = %q, want %q", out.String(), testCase.want)
			}
		})
	}
}

func TestKindOf(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no argument lists skills", want: "skills"},
		{name: "first argument names the kind", args: []string{"rules"}, want: "rules"},
		{
			name: "extra arguments are ignored",
			args: []string{"patterns", "rules"},
			want: "patterns",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := kindOf(testCase.args); got != testCase.want {
				t.Errorf("kindOf(%q) = %q, want %q", testCase.args, got, testCase.want)
			}
		})
	}
}
