package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveModulePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		goModBody  string // empty means: do not write a go.mod at all
		writeGoMod bool
		want       string
		wantErr    error
	}{
		{
			name:       "simple module directive",
			writeGoMod: true,
			goModBody:  "module example.com/target\n\ngo 1.24\n",
			want:       "example.com/target",
		},
		{
			name:       "module directive not on first line",
			writeGoMod: true,
			goModBody:  "// a leading comment\nmodule example.com/other\n",
			want:       "example.com/other",
		},
		{
			name:       "missing go.mod",
			writeGoMod: false,
			wantErr:    os.ErrNotExist,
		},
		{
			name:       "go.mod with no module directive",
			writeGoMod: true,
			goModBody:  "go 1.24\n",
			wantErr:    ErrNoModuleDirective,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			if testCase.writeGoMod {
				if err := os.WriteFile(
					filepath.Join(projectDir, "go.mod"),
					[]byte(testCase.goModBody),
					0o644,
				); err != nil {
					t.Fatal(err)
				}
			}

			got, err := ModulePath(projectDir)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("err = %v, want wrapping %v", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
