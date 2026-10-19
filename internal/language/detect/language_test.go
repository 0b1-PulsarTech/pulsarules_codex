package detect

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestParseLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		raw     string
		want    Language
		wantErr bool
	}{
		{name: "go", raw: "go", want: language.Go},
		{name: "dart", raw: "dart", want: language.Dart},
		{name: "uppercase is normalized", raw: "Go", want: language.Go},
		{name: "surrounding space is trimmed", raw: " rust ", want: language.Rust},
		{name: "unknown language is rejected", raw: "cobol", wantErr: true},
		{name: "empty token is rejected", raw: "", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseLanguage(testCase.raw)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("ParseLanguage(%q) = %q, want error", testCase.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLanguage(%q): unexpected error: %v", testCase.raw, err)
			}
			if got != testCase.want {
				t.Errorf("ParseLanguage(%q) = %q, want %q", testCase.raw, got, testCase.want)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		csv     string
		want    []string
		wantErr bool
	}{
		{name: "single language", csv: "go", want: []string{"go"}},
		{
			name: "multiple languages, deduplicated and sorted",
			csv:  "svelte,typescript,dart,dart",
			want: []string{"dart", "svelte", "typescript"},
		},
		{name: "tolerates spaces", csv: " go , dart ", want: []string{"dart", "go"}},
		{name: "empty input yields an empty set", csv: "", want: nil},
		{name: "unknown token fails the whole parse", csv: "go,bogus", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseNames(testCase.csv)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("ParseCSV(%q) = %v, want error", testCase.csv, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCSV(%q): unexpected error: %v", testCase.csv, err)
			}
			if gotStrings := got.Strings(); !slices.Equal(gotStrings, testCase.want) {
				t.Errorf("ParseCSV(%q) = %v, want %v", testCase.csv, gotStrings, testCase.want)
			}
		})
	}
}
