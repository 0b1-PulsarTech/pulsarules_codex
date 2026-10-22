package governance

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    Scope
		wantErr string
	}{
		{name: "commit spells ScopeCommit", input: "commit", want: ScopeCommit},
		{name: "changed spells ScopeChanged", input: "changed", want: ScopeChanged},
		{name: "full spells ScopeFull", input: "full", want: ScopeFull},
		{name: "empty defaults to ScopeFull", input: "", want: ScopeFull},
		{
			name: "misspelt commit is an error", input: "comit", want: ScopeFull,
			wantErr: `"comit" (want full|commit|changed)`,
		},
		{
			name: "case matters", input: "Full", want: ScopeFull,
			wantErr: `"Full" (want full|commit|changed)`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseScope(testCase.input)
			if got != testCase.want {
				t.Errorf("ParseScope(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
			if testCase.wantErr == "" && err != nil {
				t.Fatalf("ParseScope(%q) unexpected error: %v", testCase.input, err)
			}
			if testCase.wantErr != "" && (err == nil || err.Error() != testCase.wantErr) {
				t.Fatalf("ParseScope(%q) err = %v, want %q", testCase.input, err, testCase.wantErr)
			}
		})
	}
}

func TestParseScope_ErrorNamesNoFlag(t *testing.T) {
	t.Parallel()

	_, err := ParseScope("bogus")
	if unknown, ok := errors.AsType[unknownScopeError](err); !ok || unknown.Name != "bogus" {
		t.Fatalf("ParseScope() error = %v, want unknownScopeError for %q", err, "bogus")
	}
	if strings.Contains(err.Error(), "--") {
		t.Fatalf("error %q names a CLI flag, want it flag-agnostic", err)
	}
}

func TestScopeNames(t *testing.T) {
	t.Parallel()

	want := []string{"full", "commit", "changed"}
	if got := scopeNames(); !slices.Equal(got, want) {
		t.Fatalf("scopeNames() = %v, want %v", got, want)
	}
	for _, name := range scopeNames() {
		if _, err := ParseScope(name); err != nil {
			t.Errorf("ParseScope(%q) error = %v, want every listed name to parse", name, err)
		}
	}
}
