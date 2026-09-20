package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

type removeCase struct {
	name         string
	target       string
	files        map[string]string
	docs         []string
	lateFiles    map[string]string
	dropFiles    []string
	runs         int
	wantRemoved  []string
	wantRestored string
	wantOrphaned string
	present      []string
	missing      []string
	content      map[string]string
}

func removeCases() []removeCase {
	return append(removeOwnershipCases(), removeBackupCases()...)
}

func removeOwnershipCases() []removeCase {
	return []removeCase{
		{
			name: "owned docs go and foreign ones stay",
			docs: []string{"go-style", "gopls-navigation"},
			files: map[string]string{
				"my-own-notes/SKILL.md": "mine",
			},
			wantRemoved: []string{"go-style", "gopls-navigation"},
			present:     []string{"my-own-notes/SKILL.md", "."},
			missing:     []string{"go-style", "gopls-navigation"},
		},
		{
			name:        "dest goes once emptied",
			docs:        []string{"go-style"},
			wantRemoved: []string{"go-style"},
			missing:     []string{"."},
		},
		{name: "absent dest is a no-op", target: "never-created"},
		{
			name:        "nested user file survives",
			docs:        []string{"dataviz"},
			files:       map[string]string{"dataviz/references/palette.md": "palette data"},
			wantRemoved: []string{"dataviz"},
			present:     []string{"dataviz/references/palette.md", "dataviz"},
			missing:     []string{"dataviz/SKILL.md", "dataviz/.gitignore"},
		},
		{
			name:    "second run removes nothing",
			docs:    []string{"go-style"},
			files:   map[string]string{"my-own-notes/SKILL.md": "mine"},
			runs:    2,
			present: []string{"my-own-notes/SKILL.md", "."},
		},
	}
}

func removeBackupCases() []removeCase {
	orphanSlot := "security/SKILL.md" + marker.BackupSuffix + ".2"
	return []removeCase{
		{
			name:         "backup is restored",
			docs:         []string{"security"},
			files:        map[string]string{"security/SKILL.md": "# My own security notes\n"},
			wantRemoved:  []string{"security"},
			wantRestored: "security/SKILL.md",
			content:      map[string]string{"security/SKILL.md": "# My own security notes\n"},
		},
		{
			name:        "marker alone is enough",
			docs:        []string{"go-style"},
			dropFiles:   []string{"go-style/.gitignore"},
			wantRemoved: []string{"go-style"},
			missing:     []string{"go-style"},
		},
		{
			name:         "orphaned backup is reported",
			docs:         []string{"security"},
			lateFiles:    map[string]string{orphanSlot: "older\n"},
			wantRemoved:  []string{"security"},
			wantOrphaned: "security/SKILL.md",
		},
	}
}

func seedRemoveCase(t *testing.T, root string, testCase removeCase) {
	t.Helper()
	for rel, body := range testCase.files {
		seedFile(t, filepath.Join(root, filepath.FromSlash(rel)), body)
	}
	for _, id := range testCase.docs {
		mustWriteDoc(t, filepath.Join(root, id), "body\n"+marker.Installed+"\n")
	}
	for rel, body := range testCase.lateFiles {
		seedFile(t, filepath.Join(root, filepath.FromSlash(rel)), body)
	}
	for _, rel := range testCase.dropFiles {
		assertPresent(t, filepath.Join(root, filepath.FromSlash(rel)))
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("remove %s: %v", rel, err)
		}
	}
}

func TestRemoveDocs(t *testing.T) {
	t.Parallel()

	testCases := removeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			seedRemoveCase(t, root, testCase)
			dest := filepath.Join(root, testCase.target)
			var removed, restored, orphaned []string
			for range max(testCase.runs, 1) {
				var err error
				if removed, restored, orphaned, err = RemoveDocs(dest, "SKILL.md"); err != nil {
					t.Fatalf("RemoveDocs: %v", err)
				}
			}
			assertIDs(t, "removed", removed, testCase.wantRemoved...)
			assertRemoveOutcome(t, root, testCase, restored, orphaned)
		})
	}
}

func assertRemoveOutcome(
	t *testing.T, root string, testCase removeCase, restored, orphaned []string,
) {
	t.Helper()
	if testCase.wantRestored != "" {
		doc := filepath.Join(root, filepath.FromSlash(testCase.wantRestored))
		assertIDs(t, "restored", restored, marker.RestoreMessage(doc))
	}
	if testCase.wantOrphaned != "" {
		doc := filepath.Join(root, filepath.FromSlash(testCase.wantOrphaned))
		if len(orphaned) != 1 || !strings.Contains(orphaned[0], doc) {
			t.Errorf("orphaned = %v, want one note naming %q", orphaned, doc)
		}
	}
	for _, rel := range testCase.present {
		assertPresent(t, filepath.Join(root, filepath.FromSlash(rel)))
	}
	for _, rel := range testCase.missing {
		assertMissing(t, filepath.Join(root, filepath.FromSlash(rel)))
	}
	for rel, want := range testCase.content {
		assertContent(t, filepath.Join(root, filepath.FromSlash(rel)), want)
	}
}

func TestRemoveDocs_LeavesUserGitignoreAndSkipsRestore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "security")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(ignorePath, []byte("keep-me\n"), 0o600); err != nil {
		t.Fatalf("seed gitignore: %v", err)
	}
	if _, err := WriteDoc(dir, "SKILL.md", "body\n"+marker.Installed+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	backupPath := ignorePath + marker.BackupSuffix
	rendered, err := os.ReadFile(ignorePath) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read gitignore: %v", err)
	}
	if err = os.WriteFile(ignorePath, append(rendered, "mine/\n"...), 0o600); err != nil {
		t.Fatalf("add user line: %v", err)
	}

	_, restored, _, err := RemoveDocs(root, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(restored) != 0 {
		t.Errorf("restored = %v, want none while a user gitignore stands", restored)
	}
	for path, want := range map[string]string{ignorePath: "mine/\n", backupPath: "keep-me\n"} {
		got, readErr := os.ReadFile(path) //nolint:gosec // test fixture.
		if readErr != nil {
			t.Fatalf("read %q: %v", path, readErr)
		}
		if string(got) != want {
			t.Errorf("%q content = %q, want %q", path, got, want)
		}
	}
}
