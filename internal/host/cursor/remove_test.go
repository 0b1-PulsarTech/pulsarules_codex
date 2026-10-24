package cursor

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func seedRules(tb testing.TB, dir string, seed map[string]string) {
	tb.Helper()

	if seed == nil {
		return
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	for name, content := range seed {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			tb.Fatalf("seed %q: %v", name, err)
		}
	}
}

func assertSurvivors(tb testing.TB, dir string, survive []string) {
	tb.Helper()

	for _, name := range survive {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			tb.Errorf("expected %q to survive, stat err = %v", name, err)
		}
	}
}

func assertDirGone(tb testing.TB, dir string) {
	tb.Helper()

	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("expected %q removed once emptied, stat err = %v", dir, err)
	}
}

func TestRemoveRules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        map[string]string
		wantRemoved []string
		wantSurvive []string
	}{
		{
			name: "removes only tool-written files",
			seed: map[string]string{
				"go-style.mdc": seededBody,
				"foreign.mdc":  "# my own rule\n",
			},
			wantRemoved: []string{"go-style"},
			wantSurvive: []string{"foreign.mdc"},
		},
		{
			name: "skips files that are not rules",
			seed: map[string]string{
				"go-style.mdc": seededBody,
				"notes.txt":    seededBody,
			},
			wantRemoved: []string{"go-style"},
			wantSurvive: []string{"notes.txt"},
		},
		{
			name:        "removes the directory once emptied",
			seed:        map[string]string{"go-style.mdc": seededBody},
			wantRemoved: []string{"go-style"},
		},
		{
			name: "missing directory is a no-op",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			dir := filepath.Join(projectDir, RulesDir)
			seedRules(t, dir, testCase.seed)

			removed, _, err := RemoveRules(projectDir)
			if err != nil {
				t.Fatalf("RemoveRules: %v", err)
			}
			slices.Sort(removed)
			slices.Sort(testCase.wantRemoved)
			if !slices.Equal(removed, testCase.wantRemoved) {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			assertSurvivors(t, dir, testCase.wantSurvive)
			if len(testCase.wantSurvive) == 0 && testCase.seed != nil {
				assertDirGone(t, dir)
			}
		})
	}
}

func TestRemoveRules_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, RulesDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go-style.mdc"),
		[]byte(seededBody),
		0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, _, err := RemoveRules(projectDir)
	if err != nil || len(first) != 1 {
		t.Fatalf("RemoveRules #1: removed=%v err=%v", first, err)
	}
	second, _, err := RemoveRules(projectDir)
	if err != nil || len(second) != 0 {
		t.Fatalf("RemoveRules #2: removed=%v err=%v", second, err)
	}
}

func TestRemoveRules_VanishedSymlink(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, RulesDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ghost := filepath.Join(dir, "ghost.mdc")
	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), ghost); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go-style.mdc"),
		[]byte(seededBody),
		0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	removed, _, err := RemoveRules(projectDir)
	if err != nil {
		t.Fatalf("RemoveRules: %v", err)
	}
	if !slices.Contains(removed, "go-style") {
		t.Errorf("expected go-style removed, got %v", removed)
	}
	if slices.Contains(removed, "ghost") {
		t.Errorf("expected the vanished ghost file not reported as removed, got %v", removed)
	}
}

func TestRemoveRules_ReadDirErrorWhenParentIsAFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".cursor"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	if _, _, err := RemoveRules(projectDir); err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestRemoveRules_KeepsAnEditedRuleInABackup(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	const body = "---\ndescription: d\n---\n<!-- Installed by pulsarules_cli -->\n\nbody\n"
	if _, _, err := WriteRule(projectDir, "go-style", body); err != nil {
		t.Fatalf("WriteRule: %v", err)
	}
	path := filepath.Join(projectDir, RulesDir, "go-style.mdc")
	edited, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatal(err)
	}
	edited = append(edited, "my addition\n"...)
	if err = os.WriteFile(path, edited, 0o600); err != nil {
		t.Fatal(err)
	}
	removed, ledger, err := RemoveRules(projectDir)
	if err != nil || !slices.Equal(removed, []string{"go-style"}) || len(ledger.BackedUp) != 1 {
		t.Fatalf(
			"RemoveRules = %v, %v, %v, want one removed rule and one backup",
			removed,
			ledger,
			err,
		)
	}
	if got, readErr := os.ReadFile(
		path + ".pulsarules-backup",
	); readErr != nil ||
		string(got) != string(edited) { //nolint:gosec // temp dir.
		t.Errorf("backup = %q, %v, want the edited rule", got, readErr)
	}
}
