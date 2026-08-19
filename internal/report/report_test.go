package report

import (
	"slices"
	"testing"
)

func TestReport_NoteAndWarn(t *testing.T) {
	t.Parallel()

	var rpt Report
	rpt.Note("installed: %s", "x")
	rpt.Warn("skipped: %s", "y")

	if !slices.Equal(rpt.Notes, []string{"installed: x"}) {
		t.Fatalf("Notes = %v, want [installed: x]", rpt.Notes)
	}
	if !slices.Equal(rpt.Warnings, []string{"skipped: y"}) {
		t.Fatalf("Warnings = %v, want [skipped: y]", rpt.Warnings)
	}
}

func TestReport_Merge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		base        Report
		sub         Report
		wantNotes   []string
		wantWarning []string
	}{
		{
			name:      "merges into an empty report",
			base:      Report{},
			sub:       Report{Notes: []string{"a"}, Warnings: []string{"b"}},
			wantNotes: []string{"a"}, wantWarning: []string{"b"},
		},
		{
			name:      "appends after existing entries",
			base:      Report{Notes: []string{"a"}, Warnings: []string{"b"}},
			sub:       Report{Notes: []string{"c"}, Warnings: []string{"d"}},
			wantNotes: []string{"a", "c"}, wantWarning: []string{"b", "d"},
		},
		{
			name:      "merging an empty sub changes nothing",
			base:      Report{Notes: []string{"a"}},
			sub:       Report{},
			wantNotes: []string{"a"}, wantWarning: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.base
			got.Merge(testCase.sub)
			if !slices.Equal(got.Notes, testCase.wantNotes) {
				t.Errorf("Notes = %v, want %v", got.Notes, testCase.wantNotes)
			}
			if !slices.Equal(got.Warnings, testCase.wantWarning) {
				t.Errorf("Warnings = %v, want %v", got.Warnings, testCase.wantWarning)
			}
		})
	}
}
