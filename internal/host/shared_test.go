package host

import (
	"errors"
	"fmt"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func TestWarnUnparseable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		err         error
		wantHandled bool
	}{
		{
			name:        "unparseable config is a warning",
			err:         fmt.Errorf("strip hooks: %w", jsondoc.ErrUnparseable),
			wantHandled: true,
		},
		{name: "any other error is not handled", err: errors.New("disk full")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var report Report
			if got := warnUnparseable(&report, testCase.err); got != testCase.wantHandled {
				t.Fatalf("handled = %v, want %v", got, testCase.wantHandled)
			}
			wantWarnings := 0
			if testCase.wantHandled {
				wantWarnings = 1
			}
			if len(report.Warnings) != wantWarnings {
				t.Errorf("Warnings = %q, want %d", report.Warnings, wantWarnings)
			}
		})
	}
}
