package cli

import (
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func TestWriteUsage_MatchesTheGoldenText(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile("testdata/usage_golden.txt")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var got strings.Builder
	if err = writeUsage(&got); err != nil {
		t.Fatalf("writeUsage: %v", err)
	}
	if got.String() != string(want) {
		t.Errorf("usage differs from testdata/usage_golden.txt:\n%s", got.String())
	}
}

func TestUsage_SynopsisNamesEveryFlagOfEveryCommand(t *testing.T) {
	t.Parallel()

	lines, err := synopsisLines()
	if err != nil {
		t.Fatalf("synopsisLines: %v", err)
	}
	global := map[string]bool{"root": true, "log-level": true}
	for index, candidate := range table() {
		t.Run(candidate.name, func(t *testing.T) {
			t.Parallel()

			runner := candidate.build(flags.Environment{})
			flagSet := flag.NewFlagSet(candidate.name, flag.ContinueOnError)
			runner.Bind(flagSet)
			if !strings.HasPrefix(lines[index], candidate.name) {
				t.Fatalf("line %d = %q, want it to start with %q", index, lines[index],
					candidate.name)
			}
			words := strings.FieldsFunc(lines[index], func(char rune) bool {
				return strings.ContainsRune(" []()|", char)
			})
			flagSet.VisitAll(func(found *flag.Flag) {
				if !global[found.Name] && !slices.Contains(words, "--"+found.Name) {
					t.Errorf("synopsis %q does not name --%s", lines[index], found.Name)
				}
			})
		})
	}
}

func TestUsage_ListsEveryBoundCommand(t *testing.T) {
	t.Parallel()

	var got strings.Builder
	if err := writeUsage(&got); err != nil {
		t.Fatalf("writeUsage: %v", err)
	}
	for _, candidate := range table() {
		if !strings.Contains(got.String(), "\n  pulsarules_cli "+candidate.name) {
			t.Errorf("usage has no line for the bound command %q", candidate.name)
		}
	}
}
