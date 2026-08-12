package cursorwire

import (
	"os"
	"path/filepath"
	"testing"
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
			if wrote != testCase.wantWrote {
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

func TestWriteRule_MkdirError(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".cursor"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	if _, err := WriteRule(projectDir, "go-style", "body"); err == nil {
		t.Error("expected an error, got nil")
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
