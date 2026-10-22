package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func TestRun_UnknownTargetRemovesNothing(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	installOpts := &version.Options{
		Command: "install", Project: projectDir, Skills: []string{"go-style"}, NoMCP: true,
	}
	if err := install.Run(inj, installOpts); err != nil {
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
			opts, err := version.ParseArgs(args, flags.Environment{})
			if err != nil {
				t.Fatalf("ParseArgs: %v", err)
			}
			err = Run(inj, opts)
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
