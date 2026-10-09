package dartgen

import "testing"

func TestIsGenerated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		uri  string
		want bool
	}{
		{"g", "a.g.dart", true},
		{"freezed", "pkg/a.freezed.dart", true},
		{"mocks", "a.mocks.dart", true},
		{"gr", "a.gr.dart", true},
		{"config", "a.config.dart", true},
		{"pb", "a.pb.dart", true},
		{"pbenum", "a.pbenum.dart", true},
		{"pbgrpc", "a.pbgrpc.dart", true},
		{"pbjson", "a.pbjson.dart", true},
		{"template", "a.template.dart", true},
		{"plain", "a.dart", false},
		{"suffix inside the name", "a.g.dart.bak", false},
		{"empty", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsGenerated(testCase.uri); got != testCase.want {
				t.Errorf("IsGenerated(%q) = %v, want %v", testCase.uri, got, testCase.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		file string
		uri  string
		want string
	}{
		{"neighbour", "lib/a.dart", "a.g.dart", "lib/a.g.dart"},
		{"up and down", "lib/src/a.dart", "../l10n/c.g.dart", "lib/l10n/c.g.dart"},
		{"root file", "a.dart", "a.g.dart", "a.g.dart"},
		{"escapes the project", "a.dart", "../a.g.dart", ""},
		{"escapes from depth", "lib/a.dart", "../../a.g.dart", ""},
		{"package scheme", "lib/a.dart", "package:x/a.g.dart", ""},
		{"dart scheme", "lib/a.dart", "dart:io", ""},
		{"absolute", "lib/a.dart", "/a.g.dart", ""},
		{"empty", "lib/a.dart", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Resolve(testCase.file, testCase.uri); got != testCase.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q",
					testCase.file, testCase.uri, got, testCase.want)
			}
		})
	}
}
