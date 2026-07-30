package target

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRegistryHas(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	testCases := []struct {
		name    string
		target  string
		wantHas bool
	}{
		{"claude registered", "claude", true},
		{"opencode registered", "opencode", true},
		{"unknown not registered", "bogus", false},
		{"empty not registered", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := reg.Has(testCase.target); got != testCase.wantHas {
				t.Fatalf("Has(%q) = %v, want %v", testCase.target, got, testCase.wantHas)
			}
		})
	}
}

func TestRegistryNames(t *testing.T) {
	t.Parallel()
	if got := NewRegistry().Names(); !slices.Equal(got, []string{"claude", "opencode"}) {
		t.Fatalf("Names() = %v, want [claude opencode]", got)
	}
}

func TestRegistryInstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		target   string
		wantErr  bool
		wantFile string
	}{
		{
			"claude dispatch",
			"claude",
			false,
			filepath.Join(".claude", "skills", "go-style", "SKILL.md"),
		},
		{"opencode dispatch", "opencode", false, "opencode.json"},
		{"unknown target errors", "bogus", true, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			ctx := newTestContext(t, base, []string{"go-style"})

			_, err := NewRegistry().Install(testCase.target, ctx)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error for target %q", testCase.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("Install(%q): %v", testCase.target, err)
			}
			if _, statErr := os.Stat(filepath.Join(base, testCase.wantFile)); statErr != nil {
				t.Fatalf(
					"target %q did not write %q: %v",
					testCase.target,
					testCase.wantFile,
					statErr,
				)
			}
		})
	}
}
