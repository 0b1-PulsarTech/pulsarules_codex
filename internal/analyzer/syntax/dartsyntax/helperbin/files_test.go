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

func testSource() source { return source{files: testFiles(), root: "."} }

func TestHelperHash(t *testing.T) {
	t.Parallel()
	first, err := testSource().hash()
	if err != nil {
		t.Fatalf("helperHash: %v", err)
	}
	second, err := testSource().hash()
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
			err := testSource().materialise(testCase.target)
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

func TestHelperHashKey(t *testing.T) {
	t.Parallel()
	renamed := testFiles()
	renamed["bin/main.dart"] = renamed["bin/summary.dart"]
	delete(renamed, "bin/summary.dart")
	edited := testFiles()
	edited["pubspec.yaml"] = &fstest.MapFile{Data: []byte("name: other\n")}
	nested := fstest.MapFS{}
	for name, file := range testFiles() {
		nested["helper/"+name] = file
	}
	testCases := []struct {
		name   string
		source source
		want   string
	}{
		{"root and names are hashed", source{files: nested, root: "helper"}, "f54c7c127101f8ce"},
		{"renamed file same content", source{files: renamed, root: "."}, ""},
		{"edited content", source{files: edited, root: "."}, ""},
	}
	plain, err := testSource().hash()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, hashErr := testCase.source.hash()
			if hashErr != nil {
				t.Fatalf("hash: %v", hashErr)
			}
			if testCase.want != "" && got != testCase.want {
				t.Errorf("hash = %q, want the key an older binary computed, %q", got, testCase.want)
			}
			if testCase.want == "" && got == plain {
				t.Errorf("hash %q did not change", got)
			}
		})
	}
}

func TestSourceMissingRoot(t *testing.T) {
	t.Parallel()
	_, err := source{files: testFiles(), root: "/"}.hash()
	assertErrContains(t, err, "open embedded helper")
}
