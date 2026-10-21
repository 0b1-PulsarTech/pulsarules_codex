package finding

import (
	"errors"
	"slices"
	"testing"
)

func TestSeverityParseStringRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    Severity
		wantErr error
	}{
		{name: "error", input: "error", want: SeverityError},
		{name: "warning", input: "warning", want: SeverityWarning},
		{name: "info", input: "info", want: SeverityInfo},
		{name: "empty", input: "", want: SeverityInfo, wantErr: ErrUnknownSeverity},
		{name: "unknown", input: "loud", want: SeverityInfo, wantErr: ErrUnknownSeverity},
		{name: "capitalised", input: "Warning", want: SeverityInfo, wantErr: ErrUnknownSeverity},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseSeverity(testCase.input)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf(
					"ParseSeverity(%q) err = %v, want %v",
					testCase.input,
					err,
					testCase.wantErr,
				)
			}
			if got != testCase.want {
				t.Fatalf("ParseSeverity(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
			if testCase.wantErr == nil && got.String() != testCase.input {
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
