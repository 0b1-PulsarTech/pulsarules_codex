package install

import (
	"testing"
)

func TestValidateGitHooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		hooks   []string
		wantErr bool
	}{
		{name: "empty list is valid", hooks: nil},
		{name: "known hooks are valid", hooks: []string{"commit-msg", "pre-commit", "pre-push"}},
		{name: "unknown hook is rejected", hooks: []string{"bogus"}, wantErr: true},
		{
			name:    "one bad name among good ones is rejected",
			hooks:   []string{"commit-msg", "bogus"},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateGitHooks(testCase.hooks)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error for unknown git hook name")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
