package ownedfile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWriteOwned(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		wantOutcome Outcome
		wantBody    string
	}{
		{name: "absent path is written", wantOutcome: Wrote, wantBody: ownedBody},
		{
			name: "owned occupant is replaced", seed: ownedBody + "old", wantOutcome: Wrote,
			wantBody: ownedBody,
		},
		{
			name: "foreign occupant is kept", seed: "user content\n", wantOutcome: Kept,
			wantBody: "user content\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "nested", "rule.mdc")
			if testCase.seed != "" {
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			outcome, err := WriteOwned(path, []byte(ownedBody), 0o600)
			if err != nil {
				t.Fatalf("WriteOwned: %v", err)
			}
			if outcome != testCase.wantOutcome {
				t.Errorf("outcome = %v, want %v", outcome, testCase.wantOutcome)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(got) != testCase.wantBody {
				t.Errorf("body = %q, want %q", got, testCase.wantBody)
			}
		})
	}
}

func TestWriteOwned_Failures(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	readOnly := filepath.Join(root, "readonly")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })
	testCases := []struct {
		name string
		path string
	}{
		{name: "check fails under a file", path: filepath.Join(blocker, "rule.mdc")},
		{name: "mkdir fails in a read-only dir", path: filepath.Join(readOnly, "sub", "rule.mdc")},
		{name: "write fails in a read-only dir", path: filepath.Join(readOnly, "rule.mdc")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if outcome, err := WriteOwned(testCase.path, []byte("x"), 0o600); err == nil ||
				outcome != Kept {
				t.Errorf("WriteOwned = %v, %v, want Kept and an error", outcome, err)
			}
		})
	}
}

func TestRemoveOwned(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		wantOutcome Outcome
		wantExists  bool
	}{
		{name: "absent path is kept", wantOutcome: Kept},
		{
			name: "foreign file is kept", seed: "user content\n", wantOutcome: Kept,
			wantExists: true,
		},
		{name: "owned file is removed", seed: ownedBody, wantOutcome: Removed},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "AGENTS.md")
			if testCase.seed != "" {
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			outcome, err := RemoveOwned(path)
			if err != nil {
				t.Fatalf("RemoveOwned: %v", err)
			}
			if outcome != testCase.wantOutcome {
				t.Errorf("outcome = %v, want %v", outcome, testCase.wantOutcome)
			}
			_, statErr := os.Stat(path)
			if (statErr == nil) != testCase.wantExists {
				t.Errorf("exists = %v, want %v", statErr == nil, testCase.wantExists)
			}
		})
	}
}

func TestRemoveOwned_CheckFails(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if outcome, err := RemoveOwned(filepath.Join(blocker, "AGENTS.md")); err == nil ||
		outcome != Kept {
		t.Errorf("RemoveOwned = %v, %v, want Kept and an error", outcome, err)
	}
}

func TestRemoveOwned_RemoveFails(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte(ownedBody), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if outcome, err := RemoveOwned(path); err == nil || outcome != Kept {
		t.Errorf("RemoveOwned = %v, %v, want Kept and an error", outcome, err)
	}
}

func TestOutcome_ReportWrite(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		outcome      Outcome
		wantNotes    []string
		wantWarnings []string
	}{
		{
			name: "wrote is a note in the caller's words", outcome: Wrote,
			wantNotes: []string{"wrote /x/AGENTS.md"},
		},
		{
			name: "kept is a warning", outcome: Kept,
			wantWarnings: []string{"kept existing user-authored /x/AGENTS.md (not overwritten)"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.outcome.ReportWrite("/x/AGENTS.md", "wrote %s")
			if !slices.Equal(got.Notes, testCase.wantNotes) ||
				!slices.Equal(got.Warnings, testCase.wantWarnings) {
				t.Errorf("report = %+v, want notes %q warnings %q",
					got, testCase.wantNotes, testCase.wantWarnings)
			}
		})
	}
}
