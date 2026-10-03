package analysis

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func (sess *Session) emojiWindow() int {
	params := core.ParamSet(sess.cfgAnalysis.Analyzers[commit.AnalyzerID].Params)
	hard := params.Int(commit.ParamEmojiHardWindow, 0)
	// Fetch the wider window, so neither emoji rule sees fewer subjects than it counts.
	return max(hard, params.Int(commit.ParamEmojiSoftWindow, 0))
}

func (sess *Session) gitHistory() []core.GitCommitEntry {
	if sess.repo == nil {
		return nil
	}
	window := sess.emojiWindow()
	if sess.commitMsg != "" {
		// Dropping HEAD below would otherwise leave the widest window one subject short.
		window++
	}
	subjects, err := sess.repo.RecentSubjects(window)
	if err != nil {
		sess.logger.Warn("commit history unavailable", slog.Any("error", err))
		return nil
	}
	if len(subjects) == 0 {
		return nil
	}
	// On an amend or a rebase reword HEAD is the old commit; keeping it self-collides.
	if sess.replacesHead(subjects[0]) {
		subjects = subjects[1:]
	}

	entries := make([]core.GitCommitEntry, len(subjects))
	for idx, subject := range subjects {
		// RecentSubjects is newest-first; GitHistory reads the tail, so
		// reverse it into oldest-first here.
		entries[len(subjects)-1-idx] = core.GitCommitEntry{Subject: subject}
	}
	return entries
}

func (sess *Session) replacesHead(headSubject string) bool {
	if sess.commitMsg == "" {
		return false
	}
	subject, _, _ := strings.Cut(sess.commitMsg, "\n")
	if subject == headSubject {
		return true
	}
	// Subject alone misses an amend that also rewords it.
	return sess.authorDateMatchesHead()
}

func (sess *Session) authorDateMatchesHead() bool {
	// Git commit exports GIT_AUTHOR_DATE to its hooks; an amend or reword keeps HEAD's date.
	envEpoch, ok := parseGitAuthorDateEpoch(os.Getenv("GIT_AUTHOR_DATE"))
	if !ok {
		return false
	}
	headEpoch, hasHead, err := sess.repo.HeadAuthorEpoch()
	if err != nil {
		sess.logger.Warn("head author date unavailable", slog.Any("error", err))
		return false
	}
	if !hasHead {
		return false
	}
	// A fresh commit in HEAD's author second looks the same, so its repeat goes unseen.
	return envEpoch == headEpoch
}

func parseGitAuthorDateEpoch(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	// Git accepts "@<epoch> <tz>" or a plain epoch; only the epoch field matters.
	field, _, _ := strings.Cut(raw, " ")
	epoch, err := strconv.ParseInt(strings.TrimPrefix(field, "@"), 10, 64)
	if err != nil {
		return 0, false
	}
	return epoch, true
}
