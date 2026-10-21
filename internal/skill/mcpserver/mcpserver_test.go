package mcpserver

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestManagedNames(t *testing.T) {
	t.Parallel()

	got := ManagedNames()
	want := []string{"dart", "gopls"}
	if !slices.Equal(got, want) {
		t.Errorf("ManagedNames = %v, want %v", got, want)
	}
}

func fakeOnPath(tb testing.TB, names ...string) {
	tb.Helper()
	dir := tb.TempDir()
	for _, name := range names {
		if err := os.WriteFile(
			filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o700,
		); err != nil {
			tb.Fatalf("write fake %q on PATH: %v", name, err)
		}
	}
	tb.Setenv("PATH", dir)
}

func serverNames(servers []Server) []string {
	names := make([]string, 0, len(servers))
	for _, srv := range servers {
		names = append(names, srv.Name)
	}
	return names
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestAvailable(t *testing.T) {
	testCases := []struct {
		name        string
		onPath      []string
		languages   []language.ID
		wantWired   []string
		wantSkipped []string
	}{
		{
			name: "go with gopls", onPath: []string{"gopls"},
			languages: []language.ID{language.Go}, wantWired: []string{"gopls"},
		},
		{
			name: "dart missing is skipped", onPath: []string{"gopls"},
			languages: []language.ID{language.Go, language.Dart},
			wantWired: []string{"gopls"}, wantSkipped: []string{"dart"},
		},
		{
			name: "a language without a managed server", onPath: []string{"gopls", "dart"},
			languages: []language.ID{language.Rust},
		},
		{
			name: "both in the order given", onPath: []string{"gopls", "dart"},
			languages: []language.ID{language.Dart, language.Go},
			wantWired: []string{"dart", "gopls"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeOnPath(t, testCase.onPath...)
			wired, skipped := Available(testCase.languages)
			if got := serverNames(wired); !slices.Equal(got, testCase.wantWired) {
				t.Errorf("wired = %v, want %v", got, testCase.wantWired)
			}
			if !slices.Equal(skipped, testCase.wantSkipped) {
				t.Errorf("skipped = %v, want %v", skipped, testCase.wantSkipped)
			}
		})
	}
}

func TestIsManaged(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		server  string
		command string
		want    bool
	}{
		{name: "gopls running gopls", server: "gopls", command: "gopls", want: true},
		{name: "dart running dart", server: "dart", command: "dart", want: true},
		{name: "our name, another command", server: "gopls", command: "my-gopls"},
		{name: "another name, our command", server: "mine", command: "gopls"},
		{name: "empty", server: "", command: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsManaged(testCase.server, testCase.command); got != testCase.want {
				t.Errorf("IsManaged(%q, %q) = %v, want %v",
					testCase.server, testCase.command, got, testCase.want)
			}
		})
	}
}
