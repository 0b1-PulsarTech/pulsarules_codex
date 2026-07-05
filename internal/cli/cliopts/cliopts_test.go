package cliopts

import (
	"path/filepath"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(*testing.T, *Options)
	}{
		{
			name:  "generate default out",
			args:  []string{"generate"},
			check: checkGenerateDefaultOut,
		},
		{
			name:  "list defaults to skills",
			args:  []string{"list"},
			check: checkListDefaultsToSkills,
		},
		{name: "unknown command", args: []string{"frobnicate"}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts, err := ParseArgs(testCase.args)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got opts %+v", opts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.check != nil {
				testCase.check(t, opts)
			}
		})
	}
}

func checkGenerateDefaultOut(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Out != filepath.Join(".", "generated") {
		t.Errorf("Out = %q, want ./generated", opts.Out)
	}
}

func checkListDefaultsToSkills(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Kind != "skills" {
		t.Errorf("Kind = %q, want skills", opts.Kind)
	}
}
