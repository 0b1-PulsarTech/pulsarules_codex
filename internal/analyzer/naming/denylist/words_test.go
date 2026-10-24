package denylist

import (
	"reflect"
	"testing"
)

func TestSplitWords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  []string
	}{
		{"acronym end", "HTTPServerError", []string{"http", "server", "error"}},
		{"snake case", "load_alerts", []string{"load", "alerts"}},
		{"dots and dashes", "my-dir.v2", []string{"my", "dir", "v2"}},
		{"digit then upper", "abc2Def", []string{"abc2", "def"}},
		{"sigil file name", "+page.server.dart", []string{"page", "server", "dart"}},
		{"camel case", "historyWithoutReload", []string{"history", "without", "reload"}},
		{"single word", "draft", []string{"draft"}},
		{"lower then acronym", "userID", []string{"user", "id"}},
		{"screaming snake", "MAX_DRAFT_COUNT", []string{"max", "draft", "count"}},
		{"lower prefix then upper", "iOS", []string{"i", "os"}},
		{"leading underscore", "_draft", []string{"draft"}},
		{"leading dollar", "$draft", []string{"draft"}},
		{"plural acronym", "URLs", []string{"ur", "ls"}},
		{"decomposed accent stays whole", "cafe\u0301", []string{"cafe\u0301"}},
		{"decomposed accent then camel", "cafe\u0301Draft", []string{"cafe\u0301", "draft"}},
		{"only separators", "__--", nil},
		{"empty", "", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := splitWords(testCase.input); !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("splitWords(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestMatcherHitPhrase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantWord string
	}{
		{"camel token", "isSessionExpired", "session expired"},
		{"snake token", "draft_mode_on", "draft mode"},
		{"spaced token matches a camel name", "keepDraftMode", "draft mode"},
		{"words must be consecutive", "draftOtherMode", ""},
		{"words in the wrong order", "modeDraft", ""},
		{"one word of a phrase is not enough", "sessionKept", ""},
	}
	found := newMatcher([]string{"sessionExpired", "draft_mode", "draft mode"}, nil)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			word, ok := found.hit(testCase.input)
			if word != testCase.wantWord || ok != (testCase.wantWord != "") {
				t.Fatalf("hit(%q) = %q, %v", testCase.input, word, ok)
			}
		})
	}
}

func TestMatcherHitFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		base    string
		wantHit bool
	}{
		{"word in the stem", "draft_view.dart", true},
		{"extension is ignored", "view.dart", false},
		{"lock extension is ignored", "pubspec.lock", false},
		{"only the last extension is stripped", "draft.g.dart", true},
		{"dotfile keeps its full name", ".draft", true},
		{"allowed stem", "drafts.dart", false},
		{"allowed full name", "my_draft.dart", false},
	}
	found := newMatcher([]string{"draft", "dart", "lock"}, []string{"drafts", "my_draft.dart"})
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, ok := found.hitFile(testCase.base); ok != testCase.wantHit {
				t.Fatalf("hitFile(%q) = %v, want %v", testCase.base, ok, testCase.wantHit)
			}
		})
	}
}

func TestMatcherHit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantWord string
		wantHit  bool
	}{
		{"whole word", "draft", "draft", true},
		{"case is ignored", "DRAFT_copy", "draft", true},
		{"inside camel case", "probeSessionExpired", "expired", true},
		{"substring is not a word", "drafty", "", false},
		{"allowed name is exempt", "draftMode", "", false},
		{"no word", "preview", "", false},
	}
	found := newMatcher([]string{"Draft", "expired"}, []string{"draftMode"})
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			word, ok := found.hit(testCase.input)
			if word != testCase.wantWord || ok != testCase.wantHit {
				t.Fatalf("hit(%q) = %q, %v", testCase.input, word, ok)
			}
		})
	}
}
