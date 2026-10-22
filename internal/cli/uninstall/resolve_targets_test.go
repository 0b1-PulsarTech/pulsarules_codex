package uninstall

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
)

func TestRun_UnknownTargetRemovesNothing(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, inj, installArgs); err != nil {
		t.Fatalf("install.Run: %v", err)
	}
	skillDoc := filepath.Join(projectDir, ".claude", "skills", "go-style", "SKILL.md")

	testCases := []struct {
		name  string
		flags []string
		bad   string
	}{
		{"unknown name", []string{"vim"}, "vim"},
		{"two names in one value", []string{"opencode,claude"}, "opencode,claude"},
		{"padded name", []string{" opencode "}, " opencode "},
		{"empty value", []string{""}, ""},
		{"blank value", []string{" "}, " "},
		{"lone comma", []string{","}, ","},
		{"trailing comma", []string{"opencode,"}, "opencode,"},
		{"empty repeat", []string{"opencode", ""}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			args := []string{"uninstall", "--project", projectDir}
			for _, value := range testCase.flags {
				args = append(args, "--target", value)
			}
			opts := &Command{}
			fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
			opts.Bind(fs)
			if err := fs.Parse(args[1:]); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			err := opts.Run(inj, nil)
			want := fmt.Sprintf(
				"invalid --target %q (want agents|claude|cursor|opencode)",
				testCase.bad,
			)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to contain %q", err, want)
			}
			if _, statErr := os.Stat(skillDoc); statErr != nil {
				t.Errorf("an invalid --target removed %q: %v", skillDoc, statErr)
			}
		})
	}
}

func TestResolveTargets_GlobalDefaultsToClaude(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "AGENTS.md"), []byte("mine"), 0o600); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}
	got, err := resolveTargets(&Command{global: true}, host.NewRegistry(), home)
	if err != nil || !slices.Equal(got, []string{"claude"}) {
		t.Errorf("resolveTargets = %v, %v, want [claude]", got, err)
	}
}
