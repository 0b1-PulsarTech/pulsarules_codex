package cursor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

const seededBody = "---\ndescription: seed\nglobs:\nalwaysApply: false\n---\n" +
	"<!-- Installed by pulsarules_cli -->\n\nold body\n"

func seedFor(content string) map[string]string {
	if content == "" {
		return nil
	}
	return map[string]string{"go-style.mdc": content}
}

func TestWriteRule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedContent string
		wantWrote   bool
	}{
		{name: "fresh write", wantWrote: true},
		{name: "refreshes a tool-written file", seedContent: seededBody, wantWrote: true},
		{
			name:        "foreign file is not overwritten",
			seedContent: "# My own rule\nDo not touch.\n",
			wantWrote:   false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, RulesDir, "go-style.mdc")
			seedRules(t, filepath.Dir(path), seedFor(testCase.seedContent))

			newBody := "---\ndescription: new\nglobs:\nalwaysApply: false\n---\n" +
				"<!-- Installed by pulsarules_cli -->\n\nnew body\n"
			wrote, err := WriteRule(projectDir, "go-style", newBody)
			if err != nil {
				t.Fatalf("WriteRule: %v", err)
			}
			if (wrote != ownedfile.Kept) != testCase.wantWrote {
				t.Errorf("wrote = %v, want %v", wrote, testCase.wantWrote)
			}

			want := testCase.seedContent
			if testCase.wantWrote {
				want = newBody
			}
			got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
			if readErr != nil {
				t.Fatalf("read %q: %v", path, readErr)
			}
			if string(got) != want {
				t.Errorf("content = %q, want %q", got, want)
			}
		})
	}
}

func TestWriteRule_CheckErrorUnderAFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".cursor"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	if _, err := WriteRule(projectDir, "go-style", "body"); err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestWriteRule_MkdirErrorWording(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	projectDir := t.TempDir()
	cursorDir := filepath.Join(projectDir, ".cursor")
	if err := os.Mkdir(cursorDir, 0o500); err != nil {
		t.Fatalf("seed read-only .cursor: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(cursorDir, 0o700) })

	_, err := WriteRule(projectDir, "go-style", "body")
	rulesDir := filepath.Join(cursorDir, "rules")
	want := "mkdir: mkdir " + rulesDir + ": permission denied"
	if err == nil || err.Error() != want {
		t.Fatalf("WriteRule err = %v, want %q", err, want)
	}
}

func TestWriteRule_ReadErrorWhenPathIsADirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	path := filepath.Join(projectDir, RulesDir, "go-style.mdc")
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("seed directory in place of go-style.mdc: %v", err)
	}

	if _, err := WriteRule(projectDir, "go-style", "body"); err == nil {
		t.Error("expected an error reading a directory as go-style.mdc, got nil")
	}
}
