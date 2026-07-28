package emoji

import "testing"

func TestProhibitedReplacement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		shortcode       string
		wantBanned      bool
		wantReplacement string
	}{
		{"robot is banned outright", "robot", true, ""},
		{"test tube points at tea", "test_tube", true, "tea"},
		{"compass is banned outright", "compass", true, ""},
		{"sparkles is banned outright", "sparkles", true, ""},
		{"tea is allowed", "tea", false, ""},
		{"wrench is allowed", "wrench", false, ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			replacement, banned := ProhibitedReplacement(testCase.shortcode)
			if banned != testCase.wantBanned {
				t.Fatalf("banned = %v, want %v", banned, testCase.wantBanned)
			}
			if replacement != testCase.wantReplacement {
				t.Fatalf("replacement = %q, want %q", replacement, testCase.wantReplacement)
			}
			if got := IsProhibited(testCase.shortcode); got != testCase.wantBanned {
				t.Fatalf("IsProhibited = %v, want %v", got, testCase.wantBanned)
			}
		})
	}
}

func TestIsNonRendering(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		shortcode string
		want      bool
	}{
		{"clown face renders as text", "clown_face", true},
		{"adhesive bandage renders as text", "adhesive_bandage", true},
		{"abacus renders as text", "abacus", true},
		{"wrench renders fine", "wrench", false},
		{"unknown shortcode", "not_an_emoji_at_all", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := IsNonRendering(testCase.shortcode); got != testCase.want {
				t.Fatalf("IsNonRendering(%q) = %v, want %v", testCase.shortcode, got, testCase.want)
			}
		})
	}
}
