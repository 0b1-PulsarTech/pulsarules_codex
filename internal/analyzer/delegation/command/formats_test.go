package command

import (
	"slices"
	"testing"
)

func TestFormats(t *testing.T) {
	t.Parallel()

	if got := Formats(); !slices.Equal(got, []string{"dart-analyze-json"}) {
		t.Fatalf("Formats() = %v", got)
	}
}

func TestFormatsReportingExits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		exit      int
		reporting bool
	}{
		{"clean", 0, false},
		{"infos", 1, true},
		{"warnings", 2, true},
		{"errors", 3, true},
		{"usage error", 64, false},
	}
	kind := formats()["dart-analyze-json"]
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := slices.Contains(
				kind.reportingExits,
				testCase.exit,
			); got != testCase.reporting {
				t.Errorf("exit %d reporting = %v, want %v", testCase.exit, got, testCase.reporting)
			}
		})
	}
}
