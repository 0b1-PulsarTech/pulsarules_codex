package governance

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
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

			sess := NewSession(repo, nil, testCatalog(t), nil)
			entries := sess.gitHistory(testCase.commitMsg)
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

	sess := NewSession(nil, nil, testCatalog(t), nil)
	if entries := sess.gitHistory("anything"); entries != nil {
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

			sess := NewSession(repo, nil, testCatalog(t), nil)
			entries := sess.gitHistory(testCase.commitMsg)
			gotReplaces := len(entries) == 0
			if gotReplaces != testCase.wantReplaces {
				t.Fatalf("replacesHead = %v, want %v (entries = %+v)",
					gotReplaces, testCase.wantReplaces, entries)
			}
		})
	}
}

type failingHistoryRepo struct {
	Repository
}

func (failingHistoryRepo) RecentSubjects(int) ([]string, error) {
	return nil, errors.New("log exploded")
}

func TestSessionGitHistory_LogsRecentSubjectsFailure(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	sess := NewSession(failingHistoryRepo{}, nil, testCatalog(t), nil)
	sess.logger = slog.New(slog.NewTextHandler(&logs, nil))

	if entries := sess.gitHistory(""); entries != nil {
		t.Fatalf("gitHistory() = %+v, want nil when the log fails", entries)
	}
	if !strings.Contains(logs.String(), "log exploded") {
		t.Fatalf("log output %q does not report the RecentSubjects error", logs.String())
	}
}

type failingEpochRepo struct {
	Repository
}

func (failingEpochRepo) HeadAuthorEpoch() (int64, bool, error) {
	return 0, false, errors.New("epoch exploded")
}

func TestSessionReplacesHead_LogsAuthorEpochFailure(t *testing.T) {
	t.Setenv("GIT_AUTHOR_DATE", "@1704067200 +0000")

	var logs bytes.Buffer
	sess := NewSession(failingEpochRepo{}, nil, testCatalog(t), nil)
	sess.logger = slog.New(slog.NewTextHandler(&logs, nil))

	if sess.replacesHead("reworded", "another subject") {
		t.Fatal("replacesHead = true, want false when the author epoch is unavailable")
	}
	if !strings.Contains(logs.String(), "epoch exploded") {
		t.Fatalf("log output %q does not report the HeadAuthorEpoch error", logs.String())
	}
}

type limitRecordingRepo struct {
	Repository
	limit int
}

func (repo *limitRecordingRepo) RecentSubjects(limit int) ([]string, error) {
	repo.limit = limit
	return nil, nil
}

func TestSessionGitHistory_FetchesTheWidestEmojiWindow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		hardWindow int
		softWindow int
		commitMsg  string
		want       int
	}{
		{"the defaults", 0, 0, "", 20},
		{"a soft window above twenty", 0, 30, "", 30},
		{"a hard window above the soft one", 25, 10, "", 25},
		{"a commit message fetches one more for the dropped HEAD", 1, 3, ":wrench: feat: Tip", 4},
		{"no commit message fetches the window itself", 1, 3, "", 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Defaults()
			if testCase.hardWindow > 0 {
				cfg.SetParam(message.ID, message.ParamEmojiHardWindow, testCase.hardWindow)
			}
			if testCase.softWindow > 0 {
				cfg.SetParam(message.ID, message.ParamEmojiSoftWindow, testCase.softWindow)
			}
			repo := &limitRecordingRepo{}
			NewSession(repo, nil, testCatalog(t), cfg).gitHistory(testCase.commitMsg)
			if repo.limit != testCase.want {
				t.Errorf("RecentSubjects limit = %d, want %d", repo.limit, testCase.want)
			}
		})
	}
}
