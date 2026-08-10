package gitignore

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func seedGitignore(tb testing.TB, dir, rawSeed string) {
	tb.Helper()
	if rawSeed == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, ".gitignore"),
		[]byte(rawSeed),
		0o600,
	); err != nil {
		tb.Fatalf("seed: %v", err)
	}
}

func assertEntriesOnce(t *testing.T, path string, entries []string) {
	t.Helper()

	for _, entry := range entries {
		if count := strings.Count(readFile(t, path), entry); count != 1 {
			content := readFile(t, path)
			t.Errorf("expected exactly one %q entry, got %d in %q", entry, count, content)
		}
	}
}

func TestEnsure_AndRerunNeverDuplicatesAnEntry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seed    string
		entries []string
		wantHas []string
	}{
		{
			name:    "created when absent",
			entries: []string{"/bin/", "/hooks/"},
			wantHas: []string{"/bin/", "/hooks/"},
		},
		{
			name:    "idempotent when all entries present",
			seed:    "/bin/\n/hooks/\n",
			entries: []string{"/bin/", "/hooks/"},
			wantHas: []string{"/bin/", "/hooks/"},
		},
		{
			name:    "appended preserving existing lines",
			seed:    "foo\n",
			entries: []string{"/bin/"},
			wantHas: []string{"foo", "/bin/"},
		},
		{
			name:    "appended when no trailing newline",
			seed:    "foo",
			entries: []string{"/bin/"},
			wantHas: []string{"foo", "/bin/"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "target")
			path := filepath.Join(dir, ".gitignore")
			seedGitignore(t, dir, testCase.seed)
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure: %v", err)
			}
			for _, want := range testCase.wantHas {
				if !slices.Contains(splitLines(readFile(t, path)), want) {
					t.Errorf("gitignore %q missing %q", readFile(t, path), want)
				}
			}
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure (rerun): %v", err)
			}
			assertEntriesOnce(t, path, testCase.entries)
		})
	}
}

func readFile(tb testing.TB, path string) string {
	tb.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}

func TestEnsure_StampsTheMarkerOnce(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "target")
	for _, entry := range []string{"/bin/", "/hooks/"} {
		if err := Ensure(dir, entry); err != nil {
			t.Fatalf("Ensure %q: %v", entry, err)
		}
	}
	want := markerLine + "\n/bin/\n/hooks/\n"
	if got := readFile(t, filepath.Join(dir, ".gitignore")); got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestEnsure_ReportsIOErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prepare func(tb testing.TB, root string) string
		wantErr string
	}{
		{
			name: "dir cannot be created",
			prepare: func(tb testing.TB, root string) string {
				tb.Helper()
				blocker := filepath.Join(root, "blocker")
				if err := os.WriteFile(blocker, nil, 0o600); err != nil {
					tb.Fatalf("seed blocker: %v", err)
				}
				return filepath.Join(blocker, "target")
			},
			wantErr: "mkdir ",
		},
		{
			name: "gitignore cannot be read",
			prepare: func(tb testing.TB, root string) string {
				tb.Helper()
				if err := os.Mkdir(filepath.Join(root, ".gitignore"), 0o750); err != nil {
					tb.Fatalf("seed directory: %v", err)
				}
				return root
			},
			wantErr: "read ",
		},
		{
			name: "gitignore cannot be written",
			prepare: func(tb testing.TB, root string) string {
				tb.Helper()
				dangling := filepath.Join(root, "missing", ".gitignore")
				if err := os.Symlink(dangling, filepath.Join(root, ".gitignore")); err != nil {
					tb.Fatalf("seed symlink: %v", err)
				}
				return root
			},
			wantErr: "write ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := testCase.prepare(t, t.TempDir())
			err := Ensure(dir, "/bin/")
			if err == nil || !strings.HasPrefix(err.Error(), testCase.wantErr) {
				t.Errorf("Ensure err = %v, want prefix %q", err, testCase.wantErr)
			}
		})
	}
}
