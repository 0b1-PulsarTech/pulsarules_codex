package cliopts

import (
	"os"
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
			name:  "install project with hooks flags",
			args:  []string{"install", "--project", "/tmp/repo", "--no-hooks", "--print-hooks"},
			check: checkInstallProjectWithHooks,
		},
		{
			name:  "install with --no-git-hooks",
			args:  []string{"install", "--project", "/tmp/repo", "--no-git-hooks"},
			check: checkInstallNoGitHooks,
		},
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
		{
			name:  "governance suppresses generated files by default",
			args:  []string{"governance", "--project", "/tmp/repo"},
			check: checkGovernanceDefaults,
		},
		{
			name:  "governance with --include-generated",
			args:  []string{"governance", "--project", "/tmp/repo", "--include-generated"},
			check: checkGovernanceIncludeGenerated,
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

func checkInstallProjectWithHooks(t *testing.T, opts *Options) {
	t.Helper()
	if opts.Command != "install" || opts.Project != "/tmp/repo" {
		t.Errorf("unexpected opts: %+v", opts)
	}
	if !opts.NoHooks || !opts.PrintHooks {
		t.Errorf("hook flags not parsed: %+v", opts)
	}
}

func checkInstallNoGitHooks(t *testing.T, opts *Options) {
	t.Helper()
	if !opts.NoGitHooks {
		t.Error("NoGitHooks should be true")
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

func checkGovernanceDefaults(t *testing.T, opts *Options) {
	t.Helper()
	if opts.IncludeGenerated {
		t.Error("IncludeGenerated = true, want false without the flag")
	}
}

func checkGovernanceIncludeGenerated(t *testing.T, opts *Options) {
	t.Helper()
	if !opts.IncludeGenerated {
		t.Error("IncludeGenerated = false, want true with --include-generated")
	}
}

func TestBaseDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		opts    Options
		want    string
		wantErr bool
	}{
		{name: "project", opts: Options{Project: "/repo"}, want: "/repo"},
		{name: "both exclusive", opts: Options{Global: true, Project: "/repo"}, wantErr: true},
		{name: "neither", opts: Options{}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base, err := testCase.opts.BaseDir()
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", base)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if base != testCase.want {
				t.Errorf("base = %q, want %q", base, testCase.want)
			}
		})
	}
}

func TestBaseDir_Global(t *testing.T) {
	t.Parallel()

	opts := Options{Global: true}
	base, err := opts.BaseDir()
	if err != nil {
		t.Fatalf("BaseDir: %v", err)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatalf("resolve home: %v", homeErr)
	}
	if base != home {
		t.Errorf("base = %q, want %q", base, home)
	}
}
