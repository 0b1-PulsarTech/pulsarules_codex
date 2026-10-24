package ownedfile

import (
	"strings"
	"testing"
)

func TestStamp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		body     string
		wantTags int
	}{
		{"marker gets a hash", "# " + Marker + "; do not edit\nbody\n", 1},
		{"only the first marker is stamped", Marker + "\n" + Marker + "\n", 1},
		{"no marker no hash", "plain\n", 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := string(Stamp([]byte(testCase.body)))
			if tags := strings.Count(got, hashTag); tags != testCase.wantTags {
				t.Errorf("hash tags = %d, want %d in %q", tags, testCase.wantTags, got)
			}
			if Edited([]byte(got)) {
				t.Errorf("a freshly stamped body must not read as edited: %q", got)
			}
		})
	}
}

func TestEdited(t *testing.T) {
	t.Parallel()

	stamped := string(Stamp([]byte("# " + Marker + "; do not edit\nbody\n")))
	testCases := []struct {
		name    string
		content string
		want    bool
	}{
		{"untouched", stamped, false},
		{"text appended", stamped + "my notes\n", true},
		{"text inserted before the marker", "intro\n" + stamped, true},
		{"body changed", strings.Replace(stamped, "body", "bodies", 1), true},
		{"hash mangled", strings.Replace(stamped, hashTag, hashTag+"zz", 1), true},
		{"hash truncated", stamped[:strings.Index(stamped, hashTag)+len(hashTag)+10], true},
		{"older install without a hash", "# " + Marker + "; do not edit\nbody\nedited\n", false},
		{"not ours", "my own file\n", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Edited([]byte(testCase.content)); got != testCase.want {
				t.Errorf("Edited = %v, want %v", got, testCase.want)
			}
		})
	}
}
