package governance

import "testing"

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
			wantErr: `invalid --scope "comit" (want full|commit|changed)`,
		},
		{
			name: "case matters", input: "Full", want: ScopeFull,
			wantErr: `invalid --scope "Full" (want full|commit|changed)`,
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
