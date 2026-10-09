package filesize

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestHardFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		errorLines int
		isTest     bool
		want       int
	}{
		{name: "production keeps the cap", errorLines: 300, want: 300},
		{name: "test file gets 2.6x the cap", errorLines: 300, isTest: true, want: 780},
		{name: "no cap stays no cap in tests", errorLines: 0, isTest: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			change := pass.FileChange{Path: "foo.go", IsTest: testCase.isTest}
			got := newLineLimits(180, testCase.errorLines).hardFor(change)
			if got != testCase.want {
				t.Errorf("hard cap = %d, want %d", got, testCase.want)
			}
		})
	}
}
