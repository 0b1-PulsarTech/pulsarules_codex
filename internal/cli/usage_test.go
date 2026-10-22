package cli

import (
	"errors"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func wantUsage(t *testing.T) string {
	t.Helper()

	head, err := os.ReadFile("testdata/usage_golden.txt")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	return string(head) + "\n" + usageProse
}

func TestUsage_MatchesTheGoldenHeaderAndSynopsis(t *testing.T) {
	t.Parallel()

	var got strings.Builder
	if err := Usage(&got); err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if want := wantUsage(t); got.String() != want {
		t.Errorf("usage differs from testdata/usage_golden.txt plus usage.txt:\n%s", got.String())
	}
}

func TestUsage_WriteFailureIsReported(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("disk full")
	err := Usage(failingWriter{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Errorf("Usage() = %v, want it to wrap %v", err, wantErr)
	}
}

func TestUsage_SynopsisNamesEveryFlagOfEveryCommand(t *testing.T) {
	t.Parallel()

	lines, err := synopsisLines()
	if err != nil {
		t.Fatalf("synopsisLines: %v", err)
	}
	for index, candidate := range table() {
		t.Run(candidate.name, func(t *testing.T) {
			t.Parallel()

			flagSet := flag.NewFlagSet(candidate.name, flag.ContinueOnError)
			candidate.build(flags.Environment{}).Bind(flagSet)
			if !strings.HasPrefix(lines[index], candidate.name) {
				t.Fatalf("line %d = %q, want it to start with %q", index, lines[index],
					candidate.name)
			}
			words := strings.FieldsFunc(lines[index], func(char rune) bool {
				return strings.ContainsRune(" []()|", char)
			})
			flagSet.VisitAll(func(found *flag.Flag) {
				if !slices.Contains(words, "--"+found.Name) {
					t.Errorf("synopsis %q does not name --%s", lines[index], found.Name)
				}
			})
		})
	}
}
