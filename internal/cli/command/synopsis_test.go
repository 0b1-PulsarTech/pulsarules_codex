package command

import (
	"flag"
	"strings"
	"testing"
)

func newFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.Bool("all", false, "everything")
	flagSet.Bool("quiet", false, "less")
	flagSet.String("out", "", "output `FILE`")
	flagSet.String("bare", "", "no metavar")
	return flagSet
}

func TestSynopsis(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		terms []Term
		want  string
	}{
		{"optional bool", []Term{Opt("quiet")}, "[--quiet]"},
		{"optional value", []Term{Opt("out")}, "[--out FILE]"},
		{"optional alternation", []Term{Opt("quiet", "out")}, "[--quiet | --out FILE]"},
		{"required single is bare", []Term{Req("out")}, "--out FILE"},
		{"required alternation", []Term{Req("all", "out")}, "(--all | --out FILE)"},
		{"literal and order", []Term{Lit("<mode>"), Opt("all")}, "<mode> [--all]"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Synopsis(newFlagSet(), testCase.terms)
			if err != nil || got != testCase.want {
				t.Errorf("Synopsis() = %q, %v, want %q", got, err, testCase.want)
			}
		})
	}
}

func TestSynopsis_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		terms []Term
		want  string
	}{
		{"unbound flag", []Term{Opt("missing")}, "does not bind"},
		{"value flag without a metavar", []Term{Opt("bare")}, "no `metavar`"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Synopsis(newFlagSet(), testCase.terms)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("err = %v, want one containing %q", err, testCase.want)
			}
		})
	}
}
