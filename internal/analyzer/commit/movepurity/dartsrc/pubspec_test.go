package dartsrc

import (
	"errors"
	"maps"
	"slices"
	"testing"
)

func TestPackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pubspec  string
		wantName string
	}{
		{"plain", "name: ledger\nversion: 1.0.0\n", "ledger"},
		{"quoted with comment", "version: 1\nname: \"ledger\" # the app\n", "ledger"},
		{"indented key is not the package name", "dependencies:\n  name: other\n", ""},
		{"missing", "version: 1\n", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := PackageName(testCase.pubspec); got != testCase.wantName {
				t.Fatalf("PackageName() = %q, want %q", got, testCase.wantName)
			}
		})
	}
}

func TestBuildRoots(t *testing.T) {
	t.Parallel()

	contents := map[string]string{
		"pubspec.yaml":                  "name: ledger\n",
		"packages/scale/pubspec.yaml":   "name: scale\n",
		"packages/bad/pubspec.yaml":     "version: 1\n",
		"packages/gone/pubspec.yaml":    "",
		"packages/one/pubspec.yaml":     "name: twin\n",
		"packages/two/pubspec.yaml":     "name: twin\n",
		"packages/two/sub/pubspec.yaml": "name: twin\n",
		"lib/ledger.dart":               "",
	}
	read := func(file string) (string, error) {
		if file == "packages/gone/pubspec.yaml" {
			return "", errors.New("unreadable")
		}
		return contents[file], nil
	}
	got := BuildRoots(sortedKeys(contents), read)
	want := Roots{"ledger": ".", "scale": "packages/scale", "twin": ""}
	if !maps.Equal(got, want) {
		t.Fatalf("BuildRoots() = %v, want %v", got, want)
	}
}

func sortedKeys(contents map[string]string) []string {
	return slices.Sorted(maps.Keys(contents))
}
