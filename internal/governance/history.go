package governance

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func (sess *Session) gitHistory(commitMsg string) []pass.GitCommitEntry {
	if sess.repo == nil {
		return nil
	}
	window := message.HistoryWindow(
		pass.ParamSet(sess.cfgAnalysis.Analyzers[message.ID].Params),
	)
	if commitMsg != "" {
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
	if sess.replacesHead(commitMsg, subjects[0]) {
		subjects = subjects[1:]
	}

	entries := make([]pass.GitCommitEntry, len(subjects))
	for idx, subject := range subjects {
		// RecentSubjects is newest-first; GitHistory reads the tail, so
		// reverse it into oldest-first here.
		entries[len(subjects)-1-idx] = pass.GitCommitEntry{Subject: subject}
	}
	return entries
}

func (sess *Session) replacesHead(commitMsg, headSubject string) bool {
	if commitMsg == "" {
		return false
	}
	subject, _, _ := strings.Cut(commitMsg, "\n")
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
