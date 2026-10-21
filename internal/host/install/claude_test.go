package install

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func testTemplates(tb testing.TB) fs.FS {
	tb.Helper()
	_, templates, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("load knowledge: %v", err)
	}
	return templates
}

func TestClaude_InstallThenUninstallNotes(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	settings := filepath.Join(claudeDir, "settings.local.json")
	installed, err := Claude(claudeDir, testTemplates(t), "settings.local.json")
	if err != nil {
		t.Fatalf("Claude: %v", err)
	}
	if want := "wired hook into " + settings; !slices.Contains(installed.Notes, want) {
		t.Errorf("install Notes = %v, want %q", installed.Notes, want)
	}

	removed, err := UninstallClaude(claudeDir, "settings.local.json")
	if err != nil {
		t.Fatalf("UninstallClaude: %v", err)
	}
	if want := "removed hook wiring from " + settings; !slices.Contains(removed.Notes, want) {
		t.Errorf("uninstall Notes = %v, want %q", removed.Notes, want)
	}
	if _, statErr := os.Stat(settings); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("an emptied settings file must go, stat err = %v", statErr)
	}
}

func TestClaude_InstallFailureWording(t *testing.T) {
	t.Parallel()

	const objectCause = "json: cannot unmarshal array into Go value of type " +
		"map[string]jsontext.Value"
	const hooksCause = "json: cannot unmarshal array into Go value of type "
	testCases := []struct {
		name         string
		settingsFile string
		seed         string
		want         string
		prefixOnly   bool
	}{
		{
			name: "settings not an object", settingsFile: "settings.json", seed: `[1]`,
			want: "install claude: wire claude settings: merge settings: " +
				"parse existing settings: " + objectCause,
		},
		{
			name: "local settings not an object", settingsFile: "settings.local.json",
			seed: `[1]`,
			want: "install claude: wire claude settings: merge settings: " +
				"parse existing settings: " + objectCause,
		},
		{
			name: "hooks of the wrong type", settingsFile: "settings.json",
			seed: `{"hooks":[]}`,
			want: "install claude: wire claude settings: merge settings: " +
				"parse existing hooks: " + hooksCause,
			prefixOnly: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			claudeDir := t.TempDir()
			path := filepath.Join(claudeDir, testCase.settingsFile)
			if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			_, err := Claude(claudeDir, testTemplates(t), testCase.settingsFile)
			if !errors.Is(err, jsondoc.ErrUnparseable) {
				t.Fatalf("Claude err = %v, want ErrUnparseable", err)
			}
			got := err.Error()
			if testCase.prefixOnly {
				got = got[:min(len(got), len(testCase.want))]
			}
			if got != testCase.want {
				t.Fatalf("Claude err = %v, want text %q", err, testCase.want)
			}
		})
	}
}

func TestClaude_UninstallWarningWording(t *testing.T) {
	t.Parallel()

	for _, settingsFile := range []string{"settings.json", "settings.local.json"} {
		for _, blank := range []string{"", " \n"} {
			t.Run(settingsFile+strconv.Quote(blank), func(t *testing.T) {
				t.Parallel()

				claudeDir := t.TempDir()
				path := filepath.Join(claudeDir, settingsFile)
				if err := os.WriteFile(path, []byte(blank), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
				_, err := UninstallClaude(claudeDir, settingsFile)
				want := "uninstall claude: unwire claude settings: existing config file is not " +
					"valid JSON: " + strconv.Quote(path) + ": unexpected end of JSON input"
				if !errors.Is(err, jsondoc.ErrUnparseable) || err.Error() != want {
					t.Fatalf("UninstallClaude err = %v, want %q", err, want)
				}
			})
		}
	}
}
