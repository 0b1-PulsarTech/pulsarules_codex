package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

type gateCase struct {
	name       string
	rel        string
	body       string
	skipReason string
}

func gateCases() []gateCase {
	return []gateCase{
		{"go file passes", "a.go", "package a\n", ""},
		{"markdown passes", "a.md", "# title\n", ""},
		{"text file is out of scope", "a.txt", "hi\n", "extension is not .go or .md"},
		{"json is out of scope", "a.json", "{}\n", "extension is not .go or .md"},
		{"fixture under testdata", "testdata/a.go", "package a\n", "under testdata"},
		{"nested fixture", "pkg/testdata/deep/a.md", "# x\n", "under testdata"},
		{"missing file", "gone.go", "", "cannot stat"},
		{"invalid utf-8", "bad.go", "\xff\xfe binary", "not valid UTF-8"},
	}
}

func TestRead_Preconditions(t *testing.T) {
	t.Parallel()

	for _, testCase := range gateCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := filepath.Join(root, testCase.rel)
			if testCase.body != "" {
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(path, []byte(testCase.body), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			report, _, err := New(root).read(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if report.SkipReason != testCase.skipReason {
				t.Errorf("SkipReason = %q, want %q", report.SkipReason, testCase.skipReason)
			}
		})
	}
}

func TestRead_RefusesOutsideRootAndSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "victim.go")
	if err := os.WriteFile(outside, []byte("package a\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	report, _, err := New(root).read(outside)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if report.SkipReason != "outside the project root" {
		t.Errorf("SkipReason = %q, want the out-of-root refusal", report.SkipReason)
	}

	link := filepath.Join(root, "alias.go")
	if linkErr := os.Symlink(outside, link); linkErr != nil {
		t.Skipf("symlinks unavailable: %v", linkErr)
	}
	report, _, err = New(root).read(link)
	if err != nil {
		t.Fatalf("read link: %v", err)
	}
	if report.SkipReason == "" {
		t.Error("a symlink to a file outside the root was accepted")
	}
}

func TestRead_RefusesOversizedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "big.md")
	if err := os.WriteFile(path, make([]byte, fsx.MaxEditedFileBytes+1), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	report, _, err := New(root).read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if report.SkipReason != "larger than the size cap" {
		t.Errorf("SkipReason = %q, want the size refusal", report.SkipReason)
	}
}

func TestRead_RefusesWithoutRoot(t *testing.T) {
	t.Parallel()

	report, _, err := New("").read("a.go")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if report.SkipReason != "no project root" {
		t.Errorf("SkipReason = %q, want the missing-root refusal", report.SkipReason)
	}
}

func TestHandles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want bool
	}{
		{".go", true},
		{".GO", true},
		{".md", true},
		{".dart", false},
		{".mjs", false},
		{".ts", false},
		{"", false},
	}
	for _, testCase := range testCases {
		if got := Handles(testCase.ext); got != testCase.want {
			t.Errorf("Handles(%q) = %v, want %v", testCase.ext, got, testCase.want)
		}
	}
}

func TestIsFixture_RelativeRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	relRoot, err := filepath.Rel(workDir, dir)
	if err != nil {
		t.Skipf("no relative path to the temp dir: %v", err)
	}
	testCases := []struct {
		name string
		rel  string
		want bool
	}{
		{"fixture under testdata", "testdata/a.go", true},
		{"nested fixture", "pkg/testdata/deep/a.go", true},
		{"ordinary file", "pkg/a.go", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(dir, filepath.FromSlash(testCase.rel))
			if got := isFixture(relRoot, path); got != testCase.want {
				t.Errorf("isFixture(%q, %q) = %v, want %v", relRoot, path, got, testCase.want)
			}
		})
	}
}

func TestIsFixture_SymlinkedRoot(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	realRoot := filepath.Join(base, "testdata", "project")
	if err := os.MkdirAll(realRoot, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	linkRoot := filepath.Join(base, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(linkRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	if isFixture(linkRoot, filepath.Join(resolvedRoot, "pkg", "a.go")) {
		t.Error("an ordinary file under a symlinked root is not a fixture")
	}
	if !isFixture(linkRoot, filepath.Join(resolvedRoot, "testdata", "a.go")) {
		t.Error("a testdata file under a symlinked root is a fixture")
	}
}
