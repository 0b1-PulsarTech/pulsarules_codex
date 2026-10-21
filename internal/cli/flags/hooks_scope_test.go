package flags

import (
	"slices"
	"testing"
)

func TestParseHooksScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		raw       string
		wantFiles []string
		wantFile  string
		wantErr   string
	}{
		{
			name:      "empty means both",
			wantFiles: []string{"settings.json", "settings.local.json"},
			wantFile:  "settings.json",
		},
		{
			name:      "project",
			raw:       "project",
			wantFiles: []string{"settings.json"},
			wantFile:  "settings.json",
		},
		{
			name:      "local",
			raw:       "local",
			wantFiles: []string{"settings.local.json"},
			wantFile:  "settings.local.json",
		},
		{
			name:    "bogus is refused",
			raw:     "bogus",
			wantErr: `invalid --hooks-scope "bogus" (want project|local)`,
		},
		{
			name:    "case matters",
			raw:     "Project",
			wantErr: `invalid --hooks-scope "Project" (want project|local)`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scope, err := ParseHooksScope(testCase.raw)
			if testCase.wantErr != "" {
				if err == nil || err.Error() != testCase.wantErr {
					t.Fatalf("err = %v, want %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseHooksScope: %v", err)
			}
			if got := scope.Files(); !slices.Equal(got, testCase.wantFiles) {
				t.Errorf("Files() = %v, want %v", got, testCase.wantFiles)
			}
			if got := scope.File(); got != testCase.wantFile {
				t.Errorf("File() = %q, want %q", got, testCase.wantFile)
			}
		})
	}
}
