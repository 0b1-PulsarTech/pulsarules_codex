package flags

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaseDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		global  bool
		project string
		want    string
		wantErr bool
	}{
		{name: "project", project: "/repo", want: "/repo"},
		{name: "relative project is made absolute", project: ".", want: workDir(t)},
		{name: "both exclusive", global: true, project: "/repo", wantErr: true},
		{name: "neither", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base, err := BaseDir(Scope{Global: testCase.global, Project: testCase.project})
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

	testCases := []struct {
		name    string
		home    string
		want    string
		wantErr bool
	}{
		{name: "home from the scope", home: "/home/user", want: "/home/user"},
		{
			name: "relative home is made absolute",
			home: "rel",
			want: filepath.Join(workDir(t), "rel"),
		},
		{name: "unknown home", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base, err := BaseDir(Scope{Global: true, Home: testCase.home})
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", base)
				}
				return
			}
			if err != nil || base != testCase.want {
				t.Errorf("base = %q, %v, want %q", base, err, testCase.want)
			}
		})
	}
}

func TestUserClaudeDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		scope Scope
		want  string
	}{
		{name: "default under home", scope: Scope{Home: "/h"}, want: "/h/.claude"},
		{name: "config dir wins", scope: Scope{Home: "/h", ConfigDir: "/cfg"}, want: "/cfg"},
		{
			name:  "relative config dir is made absolute",
			scope: Scope{ConfigDir: "cfg"},
			want:  filepath.Join(workDir(t), "cfg"),
		},
		{name: "nothing known"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := UserClaudeDir(testCase.scope)
			if err != nil || got != testCase.want {
				t.Errorf("UserClaudeDir = %q, %v, want %q", got, err, testCase.want)
			}
		})
	}
}

func TestScope_WithEnv(t *testing.T) {
	t.Parallel()

	got := Scope{Global: true}.WithEnv(Environment{Home: "/h", ConfigDir: "/c"})
	want := Scope{Global: true, Home: "/h", ConfigDir: "/c"}
	if got != want {
		t.Errorf("WithEnv = %+v, want %+v", got, want)
	}
}

func workDir(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return dir
}
