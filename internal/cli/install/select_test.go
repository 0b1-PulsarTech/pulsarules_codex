package install

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestParseSkillSelection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		total   int
		want    []int
		wantErr bool
	}{
		{name: "blank selects nothing", input: "", total: 5, want: nil},
		{name: "blank with surrounding newline", input: "\n", total: 5, want: nil},
		{name: "a selects all", input: "a", total: 3, want: []int{1, 2, 3}},
		{name: "A is case-insensitive", input: "A", total: 2, want: []int{1, 2}},
		{name: "single index", input: "4", total: 5, want: []int{4}},
		{name: "comma list", input: "1,4", total: 5, want: []int{1, 4}},
		{name: "range", input: "7-9", total: 9, want: []int{7, 8, 9}},
		{name: "mixed list and range", input: "1,4,7-9", total: 9, want: []int{1, 4, 7, 8, 9}},
		{name: "deduplicates overlapping picks", input: "1,1,1-2", total: 5, want: []int{1, 2}},
		{name: "skips a blank token", input: "1,,2", total: 5, want: []int{1, 2}},
		{name: "tolerates spaces", input: " 1 , 4 , 7 - 9 ", total: 9, want: []int{1, 4, 7, 8, 9}},
		{name: "zero is out of range", input: "0", total: 5, wantErr: true},
		{name: "index above total is out of range", input: "6", total: 5, wantErr: true},
		{name: "inverted range is rejected", input: "9-7", total: 9, wantErr: true},
		{name: "garbage token is rejected", input: "x", total: 5, wantErr: true},
		{name: "garbage range start is rejected", input: "x-3", total: 5, wantErr: true},
		{name: "garbage range end is rejected", input: "1-x", total: 5, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSkillSelection(testCase.input, testCase.total)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPrintSkillCatalog(t *testing.T) {
	t.Parallel()

	const header = "Select skills to install (mandatory skills are always included):\n"
	testCases := []struct {
		name    string
		catalog []knowledge.Skill
		want    string
	}{
		{
			name: "benign skills are byte-identical",
			catalog: []knowledge.Skill{
				{ID: "go-style", Description: "Style. More.", AlwaysLoad: true},
				{ID: "commits", Description: "Commit."},
			},
			want: header +
				"   1. [x] go-style - Style. (mandatory, locked)\n" +
				"   2. [ ] commits - Commit.\n",
		},
		{
			name: "forged id and description stay on one line",
			catalog: []knowledge.Skill{
				{ID: "a\n[ERROR] id", Description: "b\n[ERROR] desc"},
			},
			want: header + `   1. [ ] a\n[ERROR] id - b\n[ERROR] desc` + "\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			printSkillCatalog(&out, testCase.catalog)
			if out.String() != testCase.want {
				t.Errorf("output = %q, want %q", out.String(), testCase.want)
			}
		})
	}
}
