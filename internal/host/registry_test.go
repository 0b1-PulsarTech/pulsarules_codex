package host

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
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

func TestRegistryUninstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"claude dispatch", "claude", false},
		{"opencode dispatch", "opencode", false},
		{"unknown target errors", "bogus", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			uctx := UninstallContext{
				Base:             t.TempDir(),
				HookUninstallers: install.NewRegistry(),
				SettingsFiles:    []string{"settings.json"},
			}
			_, err := NewRegistry().Uninstall(testCase.target, uctx)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error for target %q", testCase.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("Uninstall(%q): %v", testCase.target, err)
			}
		})
	}
}

func TestRegistryDetectTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		dirs  []string
		files []string
		want  []string
	}{
		{
			name: "both layouts present",
			dirs: []string{".claude", ".opencode"},
			want: []string{"claude", "opencode"},
		},
		{
			name: "only claude",
			dirs: []string{".claude"},
			want: []string{"claude"},
		},
		{
			name:  "opencode.json without .opencode dir",
			files: []string{"opencode.json"},
			want:  []string{"opencode"},
		},
		{
			name: "untouched project",
			want: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			for _, dir := range testCase.dirs {
				if err := os.MkdirAll(filepath.Join(base, dir), 0o750); err != nil {
					t.Fatalf("mkdir %q: %v", dir, err)
				}
			}
			for _, file := range testCase.files {
				if err := os.WriteFile(
					filepath.Join(base, file), []byte("{}"), 0o600,
				); err != nil {
					t.Fatalf("seed %q: %v", file, err)
				}
			}
			if got := NewRegistry().DetectTargets(base); !slices.Equal(got, testCase.want) {
				t.Errorf("DetectTargets(%q) = %v, want %v", base, got, testCase.want)
			}
		})
	}
}
