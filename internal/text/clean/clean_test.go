package clean

import (
	"os"
	"path/filepath"
	"testing"
)

func seed(tb testing.TB, root, rel, body string) string {
	tb.Helper()
	path := filepath.Join(root, rel)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { //nolint:gosec // fixture.
		tb.Fatalf("seed %q: %v", rel, err)
	}
	return path
}

func TestCleanFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		body        string
		want        string
		wantChanged bool
		wantRemains int
	}{
		{"zero width space goes", "a\u200Bb\n", "ab\n", true, 0},
		{"nbsp folds to a space", "a\u00A0b\n", "a b\n", true, 0},
		{"bidi override goes", "x\u202Ey\n", "xy\n", true, 0},
		{"nothing to do", "clean\n", "clean\n", false, 0},
		{"em dash stays and is reported", "a\u2014b\n", "a\u2014b\n", false, 1},
		{
			"emoji glue stays",
			"\U0001F468\u200D\U0001F469\n",
			"\U0001F468\u200D\U0001F469\n",
			false,
			1,
		},
		{"carrier removed, dash reported", "a\u200Bb\u2014c\n", "ab\u2014c\n", true, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "f.md", testCase.body)

			report, err := New(root).CleanFile(path)
			if err != nil {
				t.Fatalf("CleanFile: %v", err)
			}
			if report.Changed != testCase.wantChanged {
				t.Errorf("Changed = %v, want %v", report.Changed, testCase.wantChanged)
			}
			if len(report.Remaining) != testCase.wantRemains {
				t.Errorf("Remaining = %+v, want %d", report.Remaining, testCase.wantRemains)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != testCase.want {
				t.Errorf("file = %q, want %q", body, testCase.want)
			}
		})
	}
}

// TestCleanFile_PreservesMode pins the side effect that would otherwise show up
// as a spurious diff on every edit: the file keeps the permissions it had.
func TestCleanFile_PreservesMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := seed(t, root, "f.go", "package a // a\u200Bb\n")

	if _, err := New(root).CleanFile(path); err != nil {
		t.Fatalf("CleanFile: %v", err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if stat.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", stat.Mode().Perm())
	}
}

func TestCleanFile_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := seed(t, root, "f.md", "a\u200Bb\u00A0c\n")
	cleaner := New(root)

	first, err := cleaner.CleanFile(path)
	if err != nil || !first.Changed {
		t.Fatalf("first pass: changed=%v err=%v", first.Changed, err)
	}
	second, err := cleaner.CleanFile(path)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if second.Changed {
		t.Error("second pass rewrote an already-clean file")
	}
}

// TestInspect_NeverWrites is the guarantee the clean command's default relies on.
func TestInspect_NeverWrites(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const body = "a\u200Bb\u2014c\n"
	path := seed(t, root, "f.md", body)

	report, err := New(root).Inspect(path)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(report.Remaining) != 2 {
		t.Errorf("Remaining = %+v, want 2", report.Remaining)
	}
	if report.Changed {
		t.Error("Inspect reported a change")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != body {
		t.Errorf("Inspect rewrote the file to %q", got)
	}
}
