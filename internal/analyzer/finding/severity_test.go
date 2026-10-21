package finding

import (
	"slices"
	"testing"
)

func TestSeverityParseStringRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		want      Severity
		wantKnown bool
	}{
		{name: "error", input: "error", want: SeverityError, wantKnown: true},
		{name: "warning", input: "warning", want: SeverityWarning, wantKnown: true},
		{name: "info", input: "info", want: SeverityInfo, wantKnown: true},
		{name: "empty", input: "", want: SeverityInfo},
		{name: "unknown", input: "loud", want: SeverityInfo},
		{name: "capitalised", input: "Warning", want: SeverityInfo},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, known := ParseSeverity(testCase.input)
			if known != testCase.wantKnown {
				t.Fatalf("ParseSeverity(%q) known = %v, want %v",
					testCase.input, known, testCase.wantKnown)
			}
			if got != testCase.want {
				t.Fatalf("ParseSeverity(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
			if testCase.wantKnown && got.String() != testCase.input {
				t.Fatalf("String() = %q, want %q", got.String(), testCase.input)
			}
		})
	}
}

func TestSeverityNames(t *testing.T) {
	t.Parallel()

	want := []string{"error", "info", "warning"}
	if got := SeverityNames(); !slices.Equal(got, want) {
		t.Fatalf("SeverityNames() = %v, want %v", got, want)
	}
}

func TestValidateSeverityName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
	}{
		{name: "empty keeps the analyzer default"},
		{name: "error", severity: "error"},
		{name: "warning", severity: "warning"},
		{name: "info", severity: "info"},
		{name: "a typo is rejected", severity: "fatal", wantErr: true},
		{name: "capitalised is rejected", severity: "Warning", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSeverityName(testCase.severity)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ValidateSeverityName(%q) err = %v, wantErr %v",
					testCase.severity, err, testCase.wantErr)
			}
		})
	}
}
