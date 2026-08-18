package clean

import (
	"os"
	"path/filepath"
	"testing"
)

type gateCase struct {
	name       string
	rel        string
	body       string
	skipReason string
}

// Every row is a precondition that must refuse before a byte is written. A row
// that stops refusing is a hole in the mutation blast radius.
var gateCases = []gateCase{
	{"go file passes", "a.go", "package a\n", ""},
	{"markdown passes", "a.md", "# title\n", ""},
	{"text file is out of scope", "a.txt", "hi\n", "extension is not .go or .md"},
	{"json is out of scope", "a.json", "{}\n", "extension is not .go or .md"},
	{"fixture under testdata", "testdata/a.go", "package a\n", "under testdata"},
	{"nested fixture", "pkg/testdata/deep/a.md", "# x\n", "under testdata"},
	{"missing file", "gone.go", "", "cannot stat"},
	{"invalid utf-8", "bad.go", "\xff\xfe binary", "not valid UTF-8"},
}

func TestRead_Preconditions(t *testing.T) {
	t.Parallel()

	for _, testCase := range gateCases {
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
	if err := os.WriteFile(path, make([]byte, maxBytes+1), 0o600); err != nil {
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
