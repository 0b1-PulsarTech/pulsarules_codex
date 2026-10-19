package helperbin

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"testing/fstest"
)

func testFiles() fstest.MapFS {
	return fstest.MapFS{
		"pubspec.yaml":     {Data: []byte("name: helper\n")},
		"pubspec.lock":     {Data: []byte("packages: {}\n")},
		"bin/summary.dart": {Data: []byte("void main() {}\n")},
	}
}

func TestHelperHash(t *testing.T) {
	t.Parallel()
	first, err := helperHash(testFiles())
	if err != nil {
		t.Fatalf("helperHash: %v", err)
	}
	second, err := helperHash(testFiles())
	if err != nil {
		t.Fatalf("helperHash again: %v", err)
	}
	if first != second {
		t.Errorf("hash not stable: %q vs %q", first, second)
	}
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(first) {
		t.Errorf("hash %q is not 16 hex chars", first)
	}
}

func TestMaterialise(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{name: "fresh dir", target: t.TempDir()},
		{name: "target is a file", target: blocker, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := materialise(testFiles(), testCase.target)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("materialise error = %v, wantErr %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			for _, name := range []string{"pubspec.yaml", "pubspec.lock", "bin/summary.dart"} {
				full := filepath.Join(testCase.target, filepath.FromSlash(name))
				if _, err := os.Stat(full); err != nil {
					t.Errorf("missing %s: %v", name, err)
				}
			}
		})
	}
}
