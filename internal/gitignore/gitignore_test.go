package gitignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// seedGitignore writes rawSeed directly to dir/.gitignore (skipped when
// empty) and then, if seedEntries is non-empty, runs Ensure over it - so a
// case can seed a hand-authored (unmarked) file, a marked one, or both in
// sequence, without every TestRemove case branching on it inline.
func seedGitignore(tb testing.TB, dir, rawSeed string, seedEntries []string) {
	tb.Helper()
	if rawSeed != "" {
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
	if len(seedEntries) == 0 {
		return
	}
	if err := Ensure(dir, seedEntries...); err != nil {
		tb.Fatalf("Ensure seed: %v", err)
	}
}

// TestEnsure covers creating the file, idempotency when every entry is
// already present, and appending missing entries while preserving existing
// content (with or without a trailing newline).
func TestEnsure(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		hasSeed bool
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
			hasSeed: true,
			seed:    "/bin/\n/hooks/\n",
			entries: []string{"/bin/", "/hooks/"},
			wantHas: []string{"/bin/", "/hooks/"},
		},
		{
			name:    "appended preserving existing lines",
			hasSeed: true,
			seed:    "foo\n",
			entries: []string{"/bin/"},
			wantHas: []string{"foo", "/bin/"},
		},
		{
			name:    "appended when no trailing newline",
			hasSeed: true,
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
			if testCase.hasSeed {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure: %v", err)
			}
			for _, want := range testCase.wantHas {
				if !slices.Contains(splitLines(readFile(t, path)), want) {
					t.Errorf("gitignore %q missing %q", readFile(t, path), want)
				}
			}
			// Running twice must not duplicate an entry.
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure (rerun): %v", err)
			}
			for _, entry := range testCase.entries {
				if n := strings.Count(readFile(t, path), entry); n != 1 {
					t.Errorf(
						"expected exactly one %q entry, got %d in %q",
						entry,
						n,
						readFile(t, path),
					)
				}
			}
		})
	}
}

// removeTestCase is TestRemove's table row. rawSeed, when set, is written
// directly to .gitignore before the test runs, bypassing Ensure - so the
// file carries no markerLine unless seedEntries also runs Ensure over it
// afterward.
type removeTestCase struct {
	name        string
	rawSeed     string
	seedEntries []string
	entries     []string
	wantRemoved []string
	wantAbsent  bool
	wantContent string
}

// TestRemove covers stripping matched entries (and markerLine) while
// preserving the rest, the no-op when nothing matches or the file is
// absent, and deleting the .gitignore once removal leaves it empty. It also
// pins the fix for a hand-authored file (even with identical entries)
// surviving Remove untouched, since there is no marker proving it is ours.
func TestRemove(t *testing.T) {
	t.Parallel()

	testCases := []removeTestCase{
		{
			name:        "removes matched entries and markerLine, preserving the rest",
			rawSeed:     "foo\nbar\n",
			seedEntries: []string{"/bin/", "/hooks/"},
			entries:     []string{"/bin/", "/hooks/"},
			wantRemoved: []string{"/bin/", "/hooks/"},
			wantContent: "foo\nbar\n",
		},
		{
			name:        "no-op when the marked file has none of the requested entries",
			seedEntries: []string{"/keep/"},
			entries:     []string{"/other/"},
			wantContent: markerLine + "\n/keep/\n",
		},
		{
			name:        "no-op on a hand-authored file lacking the marker",
			rawSeed:     "foo\nbar\n",
			entries:     []string{"/bin/"},
			wantContent: "foo\nbar\n",
		},
		{
			name:        "no-op on a hand-authored file that happens to match the entries",
			rawSeed:     "/bin/\n/hooks/\n",
			entries:     []string{"/bin/", "/hooks/"},
			wantContent: "/bin/\n/hooks/\n",
		},
		{
			name:    "no-op when the file does not exist",
			entries: []string{"/bin/"},
			// wantRemoved stays nil; the file never existed to remove from.
			wantAbsent: true,
		},
		{
			name:        "deletes the file once removal empties it",
			seedEntries: []string{"/bin/", "/hooks/"},
			entries:     []string{"/bin/", "/hooks/"},
			wantRemoved: []string{"/bin/", "/hooks/"},
			wantAbsent:  true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "target")
			seedGitignore(t, dir, testCase.rawSeed, testCase.seedEntries)

			removed, err := Remove(dir, testCase.entries...)
			if err != nil {
				t.Fatalf("Remove: %v", err)
			}
			assertRemoveResult(t, dir, removed, testCase)
		})
	}
}

// assertRemoveResult checks the reported removed entries against
// testCase.wantRemoved, then either that dir/.gitignore is absent
// (wantAbsent) or that it survives with wantContent.
func assertRemoveResult(tb testing.TB, dir string, removed []string, testCase removeTestCase) {
	tb.Helper()
	if !slices.Equal(removed, testCase.wantRemoved) {
		tb.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
	}

	path := filepath.Join(dir, ".gitignore")
	_, statErr := os.Stat(path)
	if testCase.wantAbsent {
		if !errors.Is(statErr, fs.ErrNotExist) {
			tb.Errorf("expected %q to be absent, stat err = %v", path, statErr)
		}
		return
	}
	if statErr != nil {
		tb.Fatalf("expected %q to survive, stat err = %v", path, statErr)
	}
	if got := readFile(tb, path); got != testCase.wantContent {
		tb.Errorf("content = %q, want %q", got, testCase.wantContent)
	}
}

// TestRemove_Idempotent asserts running Remove twice is not an error, and the
// second run against an already-deleted file is a genuine no-op.
func TestRemove_Idempotent(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "target")
	if err := Ensure(dir, "/bin/"); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	removed, err := Remove(dir, "/bin/")
	if err != nil {
		t.Fatalf("Remove #1: %v", err)
	}
	if len(removed) == 0 {
		t.Fatalf("Remove #1: expected entries reported removed")
	}
	if removed, err = Remove(dir, "/bin/"); err != nil {
		t.Fatalf("Remove #2: %v", err)
	} else if len(removed) != 0 {
		t.Fatalf("Remove #2: expected no-op, got removed = %v", removed)
	}
}

func readFile(tb testing.TB, path string) string {
	tb.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(raw)
}
