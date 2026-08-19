package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestKnowledgeDriftNotice covers the tri-state the check turns on: nothing to
// compare stays silent, a real divergence speaks, and a tree that cannot be read
// says THAT rather than passing for up to date.
func TestKnowledgeDriftNotice(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		setup      func(t *testing.T, projectDir string)
		wantFound  bool
		wantPhrase string
	}{
		{
			name:  "no knowledge tree is a legitimate install target",
			setup: func(*testing.T, string) {},
		},
		{
			name: "a knowledge dir without standards stays silent",
			setup: func(t *testing.T, projectDir string) {
				t.Helper()
				mkdirOrFatal(t, filepath.Join(projectDir, "knowledge"))
			},
		},
		{
			name: "standards that differ from the embedded ones warn",
			setup: func(t *testing.T, projectDir string) {
				t.Helper()
				standards := filepath.Join(projectDir, "knowledge", "standards")
				mkdirOrFatal(t, standards)
				if err := os.WriteFile(
					filepath.Join(standards, "skills.yaml"), []byte("skills: []\n"), 0o600,
				); err != nil {
					t.Fatalf("seed standards: %v", err)
				}
			},
			wantFound:  true,
			wantPhrase: "behind the source",
		},
		{
			name: "an unreadable standards tree reports that it could not verify",
			setup: func(t *testing.T, projectDir string) {
				t.Helper()
				standards := filepath.Join(projectDir, "knowledge", "standards")
				mkdirOrFatal(t, standards)
				if err := os.WriteFile(
					filepath.Join(standards, "skills.yaml"), []byte("skills: []\n"), 0o600,
				); err != nil {
					t.Fatalf("seed standards: %v", err)
				}
				if err := os.Chmod(standards, 0o000); err != nil {
					t.Fatalf("chmod: %v", err)
				}
				t.Cleanup(func() { _ = os.Chmod(standards, 0o700) })
			},
			wantFound:  true,
			wantPhrase: "Could not verify",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			testCase.setup(t, projectDir)
			notice, found := knowledgeDriftNotice(projectDir)
			if found != testCase.wantFound {
				t.Fatalf("found = %v, want %v (notice %q)", found, testCase.wantFound, notice)
			}
			if testCase.wantPhrase != "" && !strings.Contains(notice, testCase.wantPhrase) {
				t.Errorf("notice = %q, want it to contain %q", notice, testCase.wantPhrase)
			}
		})
	}
}

// TestKnowledgeDriftNotice_ThisCheckout asserts the check is silent against the
// tree the test binary was built from - the everyday path. A false positive here
// would fire on every session and teach people to ignore the warning.
func TestKnowledgeDriftNotice_ThisCheckout(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(repoRoot, "knowledge", "standards")); err != nil {
		t.Skip("standards tree not reachable from the test's working dir")
	}
	if notice, found := knowledgeDriftNotice(repoRoot); found {
		t.Errorf("reported drift against its own source tree: %q", notice)
	}
}

func mkdirOrFatal(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir %q: %v", dir, err)
	}
}
