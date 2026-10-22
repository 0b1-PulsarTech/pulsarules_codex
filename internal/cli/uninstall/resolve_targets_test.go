package uninstall

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_UnknownTargetRemovesNothing(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	wired := testWiring(t)
	installArgs := []string{"--project", projectDir, "--skills", "go-style", "--no-mcp"}
	if err := runInstall(t, wired, installArgs); err != nil {
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
			err := opts.Run(wired.targets)
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
