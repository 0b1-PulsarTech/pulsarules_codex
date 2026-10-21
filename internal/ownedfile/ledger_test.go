package ownedfile

import (
	"slices"
	"testing"
)

func TestLedger_Report(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		ledger       Ledger
		wantNotes    []string
		wantWarnings []string
	}{
		{name: "empty ledger reports nothing"},
		{
			name:         "backups warn",
			ledger:       Ledger{BackedUp: []string{"backed up a"}},
			wantWarnings: []string{"backed up a"},
		},
		{
			name:         "orphans warn",
			ledger:       Ledger{Orphaned: []string{"orphan a"}},
			wantWarnings: []string{"orphan a"},
		},
		{
			name:      "restores are notes",
			ledger:    Ledger{Restored: []string{"restored a", "restored b"}},
			wantNotes: []string{"restored a", "restored b"},
		},
		{
			name: "every list keeps its own channel and order",
			ledger: Ledger{
				BackedUp: []string{"b1", "b2"},
				Restored: []string{"r1"},
				Orphaned: []string{"o1"},
			},
			wantNotes:    []string{"r1"},
			wantWarnings: []string{"b1", "b2", "o1"},
		},
		{
			name:         "a percent sign is carried verbatim",
			ledger:       Ledger{BackedUp: []string{"100% kept"}},
			wantWarnings: []string{"100% kept"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.ledger.Report()
			if !slices.Equal(got.Notes, testCase.wantNotes) {
				t.Errorf("Notes = %q, want %q", got.Notes, testCase.wantNotes)
			}
			if !slices.Equal(got.Warnings, testCase.wantWarnings) {
				t.Errorf("Warnings = %q, want %q", got.Warnings, testCase.wantWarnings)
			}
		})
	}
}

func TestLedger_AddIgnoresEmptyNotes(t *testing.T) {
	t.Parallel()

	var ledger Ledger
	ledger.AddBackup("")
	ledger.AddRestore("")
	ledger.AddBackup("kept")
	ledger.AddRestore("restored")
	if !slices.Equal(ledger.BackedUp, []string{"kept"}) ||
		!slices.Equal(ledger.Restored, []string{"restored"}) {
		t.Errorf("ledger = %+v, want only the non-empty notes", ledger)
	}
}
