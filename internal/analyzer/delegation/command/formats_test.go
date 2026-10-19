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
	for name, kind := range formats() {
		if kind.decode == nil || kind.precheck == nil || kind.codegen == nil {
			t.Errorf("format %q leaves a func nil: %+v", name, kind)
		}
		if len(kind.reportingExits) == 0 {
			t.Errorf("format %q has no reporting exits", name)
		}
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
