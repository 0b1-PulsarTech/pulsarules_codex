package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestSessionGitHistoryDropsHeadOnReword(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)
	commitAllowEmpty(t, dir, ":wrench: feat: The tip")

	testCases := []struct {
		name       string
		commitMsg  string
		wantNewest string
	}{
		{"reword with the same subject drops HEAD", ":wrench: feat: The tip", ""},
		{
			"a different subject keeps HEAD",
			":wrench: feat: Something else",
			":wrench: feat: The tip",
		},
		{"an empty commit message keeps HEAD", "", ":wrench: feat: The tip"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo, err := vcs.Open(dir)
			if err != nil {
				t.Fatalf("vcs.Open: %v", err)
			}

			sess := NewSession(repo, testCase.commitMsg, nil, nil)
			entries := sess.gitHistory()
			if testCase.wantNewest == "" {
				if len(entries) != 0 {
					t.Fatalf("expected no history, got %+v", entries)
				}
				return
			}
			if len(entries) == 0 {
				t.Fatal("expected history, got none")
			}
			if got := entries[len(entries)-1].Subject; got != testCase.wantNewest {
				t.Fatalf("newest entry = %q, want %q", got, testCase.wantNewest)
			}
		})
	}
}

func TestSessionGitHistory_NilRepo(t *testing.T) {
	t.Parallel()

	sess := NewSession(nil, "anything", nil, nil)
	if entries := sess.gitHistory(); entries != nil {
		t.Fatalf("gitHistory() = %+v, want nil for a nil repo", entries)
	}
}

func TestSessionReplacesHead_AuthorDate(t *testing.T) {
	const headAuthorDate = "2024-01-01T00:00:00Z"
	const headAuthorEpoch = "1704067200"

	dir := t.TempDir()
	gitInitRepo(t, dir)
	commitAllowEmptyAt(t, dir, ":wrench: feat: The tip", headAuthorDate)

	testCases := []struct {
		name          string
		commitMsg     string
		gitAuthorDate string
		wantReplaces  bool
	}{
		{
			name:          "subject match alone drops HEAD",
			commitMsg:     ":wrench: feat: The tip",
			gitAuthorDate: "",
			wantReplaces:  true,
		},
		{
			name:          "author-date match with a changed subject drops HEAD",
			commitMsg:     ":wrench: feat: The tip, reworded",
			gitAuthorDate: "@" + headAuthorEpoch + " +0000",
			wantReplaces:  true,
		},
		{
			name:          "neither subject nor author date matching keeps HEAD",
			commitMsg:     ":wrench: feat: Something else entirely",
			gitAuthorDate: "@1600000000 +0000",
			wantReplaces:  false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("GIT_AUTHOR_DATE", testCase.gitAuthorDate)

			repo, err := vcs.Open(dir)
			if err != nil {
				t.Fatalf("vcs.Open: %v", err)
			}

			sess := NewSession(repo, testCase.commitMsg, nil, nil)
			entries := sess.gitHistory()
			gotReplaces := len(entries) == 0
			if gotReplaces != testCase.wantReplaces {
				t.Fatalf("replacesHead = %v, want %v (entries = %+v)",
					gotReplaces, testCase.wantReplaces, entries)
			}
		})
	}
}
